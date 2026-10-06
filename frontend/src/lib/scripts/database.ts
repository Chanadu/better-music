import { writable } from 'svelte/store';
import { albumsApi, ApiError, artistsApi } from './api';
import { cacheLibraryArtwork } from './artwork-cache';
import { getCurrentUserId, getValidAccessToken, hasStoredSession, invalidateSession } from './auth';
import {
	applyOptimisticMutation,
	getPendingMutations,
	getStoredDatabaseCache,
	setStoredDatabaseCache,
} from './database-cache';
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
	ServerDatabaseData,
} from './types';

export type { DatabaseData } from './types';
export type SyncStatus = {
	state: 'connecting' | 'synced' | 'pending' | 'offline' | 'error';
	lastSyncedAt: number | null;
};

export const database = writable<DatabaseData | null>(null);
export const pendingMutations = writable<PendingMutation[]>([]);
export const syncStatus = writable<SyncStatus>({ state: 'connecting', lastSyncedAt: null });
let current: { userId: number; data: DatabaseData } | null = null;
let request: { userId: number; promise: Promise<DatabaseData> } | null = null;
let lastRefreshStartedAt = 0;

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

const publishFresh = async (userId: number, data: ServerDatabaseData) => {
	if (getCurrentUserId() !== userId) return data;

	const saved = await setStoredDatabaseCache(userId, data);
	if (getCurrentUserId() !== userId) return data;

	if (!saved) {
		const cached = await getStoredDatabaseCache(userId);
		if (!cached) throw new Error('Could not load the library with pending changes');

		const mutations = await getPendingMutations(userId);
		if (getCurrentUserId() === userId) pendingMutations.set(mutations);
		return publishCached(userId, cached, true);
	}
	current = { userId, data };
	database.set(data);
	cacheLibraryArtwork(data);
	syncStatus.set({ state: 'synced', lastSyncedAt: data.loadedAt });

	return data;
};

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

export const refreshDatabaseData = async () => {
	const userId = getCurrentUserId();
	if (userId === null) throw new Error('Not authenticated');

	if (request?.userId === userId) return request.promise;

	lastRefreshStartedAt = Date.now();
	markConnecting();

	const promise = Promise.all([artistsApi.list(), albumsApi.list()])
		.then(([artists, albums]) => publishFresh(userId, { artists, albums, loadedAt: Date.now() }))
		.catch(async (error) => {
			if (error instanceof ApiError && error.status === 401) {
				await invalidateSession();
				location.assign('/login');
			} else {
				setSyncState(navigator.onLine ? 'error' : 'offline');
			}

			throw error;
		})
		.finally(() => {
			if (request?.promise === promise) request = null;
		});
	request = { userId, promise };

	return promise;
};

export const refreshStaleDatabaseData = async () => {
	const cached = await getDatabaseData();

	if (request) return request.promise;

	if (Date.now() - Math.max(cached?.loadedAt ?? 0, lastRefreshStartedAt) < 30000) {
		const userId = getCurrentUserId();

		if (cached && userId !== null && !(await getPendingMutations(userId)).length)
			syncStatus.set({ state: 'synced', lastSyncedAt: cached.loadedAt });

		return cached;
	}

	return refreshDatabaseData();
};

export const refreshDatabaseSafely = async (force = false) => {
	try {
		if (!navigator.onLine) {
			markOffline();
			return;
		}

		if (force) markConnecting();
		const token = await getValidAccessToken();
		if (!token) {
			if (!hasStoredSession()) location.assign('/login');
			else markSyncError();
			return;
		}

		if (force) await refreshDatabaseData();
		else await refreshStaleDatabaseData();
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
