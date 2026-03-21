import { apiFetch, getAccessToken, getRefreshToken } from './api';

export type LibraryData = {
	albums: unknown[];
	artists: unknown[];
};

let cachedLibraryData: LibraryData | null = null;
let libraryRequest: Promise<LibraryData> | null = null;

const hasAuthTokens = () => Boolean(getAccessToken() || getRefreshToken());

const fetchLibraryData = async (): Promise<LibraryData> => {
	const [albumsRes, artistsRes] = await Promise.all([
		apiFetch('/api/albums', { method: 'GET' }, { auth: true }),
		apiFetch('/api/artists', { method: 'GET' }, { auth: true }),
	]);

	if (!albumsRes.ok || !artistsRes.ok) {
		throw new Error('Failed to load library data.');
	}

	const [albums, artists] = await Promise.all([
		albumsRes.json() as Promise<unknown[]>,
		artistsRes.json() as Promise<unknown[]>,
	]);

	const nextData = {
		albums,
		artists,
	};
	cachedLibraryData = nextData;
	return nextData;
};

export const loadLibraryData = async (force = false): Promise<LibraryData> => {
	if (!force && cachedLibraryData) return cachedLibraryData;
	if (libraryRequest) return libraryRequest;

	libraryRequest = fetchLibraryData().finally(() => {
		libraryRequest = null;
	});

	return libraryRequest;
};

export const warmLibraryData = (): void => {
	if (cachedLibraryData || libraryRequest || !hasAuthTokens()) return;
	void loadLibraryData().catch(() => undefined);
};

export const invalidateLibraryData = (): void => {
	cachedLibraryData = null;
};
