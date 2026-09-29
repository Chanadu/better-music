import { writable } from 'svelte/store';
import { albumsApi, ApiError, artistsApi } from './api';
import { clearTokens, getCurrentUserId } from './auth';
import { getStoredDatabaseCache, setStoredDatabaseCache } from './database-cache';
import type { Album, Artist } from './types';

export type DatabaseData = { artists: Artist[]; albums: Album[]; loadedAt: number };
export const database = writable<DatabaseData | null>(null);
let current: { userId: number; data: DatabaseData } | null = null;
let request: { userId: number; promise: Promise<DatabaseData> } | null = null;
let lastRefreshStartedAt = 0;

const publish = (userId: number, data: DatabaseData) => {
	if (getCurrentUserId() !== userId) return data;

	current = { userId, data };
	void setStoredDatabaseCache(userId, data);
	database.set(data);

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

	if (!value) return null;

	if (Array.isArray(value.artists) && Array.isArray(value.albums)) {
		current = { userId, data: value };
		database.set(value);

		return value;
	}

	return null;
};

export const fetchDatabaseData = async ({ force = false } = {}) => {
	const userId = getCurrentUserId();
	if (userId === null) throw new Error('Not authenticated');

	const cached = await loadCachedDatabase();

	if (!force && cached) return cached;
	if (!force && request?.userId === userId) return request.promise;

	lastRefreshStartedAt = Date.now();

	const promise = Promise.all([artistsApi.list(), albumsApi.list()])
		.then(([artists, albums]) => publish(userId, { artists, albums, loadedAt: Date.now() }))
		.catch((error) => {
			if (error instanceof ApiError && error.status === 401) {
				clearTokens();
				location.assign('/login');
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
	if (request || Date.now() - Math.max(cached?.loadedAt ?? 0, lastRefreshStartedAt) < 30000) return cached;
	return refreshDatabaseData();
};
