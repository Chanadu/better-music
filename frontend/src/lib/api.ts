const ACCESS_TOKEN_KEY = 'token';
const REFRESH_TOKEN_KEY = 'refreshToken';
const TOKEN_EXPIRY_BUFFER_MS = 30_000;

export const LOGIN_PATH = '/';
export const APP_HOME_PATH = '/artists';

type TokenResponse = {
	access_token?: string;
	refresh_token?: string;
	token_type?: string;
	expires_in?: number;
};

type ApiFetchOptions = {
	auth?: boolean;
	retryOnUnauthorized?: boolean;
};

let refreshRequest: Promise<string | null> | null = null;

const redirectToLogin = (): void => {
	window.location.replace(LOGIN_PATH);
};

const decodeBase64Url = (value: string): string => {
	const normalized = value.replace(/-/g, '+').replace(/_/g, '/');
	const padding = normalized.length % 4;
	const padded = padding === 0 ? normalized : `${normalized}${'='.repeat(4 - padding)}`;
	return atob(padded);
};

const parseJwtPayload = (token: string): { exp?: number } | null => {
	const [, payload] = token.split('.');
	if (!payload) {
		return null;
	}

	try {
		const decoded = decodeBase64Url(payload);
		const parsed = JSON.parse(decoded) as { exp?: number };
		return typeof parsed === 'object' && parsed !== null ? parsed : null;
	} catch {
		return null;
	}
};

export const isAccessTokenValid = (token: string): boolean => {
	const payload = parseJwtPayload(token);
	if (typeof payload?.exp !== 'number') {
		return false;
	}

	return payload.exp * 1000 > Date.now() + TOKEN_EXPIRY_BUFFER_MS;
};

const mergeHeaders = (base?: HeadersInit, extra?: Record<string, string>): Headers => {
	const headers = new Headers(base ?? {});
	for (const [key, value] of Object.entries(extra ?? {})) {
		headers.set(key, value);
	}
	return headers;
};

export const getAccessToken = (): string | null => {
	const token = localStorage.getItem(ACCESS_TOKEN_KEY);
	if (!token || token === 'undefined' || token === 'null') {
		return null;
	}
	return token;
};

export const getRefreshToken = (): string | null => {
	const token = localStorage.getItem(REFRESH_TOKEN_KEY);
	if (!token || token === 'undefined' || token === 'null') {
		return null;
	}
	return token;
};

export const hasStoredAuthTokens = (): boolean => Boolean(getAccessToken() || getRefreshToken());

export const storeAuthTokens = (payload: TokenResponse): void => {
	if (payload.access_token) {
		localStorage.setItem(ACCESS_TOKEN_KEY, payload.access_token);
	}
	if (payload.refresh_token) {
		localStorage.setItem(REFRESH_TOKEN_KEY, payload.refresh_token);
	}
};

export const clearAuthTokens = (): void => {
	localStorage.removeItem(ACCESS_TOKEN_KEY);
	localStorage.removeItem(REFRESH_TOKEN_KEY);
};

export const restoreAuthSession = async (): Promise<boolean> => {
	const accessToken = getAccessToken();
	if (accessToken && isAccessTokenValid(accessToken)) {
		return true;
	}

	const refreshToken = getRefreshToken();
	if (!refreshToken) {
		if (accessToken) {
			clearAuthTokens();
		}
		return false;
	}

	const refreshedToken = await refreshAccessToken();
	return Boolean(refreshedToken && isAccessTokenValid(refreshedToken));
};

export const requireAuth = (): string => {
	const accessToken = getAccessToken();
	const refreshToken = getRefreshToken();
	if (!accessToken && !refreshToken) {
		redirectToLogin();
		return '';
	}
	return accessToken ?? '';
};

export const getErrorMessage = async (response: Response, fallback = 'Request failed'): Promise<string> => {
	try {
		const data = await response.json();
		if (typeof data?.error === 'string' && data.error) {
			return data.error;
		}
		return fallback;
	} catch {
		return fallback;
	}
};

export const apiFetch = async (
	url: string,
	init: RequestInit = {},
	options: ApiFetchOptions = {},
): Promise<Response> => {
	const { auth = false, retryOnUnauthorized = true } = options;
	const headers = mergeHeaders(init.headers);

	if (auth) {
		const accessToken = getAccessToken();
		const refreshToken = getRefreshToken();
		if (!accessToken && !refreshToken) {
			redirectToLogin();
			return new Response(null, { status: 401, statusText: 'Not authenticated' });
		}
		if (accessToken) {
			headers.set('Authorization', `Bearer ${accessToken}`);
		}
	}

	const response = await fetch(url, {
		...init,
		headers,
	});

	if (auth && response.status === 401 && retryOnUnauthorized) {
		const refreshedToken = await refreshAccessToken();
		if (!refreshedToken) {
			clearAuthTokens();
			redirectToLogin();
			return response;
		}

		const retryHeaders = mergeHeaders(init.headers, {
			Authorization: `Bearer ${refreshedToken}`,
		});
		return fetch(url, {
			...init,
			headers: retryHeaders,
		});
	}

	return response;
};

export const refreshAccessToken = async (): Promise<string | null> => {
	if (refreshRequest) {
		return refreshRequest;
	}

	const refreshToken = getRefreshToken();
	if (!refreshToken) {
		return null;
	}

	refreshRequest = (async () => {
		const response = await apiFetch(
			'/api/auth/refresh',
			{
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
				},
				body: JSON.stringify({ refresh_token: refreshToken }),
			},
			{ auth: false, retryOnUnauthorized: false },
		);

		if (!response.ok) {
			clearAuthTokens();
			return null;
		}

		const data = (await response.json()) as TokenResponse;
		storeAuthTokens(data);
		return data.access_token ?? null;
	})();

	try {
		return await refreshRequest;
	} finally {
		refreshRequest = null;
	}
};

export const logout = async (): Promise<void> => {
	const refreshToken = getRefreshToken();

	if (refreshToken) {
		await apiFetch(
			'/api/auth/logout',
			{
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
				},
				body: JSON.stringify({ refresh_token: refreshToken }),
			},
			{ auth: false, retryOnUnauthorized: false },
		).catch(() => undefined);
	}

	clearAuthTokens();
};
