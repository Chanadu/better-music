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

export class ApiError extends Error {
	constructor(
		message: string,
		public status: number,
		public body: unknown,
	) {
		super(message);
		this.name = 'ApiError';
	}
}

type JsonInit = Omit<RequestInit, 'body'> & { body?: unknown };

const json = async <T>(path: string, init: JsonInit = {}, fetcher: typeof fetch = fetch): Promise<T> => {
	const headers = new Headers(init.headers);
	if (init.body !== undefined) headers.set('Content-Type', 'application/json');
	const response = await fetcher(path, {
		...init,
		headers,
		body: init.body === undefined ? undefined : JSON.stringify(init.body),
	});
	const body = await response.json().catch(() => null);
	if (!response.ok) {
		const candidate = body as { error?: string; message?: string } | null;
		throw new ApiError(
			candidate?.error ?? candidate?.message ?? `Request failed: ${response.status}`,
			response.status,
			body,
		);
	}
	return body as T;
};

const secureJson = <T>(path: string, init: JsonInit = {}) => json<T>(path, init, authenticatedFetch);
const query = (values: Record<string, string | number | undefined>) => {
	const params = new URLSearchParams();
	Object.entries(values).forEach(([key, value]) => value !== undefined && params.set(key, String(value)));
	return params.size ? `?${params}` : '';
};

export const authApi = {
	login: (body: AuthRequest) => json<TokenResponse>('/api/auth/login', { method: 'POST', body }),
	register: (body: AuthRequest) => json<TokenResponse>('/api/auth/register', { method: 'POST', body }),
	refresh: (body: RefreshTokenRequest) => json<TokenResponse>('/api/auth/refresh', { method: 'POST', body }),
	logout: (body: RefreshTokenRequest) => json<MessageResponse>('/api/auth/logout', { method: 'POST', body }),
};

export const artistsApi = {
	list: () => secureJson<Artist[]>('/api/artists'),
	get: (id: number) => secureJson<Artist>(`/api/artists/${id}`),
	create: (body: CreateArtistRequest) => secureJson<Artist>('/api/artists', { method: 'POST', body }),
	update: (id: number, body: UpdateArtistRequest) =>
		secureJson<MessageResponse>(`/api/artists/${id}`, { method: 'PUT', body }),
	delete: (id: number) => secureJson<MessageResponse>(`/api/artists/${id}`, { method: 'DELETE' }),
	albums: (id: number) => secureJson<Album[]>(`/api/artists/${id}/albums`),
};

export const albumsApi = {
	list: () => secureJson<Album[]>('/api/albums'),
	get: (id: number, artistId: number) => secureJson<Album>(`/api/albums/${id}${query({ artist_id: artistId })}`),
	create: (body: CreateAlbumRequest) => secureJson<Album>('/api/albums', { method: 'POST', body }),
	update: (id: number, body: UpdateAlbumRequest) =>
		secureJson<MessageResponse>(`/api/albums/${id}`, { method: 'PUT', body }),
	delete: (id: number, artistId: number) =>
		secureJson<MessageResponse>(`/api/albums/${id}`, {
			method: 'DELETE',
			body: { artist_id: artistId },
		}),
};

export const spotifyApi = {
	searchArtists: (q: string, limit?: number) =>
		secureJson<SpotifyArtistSearchResult[]>(`/api/spotify/search/artists${query({ q, limit })}`),
	searchAlbums: (q: string, limit?: number) =>
		secureJson<SpotifyAlbumSearchResult[]>(`/api/spotify/search/albums${query({ q, limit })}`),
};
