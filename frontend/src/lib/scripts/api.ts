import { authenticatedFetch, runAccountChange, sessionGuard } from './auth';
import type {
	AccountResponse,
	MessageResponse,
	DeleteAccountRequest,
	SpotifyAlbumSearchResult,
	SpotifyArtistSearchResult,
	UpdateEmailRequest,
	UpdatePasswordRequest,
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

const secureJson = async <T>(path: string, init: JsonInit = {}) => {
	const checkSession = sessionGuard();
	const result = await json<T>(path, init, authenticatedFetch);
	checkSession();
	return result;
};

const query = (values: Record<string, string | number | undefined>) => {
	const params = new URLSearchParams();
	Object.entries(values).forEach(([key, value]) => value !== undefined && params.set(key, String(value)));
	return params.size ? `?${params}` : '';
};

const accountChange = (path: string, init: JsonInit) =>
	runAccountChange((fetcher) => json<MessageResponse>(path, init, fetcher));

export const accountApi = {
	get: () => secureJson<AccountResponse>('/api/account'),
	updateEmail: (body: UpdateEmailRequest) => accountChange('/api/account/email', { method: 'PUT', body }),
	updatePassword: (body: UpdatePasswordRequest) => accountChange('/api/account/password', { method: 'PUT', body }),
	delete: (body: DeleteAccountRequest) => accountChange('/api/account', { method: 'DELETE', body }),
};

export const spotifyApi = {
	getArtist: (id: string) => secureJson<SpotifyArtistSearchResult>(`/api/spotify/artists/${encodeURIComponent(id)}`),
	getAlbum: (id: string) => secureJson<SpotifyAlbumSearchResult>(`/api/spotify/albums/${encodeURIComponent(id)}`),
	searchArtists: (q: string, limit?: number) =>
		secureJson<SpotifyArtistSearchResult[]>(`/api/spotify/search/artists${query({ q, limit })}`),
	searchAlbums: (q: string, limit?: number) =>
		secureJson<SpotifyAlbumSearchResult[]>(`/api/spotify/search/albums${query({ q, limit })}`),
};
