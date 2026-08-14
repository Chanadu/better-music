import { writable } from 'svelte/store';
import { albumsApi, ApiError, artistsApi } from './api';
import { clearTokens } from './auth';
import type { Album, Artist } from './types';

export type DatabaseData = { artists: Artist[]; albums: Album[]; loadedAt: number };
export const database = writable<DatabaseData | null>(null);
const cacheKey = 'betterMusicDatabaseData';
let current: DatabaseData | null = null;
let request: Promise<DatabaseData> | null = null;
let lastRefreshStartedAt = 0;

const publish = (data: DatabaseData) => {
	current = data;
	sessionStorage.setItem(cacheKey, JSON.stringify(data));
	database.set(data);

	return data;
};

export const loadCachedDatabase = () => {
	if (current) return current;

	const raw = sessionStorage.getItem(cacheKey);

	if (!raw) return null;

	try {
		const value = JSON.parse(raw) as DatabaseData;

		if (!Array.isArray(value.artists) || !Array.isArray(value.albums)) throw new Error();

		current = value;
		database.set(value);

		return value;
	} catch {
		sessionStorage.removeItem(cacheKey);

		return null;
	}
};

export const fetchDatabaseData = async ({ force = false } = {}) => {
	const cached = loadCachedDatabase();

	if (!force && cached) return cached;
	if (!force && request) return request;

	lastRefreshStartedAt = Date.now();

	request = Promise.all([artistsApi.list(), albumsApi.list()])
		.then(([artists, albums]) => publish({ artists, albums, loadedAt: Date.now() }))
		.catch((error) => {
			if (error instanceof ApiError && error.status === 401) {
				clearTokens();
				location.assign('/login');
			}

			throw error;
		})
		.finally(() => {
			request = null;
		});

	return request;
};
export const refreshDatabaseData = () => fetchDatabaseData({ force: true });

export const refreshStaleDatabaseData = () => {
	const cached = loadCachedDatabase();

	if (request || Date.now() - Math.max(cached?.loadedAt ?? 0, lastRefreshStartedAt) < 30_000)
		return Promise.resolve(cached);

	return refreshDatabaseData();
};
