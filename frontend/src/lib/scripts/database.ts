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
	getStoredLibrary,
	reconcileSuccessfulMutation,
	getSyncMetadata,
	markQueueCompletion,
	mergeLibraryRefresh,
	getStoredSyncState,
	LocalStorageError,
} from './database-cache';
import { withSyncOwnership, sendWithTimeout, type QueueOwnership } from './sync-coordination';
import { getMutationStatus, hasActiveQueueWork, hasUnknownOutcome } from './queue-state';
import { syncConfig } from './sync-config';
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
	lastQueueCompletedAt?: number | null;
	lastLibraryRefreshedAt?: number | null;
	pauseReason?: string;
	error?: string;
};

export const database = writable<DatabaseData | null>(null);
export const pendingMutations = writable<PendingMutation[]>([]);
export const syncStatus = writable<SyncStatus>({ state: 'connecting', lastSyncedAt: null });
export const storageProblem = writable<string | null>(null);
export const queueProcessorStatus = writable<SyncChange | null>(null);

let current: { userId: number; data: DatabaseData } | null = null;
type SyncRun = { userId: number; session: string | null; rerunRequested: boolean };
let request: { run: SyncRun; promise: Promise<DatabaseData> } | null = null;
let retryTimer: ReturnType<typeof setTimeout> | undefined;
let schedulingRevision = 0;

const setSyncState = (state: SyncStatus['state']) => syncStatus.update((status) => ({ ...status, state }));

export const markOffline = () => setSyncState('offline');
export const markSyncError = (reason = 'Synchronization could not complete. Try Sync now.') =>
	syncStatus.update((status) => ({ ...status, state: 'error', pauseReason: reason }));
export const markConnecting = () => setSyncState('connecting');

const publishCached = (userId: number, data: DatabaseData, hasPending = false) => {
	if (getCurrentUserId() !== userId) return data;

	current = { userId, data };
	database.set(data);
	cacheLibraryArtwork(data);
	syncStatus.update((previous) => ({
		...previous,
		state:
			navigator.onLine ?
				hasPending ? 'pending'
				:	'connecting'
			:	'offline',
	}));

	return data;
};

const publishStoredState = async (userId: number, stored: Awaited<ReturnType<typeof getStoredSyncState>>) => {
	const session = getSessionGeneration();
	const held = navigator.locks ? ((await navigator.locks.query()).held ?? []) : [];
	if (getCurrentUserId() !== userId || getSessionGeneration() !== session) return;
	const active =
		stored.metadata?.activity === 'syncing' && held.some((lock) => lock.name === syncConfig.lockPrefix + userId);
	const { library, mutations, metadata } = stored;
	current = library ? { userId, data: library } : null;
	database.set(library);
	pendingMutations.set(mutations);
	if (library) cacheLibraryArtwork(library);
	const interrupted = metadata?.activity === 'syncing' && !active;
	syncStatus.set({
		state:
			!navigator.onLine ? 'offline'
			: active ? 'connecting'
			: metadata?.activity === 'paused' || interrupted ? 'error'
			: hasActiveQueueWork(mutations) ? 'pending'
			: metadata?.lastLibraryRefreshedAt ? 'synced'
			: 'pending',
		lastSyncedAt: metadata?.lastSyncedAt ?? null,
		lastQueueCompletedAt: metadata?.lastQueueCompletedAt ?? null,
		lastLibraryRefreshedAt: metadata?.lastLibraryRefreshedAt ?? null,
		pauseReason:
			interrupted ?
				'A previous sync was interrupted. Reconcile any unknown request outcomes before recovery.'
			:	metadata?.pauseReason,
		error: metadata?.error,
	});
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
			const session = getSessionGeneration();
			if (userId === null) continue;
			const stored = await getStoredSyncState(userId);
			if (getCurrentUserId() === userId && getSessionGeneration() === session)
				await publishStoredState(userId, stored);
		}
	} catch (error) {
		if (error instanceof LocalStorageError) storageProblem.set(error.message);
		throw error;
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
		current = null;
		database.set(null);
		pendingMutations.set([]);
		syncStatus.set({ state: 'error', lastSyncedAt: null });
		return null;
	}
	if (current?.userId === userId) return current.data;
	try {
		const session = getSessionGeneration();
		const stored = await getStoredSyncState(userId);
		if (getCurrentUserId() !== userId || getSessionGeneration() !== session) return null;
		await publishStoredState(userId, stored);
		return stored.library;
	} catch (error) {
		if (error instanceof LocalStorageError) storageProblem.set(error.message);
		markSyncError();
		return null;
	}
};

export const retryLocalStorage = async () => {
	await reloadLocalState();
	storageProblem.set(null);
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
	if (queueResult.paused)
		throw new Error(
			{
				connection: 'Connection unavailable. Reconnect to resume.',
				authentication: 'Authentication unavailable. Sign in to the same account to resume.',
				storage:
					'Local storage could not commit the server result. Fix storage, then reconcile the original request.',
				account: 'The active account changed.',
				coordination: 'Synchronization was interrupted in another browser context.',
			}[queueResult.pauseReason ?? 'coordination'],
		);
	await markQueueCompletion(userId, generation, ownership.check);

	const baseline = await getStoredLibrary(userId);
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
	storageProblem.set(null);
	syncStatus.set({
		state: hasActiveQueueWork(merged.mutations) ? 'pending' : 'synced',
		lastSyncedAt: metadata.lastSyncedAt ?? null,
		lastQueueCompletedAt: metadata.lastQueueCompletedAt ?? null,
		lastLibraryRefreshedAt: metadata.lastLibraryRefreshedAt ?? null,
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
		if (error instanceof LocalStorageError) storageProblem.set(error.message);
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
			else markSyncError('Authentication unavailable. Sign in to the same account to resume.');
			return;
		}

		await refreshDatabaseData();
	} catch (error) {
		if (error instanceof LocalStorageError) storageProblem.set(error.message);
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
	paused: 2,
	blocked: 3,
	failed: 4,
	conflict: 5,
};

export const getRecordSyncStatus = (
	mutations: readonly PendingMutation[],
	userId: number,
	entity: MutationEntity,
	entityId: EntityId,
	syncState?: SyncStatus['state'],
): RecordSyncStatus => {
	let status: RecordSyncStatus = 'synced';

	for (const mutation of mutations) {
		if (mutation.userId !== userId || mutation.entity !== entity || mutation.entityId !== entityId) continue;
		let next: RecordSyncStatus = getMutationStatus(mutation, mutations);
		if (
			['pending', 'syncing'].includes(next) &&
			syncState !== 'connecting' &&
			(syncState === 'offline' || syncState === 'error' || hasUnknownOutcome(mutation))
		)
			next = 'paused';
		if (statusPriority[next] > statusPriority[status]) status = next;
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
			mutation.before = existing ? { ...existing } : null;

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
		mutation.before = existing ? { ...existing } : null;

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
	}).catch((error) => {
		if (error instanceof LocalStorageError) storageProblem.set(error.message);
		throw error;
	});
	storageProblem.set(null);

	if (getCurrentUserId() === mutation.userId) {
		current = { userId: mutation.userId, data: result.library };
		database.set(result.library);
		pendingMutations.set(result.mutations);
		setSyncState(navigator.onLine ? 'pending' : 'offline');
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
