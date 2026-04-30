import { authenticatedFetch, clearTokens } from './auth';

export type Artist = {
	id: number;
	name: string;
	cover_url?: string;
	spotify_id?: string;
	created_at: string;
};

export type Album = {
	id: number;
	artist_id: number;
	title: string;
	cover_url?: string;
	year?: number;
	spotify_id?: string;
	listened: boolean;
	rating?: number;
	comment?: string;
	listened_at?: string;
	created_at: string;
};

export type DatabaseData = {
	artists: Artist[];
	albums: Album[];
	loadedAt: number;
};

export type DatabaseDataEvent = CustomEvent<DatabaseData>;

const databaseDataEventName = 'better-music:database-data';
const databaseErrorEventName = 'better-music:database-error';
const databaseCacheKey = 'betterMusicDatabaseData';

let databaseData: DatabaseData | null = null;
let databaseDataPromise: Promise<DatabaseData> | null = null;

const readJson = async <T>(path: string): Promise<T> => {
	const response = await authenticatedFetch(path);

	if (response.status === 401) {
		clearTokens();
		window.location.assign('/login');
		throw new Error('Not authenticated');
	}

	if (!response.ok) {
		throw new Error(`Database request failed: ${response.status}`);
	}

	return (await response.json()) as T;
};

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

	databaseDataPromise = Promise.all([readJson<Artist[]>('/api/artists'), readJson<Album[]>('/api/albums')])
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
			emitDatabaseError(error);
			throw error;
		})
		.finally(() => {
			databaseDataPromise = null;
		});

	return databaseDataPromise;
};

export const refreshDatabaseData = () => fetchDatabaseData({ force: true });

export { databaseDataEventName, databaseErrorEventName };
