import { authenticatedFetch } from './auth';
import type {
	Album,
	AccountResponse,
	Artist,
	AuthRequest,
	CreateAlbumRequest,
	CreateArtistRequest,
	MessageResponse,
	DeleteAccountRequest,
	SpotifyAlbumSearchResult,
	SpotifyArtistSearchResult,
	TokenResponse,
	UpdateAlbumRequest,
	UpdateEmailRequest,
	UpdatePasswordRequest,
	UpdateArtistRequest,
} from './api-types';

export class ApiError extends Error {
	constructor(
		message: string,
		public status: number,
		public body: unknown,
		public retryAfter: string | null = null,
	) {
		super(message);
		this.name = 'ApiError';
	}
}

type JsonInit = Omit<RequestInit, 'body'> & { body?: unknown };

export type MutationOptions = { mutationId: string };
export type VersionedMutationOptions = MutationOptions & { baseVersion: number };

const mutationHeaders = (options: MutationOptions): Headers => {
	if (!options.mutationId.trim() || options.mutationId.length > 255) {
		throw new Error('A persisted mutation ID is required');
	}

	return new Headers({ 'Idempotency-Key': options.mutationId });
};

const versionedMutationHeaders = (options: VersionedMutationOptions): Headers => {
	const headers = mutationHeaders(options);
	if (!Number.isSafeInteger(options.baseVersion) || options.baseVersion <= 0) {
		throw new Error('A positive record version is required');
	}

	headers.set('If-Match', String(options.baseVersion));
	return headers;
};

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
			response.headers.get('Retry-After'),
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
};

export const accountApi = {
	get: () => secureJson<AccountResponse>('/api/account'),
	updateEmail: (body: UpdateEmailRequest) =>
		secureJson<MessageResponse>('/api/account/email', { method: 'PUT', body }),
	updatePassword: (body: UpdatePasswordRequest) =>
		secureJson<MessageResponse>('/api/account/password', { method: 'PUT', body }),
	delete: (body: DeleteAccountRequest) => secureJson<MessageResponse>('/api/account', { method: 'DELETE', body }),
};

export const artistsApi = {
	list: () => secureJson<Artist[]>('/api/artists'),
	get: (id: number) => secureJson<Artist>(`/api/artists/${id}`),
	create: (body: CreateArtistRequest, options: MutationOptions) =>
		secureJson<Artist>('/api/artists', { method: 'POST', body, headers: mutationHeaders(options) }),
	update: (id: number, body: UpdateArtistRequest, options: VersionedMutationOptions) =>
		secureJson<Artist>(`/api/artists/${id}`, { method: 'PUT', body, headers: versionedMutationHeaders(options) }),
	delete: (id: number, options: VersionedMutationOptions) =>
		secureJson<MessageResponse>(`/api/artists/${id}`, {
			method: 'DELETE',
			headers: versionedMutationHeaders(options),
		}),
};

export const albumsApi = {
	list: () => secureJson<Album[]>('/api/albums'),
	get: (id: number, artistId: number) => secureJson<Album>(`/api/albums/${id}${query({ artist_id: artistId })}`),
	create: (body: CreateAlbumRequest, options: MutationOptions) =>
		secureJson<Album>('/api/albums', { method: 'POST', body, headers: mutationHeaders(options) }),
	update: (id: number, body: UpdateAlbumRequest, options: VersionedMutationOptions) =>
		secureJson<Album>(`/api/albums/${id}`, { method: 'PUT', body, headers: versionedMutationHeaders(options) }),
	delete: (id: number, artistId: number, options: VersionedMutationOptions) =>
		secureJson<MessageResponse>(`/api/albums/${id}`, {
			method: 'DELETE',
			body: { artist_id: artistId },
			headers: versionedMutationHeaders(options),
		}),
};

export const spotifyApi = {
	getArtist: (id: string) => secureJson<SpotifyArtistSearchResult>(`/api/spotify/artists/${encodeURIComponent(id)}`),
	getAlbum: (id: string) => secureJson<SpotifyAlbumSearchResult>(`/api/spotify/albums/${encodeURIComponent(id)}`),
	searchArtists: (q: string, limit?: number) =>
		secureJson<SpotifyArtistSearchResult[]>(`/api/spotify/search/artists${query({ q, limit })}`),
	searchAlbums: (q: string, limit?: number) =>
		secureJson<SpotifyAlbumSearchResult[]>(`/api/spotify/search/albums${query({ q, limit })}`),
};
