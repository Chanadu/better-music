import { authenticatedFetch } from './auth';
import type {
	Album,
	Artist,
	AuthRequest,
	CreateAlbumRequest,
	CreateArtistRequest,
	MessageResponse,
	RefreshTokenRequest,
	SpotifyAlbumSearchResult,
	SpotifyArtistSearchResult,
	TokenResponse,
	UpdateAlbumRequest,
	UpdateArtistRequest,
} from './api-types';

type ApiRequestInit = Omit<RequestInit, 'body'> & {
	body?: unknown;
};

export class ApiError extends Error {
	status: number;
	body: unknown;

	constructor(message: string, status: number, body: unknown) {
		super(message);
		this.name = 'ApiError';
		this.status = status;
		this.body = body;
	}
}

const toQueryString = (params: Record<string, string | number | boolean | undefined>) => {
	const searchParams = new URLSearchParams();

	Object.entries(params).forEach(([key, value]) => {
		if (value === undefined) return;
		searchParams.set(key, String(value));
	});

	const queryString = searchParams.toString();
	return queryString ? `?${queryString}` : '';
};

const getErrorMessage = (body: unknown, fallback: string) => {
	if (!body || typeof body !== 'object') return fallback;

	const apiError = body as { error?: string; message?: string };
	return apiError.error ?? apiError.message ?? fallback;
};

const requestJson = async <T>(path: string, init: ApiRequestInit = {}, fetcher: typeof fetch = fetch) => {
	const headers = new Headers(init.headers);

	if (init.body !== undefined && !headers.has('Content-Type')) {
		headers.set('Content-Type', 'application/json');
	}

	const response = await fetcher(path, {
		...init,
		headers,
		body: init.body === undefined ? undefined : JSON.stringify(init.body),
	});

	const body = await response.json().catch(() => null);

	if (!response.ok) {
		throw new ApiError(getErrorMessage(body, `Request failed: ${response.status}`), response.status, body);
	}

	return body as T;
};

export const apiJson = <T>(path: string, init: ApiRequestInit = {}) => requestJson<T>(path, init, authenticatedFetch);

const publicApiJson = <T>(path: string, init: ApiRequestInit = {}) => requestJson<T>(path, init);

export const authApi = {
	register: (body: AuthRequest) =>
		publicApiJson<TokenResponse>('/api/auth/register', {
			method: 'POST',
			body,
		}),
	login: (body: AuthRequest) =>
		publicApiJson<TokenResponse>('/api/auth/login', {
			method: 'POST',
			body,
		}),
	logout: (body: RefreshTokenRequest) =>
		publicApiJson<MessageResponse>('/api/auth/logout', {
			method: 'POST',
			body,
		}),
};

export const artistsApi = {
	list: () => apiJson<Artist[]>('/api/artists'),
	get: (id: number) => apiJson<Artist>(`/api/artists/${id}`),
	create: (body: CreateArtistRequest) =>
		apiJson<Artist>('/api/artists', {
			method: 'POST',
			body,
		}),
	update: (id: number, body: UpdateArtistRequest) =>
		apiJson<MessageResponse>(`/api/artists/${id}`, {
			method: 'PUT',
			body,
		}),
	delete: (id: number) =>
		apiJson<MessageResponse>(`/api/artists/${id}`, {
			method: 'DELETE',
		}),
	albums: (id: number) => apiJson<Album[]>(`/api/artists/${id}/albums`),
};

export const albumsApi = {
	list: () => apiJson<Album[]>('/api/albums'),
	get: (id: number, artistId: number) =>
		apiJson<Album>(
			`/api/albums/${id}${toQueryString({
				artist_id: artistId,
			})}`,
		),
	create: (body: CreateAlbumRequest) =>
		apiJson<Album>('/api/albums', {
			method: 'POST',
			body,
		}),
	update: (id: number, body: UpdateAlbumRequest) =>
		apiJson<MessageResponse>(`/api/albums/${id}`, {
			method: 'PUT',
			body,
		}),
	delete: (id: number, artistId: number) =>
		apiJson<MessageResponse>(`/api/albums/${id}`, {
			method: 'DELETE',
			body: { artist_id: artistId },
		}),
};

export const spotifyApi = {
	searchArtists: (query: string, limit?: number) =>
		apiJson<SpotifyArtistSearchResult[]>(
			`/api/spotify/search/artists${toQueryString({
				q: query,
				limit,
			})}`,
		),
	searchAlbums: (query: string, limit?: number) =>
		apiJson<SpotifyAlbumSearchResult[]>(
			`/api/spotify/search/albums${toQueryString({
				q: query,
				limit,
			})}`,
		),
};
