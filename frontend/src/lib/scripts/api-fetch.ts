import { authenticatedFetch, runAccountChange, sessionGuard } from './auth';

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

const parseResponse = async <T>(response: Response): Promise<T> => {
	const body = await response.json().catch(() => null);

	if (!response.ok) {
		const candidate = body as { error?: string; message?: string; detail?: string } | null;
		throw new ApiError(
			candidate?.error ?? candidate?.message ?? candidate?.detail ?? `Request failed: ${response.status}`,
			response.status,
			body,
			response.headers.get('Retry-After'),
		);
	}

	return body as T;
};

export const apiFetch = async <T>(path: string, init?: RequestInit): Promise<T> => {
	const checkSession = sessionGuard();

	const response = await authenticatedFetch(path, init);
	const result = await parseResponse<T>(response);

	checkSession();

	return result;
};

// Account changes already hold the exclusive lifecycle lock. Use its fetcher
// to send the owner header without reacquiring the shared request lock.
export const accountChangeFetch = <T>(path: string, init?: RequestInit): Promise<T> =>
	runAccountChange(async (fetcher) => parseResponse<T>(await fetcher(path, init)));

// Sign-in loads the owner before an authenticated request lock can be used.
export const sessionFetch = <T>(
	path: string,
	{ fetcher, ...init }: RequestInit & { fetcher?: typeof fetch } = {},
): Promise<T> =>
	fetcher ? fetcher(path, init).then(parseResponse<T>) : apiFetch<T>(path, init);
