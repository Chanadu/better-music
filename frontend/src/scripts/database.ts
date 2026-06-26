import { albumsApi, ApiError, artistsApi } from './api';
import { redirectToLogin } from './auth';
import type { Album, Artist } from './api-types';

export type { Album, Artist } from './api-types';

export type DatabaseData = {
	artists: Artist[];
	albums: Album[];
	loadedAt: number;
};

export type DatabaseDataEvent = CustomEvent<DatabaseData>;

const databaseDataEventName = 'better-music:database-data';
const databaseErrorEventName = 'better-music:database-error';
const databaseCacheKey = 'betterMusicDatabaseData';
const defaultStaleAfterMs = 30_000;

let databaseData: DatabaseData | null = null;
let databaseDataPromise: Promise<DatabaseData> | null = null;
let lastRefreshStartedAt = 0;

const saveCachedDatabaseData = (data: DatabaseData) => {
	sessionStorage.setItem(databaseCacheKey, JSON.stringify(data));
};

const readCachedDatabaseData = () => {
	if (databaseData) return databaseData;

	const cachedData = sessionStorage.getItem(databaseCacheKey);
	if (!cachedData) return null;

	try {
		databaseData = JSON.parse(cachedData) as DatabaseData;
		return databaseData;
	} catch {
		sessionStorage.removeItem(databaseCacheKey);
		return null;
	}
};

const emitDatabaseData = (data: DatabaseData) => {
	window.dispatchEvent(new CustomEvent(databaseDataEventName, { detail: data }));
};

const emitDatabaseError = (error: unknown) => {
	window.dispatchEvent(new CustomEvent(databaseErrorEventName, { detail: error }));
};

export const getCachedDatabaseData = () => readCachedDatabaseData();

export const onDatabaseData = (callback: (data: DatabaseData) => void) => {
	const cachedData = readCachedDatabaseData();
	if (cachedData) callback(cachedData);

	const listener = (event: Event) => {
		callback((event as DatabaseDataEvent).detail);
	};

	window.addEventListener(databaseDataEventName, listener);

	return () => window.removeEventListener(databaseDataEventName, listener);
};

export const fetchDatabaseData = async ({ force = false } = {}) => {
	if (!force) {
		const cachedData = readCachedDatabaseData();
		if (cachedData) return cachedData;
	}

	if (!force && databaseDataPromise) return databaseDataPromise;

	lastRefreshStartedAt = Date.now();

	databaseDataPromise = Promise.all([artistsApi.list(), albumsApi.list()])
		.then(([artists, albums]) => {
			databaseData = {
				artists,
				albums,
				loadedAt: Date.now(),
			};

			saveCachedDatabaseData(databaseData);
			emitDatabaseData(databaseData);

			return databaseData;
		})
		.catch((error) => {
			if (error instanceof ApiError && error.status === 401) {
				redirectToLogin();
			}

			emitDatabaseError(error);
			throw error;
		})
		.finally(() => {
			databaseDataPromise = null;
		});

	return databaseDataPromise;
};

export const refreshDatabaseData = () => fetchDatabaseData({ force: true });

export const refreshStaleDatabaseData = ({ staleAfterMs = defaultStaleAfterMs } = {}) => {
	const cachedData = readCachedDatabaseData();
	const now = Date.now();
	const lastLoadedAt = cachedData?.loadedAt ?? 0;
	const lastSyncAt = Math.max(lastLoadedAt, lastRefreshStartedAt);

	if (databaseDataPromise || now - lastSyncAt < staleAfterMs) {
		return Promise.resolve(cachedData);
	}

	return refreshDatabaseData();
};

export { databaseDataEventName, databaseErrorEventName };
