import { writable } from 'svelte/store';
import { albumsApi, ApiError, artistsApi } from './api';
import { cacheLibraryArtwork } from './artwork-cache';
import {
	getCurrentUserId,
	getSessionGeneration,
	getValidAccessToken,
	hasStoredSession,
	invalidateSession,
} from './auth';
import {
	applyOptimisticMutation,
	getPendingMutations,
	getStoredDatabaseCache,
	getStoredLibraryUpdate,
	reconcileSuccessfulMutation,
	getSyncMetadata,
	markQueueCompletion,
	mergeLibraryRefresh,
} from './database-cache';
import { withSyncOwnership, sendWithTimeout, type QueueOwnership } from './sync-coordination';
import { subscribeSyncChanges, type SyncChange } from './sync-events';
import type {
	DatabaseData,
	EntityId,
	MutationEntity,
	PendingMutation,
	RecordSyncStatus,
	TemporaryId,
	Artist,
	Album,
	CreateArtistRequest,
	UpdateArtistRequest,
	CreateAlbumMutationPayload,
	UpdateAlbumMutationPayload,
	SuccessfulMutationResponse,
} from './types';

export type { DatabaseData } from './types';
export type SyncStatus = {
	state: 'connecting' | 'synced' | 'pending' | 'offline' | 'error';
	lastSyncedAt: number | null;
};

export const database = writable<DatabaseData | null>(null);
export const pendingMutations = writable<PendingMutation[]>([]);
export const syncStatus = writable<SyncStatus>({ state: 'connecting', lastSyncedAt: null });
export const queueProcessorStatus = writable<SyncChange | null>(null);

let current: { userId: number; data: DatabaseData } | null = null;
type SyncRun = { userId: number; session: string | null; rerunRequested: boolean };
let request: { run: SyncRun; promise: Promise<DatabaseData> } | null = null;
let retryTimer: ReturnType<typeof setTimeout> | undefined;
let schedulingRevision = 0;

const setSyncState = (state: SyncStatus['state']) => syncStatus.update((status) => ({ ...status, state }));

export const markOffline = () => setSyncState('offline');
export const markSyncError = () => setSyncState('error');
export const markConnecting = () => setSyncState('connecting');

const publishCached = (userId: number, data: DatabaseData, hasPending = false) => {
	if (getCurrentUserId() !== userId) return data;

	current = { userId, data };
	database.set(data);
	cacheLibraryArtwork(data);
	syncStatus.set({
		state:
			navigator.onLine ?
				hasPending ? 'pending'
				:	'connecting'
			:	'offline',
		lastSyncedAt: data.loadedAt || null,
	});

	return data;
};

let reloadingLocalState = false;
let localStateChanged = false;

const reloadLocalState = async () => {
	localStateChanged = true;
	if (reloadingLocalState) return;
	reloadingLocalState = true;
	try {
		while (localStateChanged) {
			localStateChanged = false;
			const userId = getCurrentUserId();

			if (userId === null) continue;

			const stored = await getStoredLibraryUpdate(userId);
			if (getCurrentUserId() !== userId) continue;

			pendingMutations.set(stored?.mutations ?? []);

			if (stored) {
				current = { userId, data: stored.library };
				database.set(stored.library);

				cacheLibraryArtwork(stored.library);

				if (stored.mutations.length > 0) setSyncState(navigator.onLine ? 'pending' : 'offline');
			} else {
				current = null;
				database.set(null);
			}
		}
	} finally {
		reloadingLocalState = false;
	}
};

subscribeSyncChanges((change) => {
	if (change.userId !== getCurrentUserId()) return;

	if (change.type === 'processor') {
		queueProcessorStatus.update((current) => {
			if (change.state !== 'syncing' && current?.ownerId && current.ownerId !== change.ownerId) return current;
			return change;
		});
	}

	if (change.type === 'queue') void refreshDatabaseSafely();
	if (change.type === 'library') void scheduleRetry().catch(console.error);

	void reloadLocalState().catch((error) => console.warn('Could not reload changed local library', error));
});

export const getDatabaseData = async () => {
	const userId = getCurrentUserId();

	if (userId === null) {
		if (current) database.set(null);
		current = null;
		pendingMutations.set([]);
		return null;
	}

	if (current?.userId === userId) return current.data;

	if (current) database.set(null);
	current = null;
	const value = await getStoredDatabaseCache(userId);

	if (!value || !Array.isArray(value.artists) || !Array.isArray(value.albums)) return null;

	const mutations = await getPendingMutations(userId);
	if (getCurrentUserId() === userId) pendingMutations.set(mutations);

	return publishCached(userId, value, mutations.length > 0);
};

const scheduleRetry = async () => {
	const revision = ++schedulingRevision;

	clearTimeout(retryTimer);

	const userId = getCurrentUserId();

	if (userId === null || typeof navigator === 'undefined') return;

	const { getQueueWork } = await import('./queue-processor');
	const queue = await getPendingMutations(userId);

	if (revision !== schedulingRevision || getCurrentUserId() !== userId) return;

	const { retryAt } = getQueueWork(queue);
	if (retryAt === null || !navigator.onLine) return;
	const remaining = retryAt - Date.now();
	const delay = remaining > 0 ? remaining : 1000;

	retryTimer = setTimeout(() => void refreshDatabaseSafely(), Math.min(delay, 2_147_483_647));
};

const syncPass = async (userId: number, generation: string, ownership: QueueOwnership) => {
	const { processMutationQueue } = await import('./queue-processor');
	ownership.check();
	if (!(await getValidAccessToken())) throw new Error('Authentication is unavailable');
	ownership.check();
	const queueResult = await processMutationQueue(ownership);
	if (queueResult.paused) throw new Error(`Synchronization paused: ${queueResult.pauseReason}`);
	await markQueueCompletion(userId, generation, ownership.check);

	const baseline = await getStoredDatabaseCache(userId);
	const [artists, albums] = await sendWithTimeout(ownership, (signal) =>
		Promise.all([artistsApi.list(signal), albumsApi.list(signal)]),
	);
	const merged = await mergeLibraryRefresh(
		userId,
		{ artists, albums, loadedAt: Date.now() },
		baseline,
		generation,
		ownership.check,
	);
	const metadata = await getSyncMetadata(userId);
	ownership.check();

	current = { userId, data: merged.library };
	database.set(merged.library);
	pendingMutations.set(merged.mutations);
	cacheLibraryArtwork(merged.library);
	syncStatus.set({
		state:
			merged.mutations.some((item) => item.status === 'pending' || item.status === 'syncing') ?
				'pending'
			:	'synced',
		lastSyncedAt: metadata.lastSyncedAt ?? null,
	});
	return merged.library;
};

const synchronize = async (run: SyncRun): Promise<DatabaseData> => {
	const { userId, session } = run;
	const sameSession = () => getCurrentUserId() === userId && getSessionGeneration() === session;
	const canContinue = () => sameSession() && navigator.onLine;
	markConnecting();
	try {
		const result = await withSyncOwnership(userId, canContinue, async (ownership) => {
			const { generation } = await getSyncMetadata(userId);
			let data: DatabaseData;
			do {
				run.rerunRequested = false;
				data = await syncPass(userId, generation, ownership);
			} while (run.rerunRequested);
			return { paused: false, data };
		});
		if (!result) throw new Error('Synchronization interrupted');
		return result.data;
	} catch (error) {
		if (sameSession()) {
			if (error instanceof ApiError && error.status === 401) await invalidateSession();
			setSyncState(navigator.onLine ? 'error' : 'offline');
		}
		throw error;
	} finally {
		if (request?.run === run) {
			request = null;
			if (run.rerunRequested && canContinue()) void refreshDatabaseSafely();
		}
		void scheduleRetry().catch(console.error);
	}
};

export const refreshDatabaseData = (): Promise<DatabaseData> => {
	const userId = getCurrentUserId();
	if (userId === null) return Promise.reject(new Error('Not authenticated'));
	if (request?.run.userId === userId && request.run.session === getSessionGeneration()) {
		request.run.rerunRequested = true;
		return request.promise;
	}
	const run: SyncRun = { userId, session: getSessionGeneration(), rerunRequested: false };
	const promise = synchronize(run);
	request = { run, promise };
	return promise;
};

export const refreshDatabaseSafely = async () => {
	try {
		if (!navigator.onLine) {
			markOffline();
			return;
		}

		const token = await getValidAccessToken();

		if (!token) {
			if (!hasStoredSession()) location.assign('/login');
			else markSyncError();
			return;
		}

		await refreshDatabaseData();
	} catch (error) {
		setSyncState(navigator.onLine ? 'error' : 'offline');
		console.error('Failed to refresh database data', error);
	}
};

export const createTemporaryId = (): TemporaryId => `local:${crypto.randomUUID()}`;

export const isTemporaryId = (id: EntityId): id is TemporaryId => typeof id === 'string' && id.startsWith('local:');

const statusPriority: Record<RecordSyncStatus, number> = {
	synced: 0,
	pending: 1,
	syncing: 2,
	failed: 3,
	conflict: 4,
};

export const getRecordSyncStatus = (
	mutations: readonly PendingMutation[],
	userId: number,
	entity: MutationEntity,
	entityId: EntityId,
): RecordSyncStatus => {
	let status: RecordSyncStatus = 'synced';

	for (const mutation of mutations) {
		if (mutation.userId !== userId || mutation.entity !== entity || mutation.entityId !== entityId) continue;
		if (statusPriority[mutation.status] > statusPriority[status]) status = mutation.status;
	}

	return status;
};

export const parseEntityId = (value: string | null): EntityId | null => {
	if (value?.startsWith('local:')) return value as TemporaryId;
	const id = Number(value);

	return Number.isInteger(id) && id > 0 ? id : null;
};

export const reconcileMutationResponse = async (
	userId: number,
	mutationId: string,
	response: SuccessfulMutationResponse,
	checkOwnership?: () => void,
) => {
	const result = await reconcileSuccessfulMutation(userId, mutationId, response, checkOwnership);

	if (result && getCurrentUserId() === userId) {
		pendingMutations.set(
			[...result.mutations].sort(
				(left, right) => (left.sequence ?? left.createdAt) - (right.sequence ?? right.createdAt),
			),
		);

		publishCached(userId, result.library, result.mutations.length > 0);
	}

	return result;
};

const newMutation = () => {
	const userId = getCurrentUserId();
	if (userId === null) throw new Error('Not authenticated');

	return { mutationId: crypto.randomUUID(), userId, createdAt: Date.now(), attempts: 0, status: 'pending' as const };
};

const requiredName = (value: string, label: string) => {
	const name = value.trim();
	if (!name) throw new Error(`${label} is required`);

	return name;
};

const definedFields = <T extends object>(value: T): T =>
	Object.fromEntries(Object.entries(value).filter(([, item]) => item !== undefined)) as T;

const queueMutation = async (mutation: PendingMutation) => {
	const result = await applyOptimisticMutation(mutation.userId, mutation, (library) => {
		if (getCurrentUserId() !== mutation.userId) throw new Error('Account changed before the change was saved');

		if (mutation.entity === 'artist') {
			const existing = library.artists.find((item) => item.id === mutation.entityId);
			mutation.baseVersion = existing?.version ?? undefined;

			if (mutation.operation === 'delete') {
				if (library.albums.some((item) => item.artist_id === mutation.entityId))
					throw new Error('Remove this artist’s albums first');
				return { ...library, artists: library.artists.filter((item) => item.id !== mutation.entityId) };
			}

			if (mutation.operation === 'update' && !existing) throw new Error('Artist not found');

			const payload = definedFields(mutation.payload);
			if (!Object.keys(payload).length) throw new Error('At least one field must be provided');

			const name = requiredName(payload.name ?? existing?.name ?? '', 'Name');
			if (
				library.artists.some(
					(item) => item.id !== mutation.entityId && item.name.toLowerCase() === name.toLowerCase(),
				)
			)
				throw new Error('Artist with this name already exists');

			const record: Artist =
				mutation.operation === 'create' ?
					{
						...payload,
						name,
						id: mutation.entityId as TemporaryId,
						version: null,
						created_at: new Date(mutation.createdAt).toISOString(),
					}
				:	{ ...existing!, ...payload, name };
			return {
				...library,
				artists:
					mutation.operation === 'create' ?
						[record, ...library.artists]
					:	library.artists.map((item) => (item.id === record.id ? record : item)),
			};
		}

		const existing = library.albums.find((item) => item.id === mutation.entityId);

		mutation.baseVersion = existing?.version ?? undefined;

		if (mutation.operation === 'delete')
			return { ...library, albums: library.albums.filter((item) => item.id !== mutation.entityId) };

		if (mutation.operation === 'update' && !existing) throw new Error('Album not found');

		const payload = definedFields(mutation.payload);

		if (mutation.operation === 'update' && !Object.keys(payload).some((field) => field !== 'artist_id'))
			throw new Error('At least one field must be provided');

		if (!library.artists.some((item) => item.id === payload.artist_id)) throw new Error('Artist not found');

		if (existing && existing.artist_id !== payload.artist_id) throw new Error('Cannot change an album’s artist');

		const title = requiredName(payload.title ?? existing?.title ?? '', 'Title');

		if (payload.rating != null && (!Number.isInteger(payload.rating) || payload.rating < 1 || payload.rating > 10))
			throw new Error('Rating must be between 1 and 10');

		if (payload.year != null && !Number.isInteger(payload.year)) throw new Error('Year must be an integer');

		if (
			library.albums.some(
				(item) =>
					item.id !== mutation.entityId &&
					item.artist_id === payload.artist_id &&
					item.title.toLowerCase() === title.toLowerCase(),
			)
		)
			throw new Error('Album with this title already exists for this artist');

		const record: Album =
			mutation.operation === 'create' ?
				{
					...payload,
					title,
					id: mutation.entityId as TemporaryId,
					version: null,
					listened: payload.listened ?? false,
					created_at: new Date(mutation.createdAt).toISOString(),
				}
			:	{ ...existing!, ...payload, title };

		if (payload.listened === false) {
			record.rating = null;
			record.listened_at = null;
		}

		return {
			...library,
			albums:
				mutation.operation === 'create' ?
					[record, ...library.albums]
				:	library.albums.map((item) => (item.id === record.id ? record : item)),
		};
	});

	if (getCurrentUserId() === mutation.userId) {
		current = { userId: mutation.userId, data: result.library };
		database.set(result.library);
		pendingMutations.set(result.mutations);
		syncStatus.set({
			state: navigator.onLine ? 'pending' : 'offline',
			lastSyncedAt: result.library.loadedAt || null,
		});
		cacheLibraryArtwork(result.library);
	}

	return result.library;
};

export const artistsLibrary = {
	create: async (payload: CreateArtistRequest) => {
		const entityId = createTemporaryId();
		const library = await queueMutation({
			...newMutation(),
			entity: 'artist',
			operation: 'create',
			entityId,
			payload: definedFields(payload),
		});

		return library.artists.find((item) => item.id === entityId)!;
	},
	update: async (entityId: EntityId, payload: UpdateArtistRequest) => {
		const library = await queueMutation({
			...newMutation(),
			entity: 'artist',
			operation: 'update',
			entityId,
			payload: definedFields(payload),
		});

		return library.artists.find((item) => item.id === entityId)!;
	},

	delete: async (entityId: EntityId) => {
		await queueMutation({ ...newMutation(), entity: 'artist', operation: 'delete', entityId, payload: null });
	},
};

export const albumsLibrary = {
	create: async (payload: CreateAlbumMutationPayload) => {
		const entityId = createTemporaryId();
		const library = await queueMutation({
			...newMutation(),
			entity: 'album',
			operation: 'create',
			entityId,
			payload: definedFields(payload),
		});
		return library.albums.find((item) => item.id === entityId)!;
	},

	update: async (entityId: EntityId, payload: UpdateAlbumMutationPayload) => {
		const library = await queueMutation({
			...newMutation(),
			entity: 'album',
			operation: 'update',
			entityId,
			payload: definedFields(payload),
		});
		return library.albums.find((item) => item.id === entityId)!;
	},

	delete: async (entityId: EntityId, artistId: EntityId) => {
		await queueMutation({
			...newMutation(),
			entity: 'album',
			operation: 'delete',
			entityId,
			payload: { artist_id: artistId },
		});
	},
};
