import { writable } from 'svelte/store';
import { albumsApi, ApiError, artistsApi } from './api';
import { cacheLibraryArtwork } from './artwork-cache';
import { clearTokens, getCurrentUserId } from './auth';
import { getStoredDatabaseCache, setStoredDatabaseCache } from './database-cache';
import type { DatabaseData } from './types';

export type { DatabaseData } from './types';
export type SyncStatus = {
	state: 'connecting' | 'synced' | 'offline' | 'error';
	lastSyncedAt: number | null;
};

export const database = writable<DatabaseData | null>(null);
export const syncStatus = writable<SyncStatus>({ state: 'connecting', lastSyncedAt: null });
let current: { userId: number; data: DatabaseData } | null = null;
let request: { userId: number; promise: Promise<DatabaseData> } | null = null;
let lastRefreshStartedAt = 0;

const setSyncState = (state: SyncStatus['state']) => syncStatus.update((status) => ({ ...status, state }));

export const markOffline = () => setSyncState('offline');
export const markSyncError = () => setSyncState('error');
export const markConnecting = () => setSyncState('connecting');

const publishCached = (userId: number, data: DatabaseData) => {
	if (getCurrentUserId() !== userId) return data;

	current = { userId, data };
	database.set(data);
	cacheLibraryArtwork(data);
	syncStatus.set({
		state: navigator.onLine ? 'connecting' : 'offline',
		lastSyncedAt: data.loadedAt,
	});

	return data;
};

const publishFresh = (userId: number, data: DatabaseData) => {
	if (getCurrentUserId() !== userId) return data;

	current = { userId, data };
	void setStoredDatabaseCache(userId, data);
	database.set(data);
	cacheLibraryArtwork(data);
	syncStatus.set({ state: 'synced', lastSyncedAt: data.loadedAt });

	return data;
};

export const loadCachedDatabase = async () => {
	const userId = getCurrentUserId();
	if (userId === null) {
		if (current) database.set(null);
		current = null;
		return null;
	}
	if (current?.userId === userId) return current.data;

	if (current) database.set(null);
	current = null;
	const value = await getStoredDatabaseCache(userId);

	if (!value || !Array.isArray(value.artists) || !Array.isArray(value.albums)) return null;

	return publishCached(userId, value);
};

export const fetchDatabaseData = async ({ force = false } = {}) => {
	const userId = getCurrentUserId();
	if (userId === null) throw new Error('Not authenticated');

	const cached = await loadCachedDatabase();

	if (!force && cached) return cached;
	if (request?.userId === userId) return request.promise;

	lastRefreshStartedAt = Date.now();
	markConnecting();

	const promise = Promise.all([artistsApi.list(), albumsApi.list()])
		.then(([artists, albums]) => publishFresh(userId, { artists, albums, loadedAt: Date.now() }))
		.catch((error) => {
			if (error instanceof ApiError && error.status === 401) {
				clearTokens();
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
export const refreshDatabaseData = () => fetchDatabaseData({ force: true });

export const refreshStaleDatabaseData = async () => {
	const cached = await loadCachedDatabase();
	if (request) return request.promise;
	if (Date.now() - Math.max(cached?.loadedAt ?? 0, lastRefreshStartedAt) < 30000) {
		if (cached) syncStatus.set({ state: 'synced', lastSyncedAt: cached.loadedAt });
		return cached;
	}
	return refreshDatabaseData();
};
