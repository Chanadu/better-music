import type { TokenResponse } from './api-types';

export type { TokenResponse } from './api-types';

const storageKeys = {
	accessToken: 'betterMusicAccessToken',
	refreshToken: 'betterMusicRefreshToken',
	accessTokenExpiresAt: 'betterMusicAccessTokenExpiresAt',
};

export const saveTokens = (tokens: TokenResponse) => {
	const expiresAt = Date.now() + tokens.expires_in * 1000;

	localStorage.setItem(storageKeys.accessToken, tokens.access_token);
	localStorage.setItem(storageKeys.refreshToken, tokens.refresh_token);
	localStorage.setItem(storageKeys.accessTokenExpiresAt, String(expiresAt));
};

export const clearTokens = () => {
	Object.values(storageKeys).forEach((key) => localStorage.removeItem(key));
};

const getRefreshToken = () => localStorage.getItem(storageKeys.refreshToken);

const refreshAccessToken = async () => {
	const refreshToken = getRefreshToken();

	if (!refreshToken) return null;

	try {
		const { authApi } = await import('./api');
		const tokens = await authApi.refresh({ refresh_token: refreshToken });
		saveTokens(tokens);

		return tokens.access_token;
	} catch {
		clearTokens();
		return null;
	}
};

export const getValidAccessToken = async () => {
	const accessToken = localStorage.getItem(storageKeys.accessToken);
	const expiresAt = Number(localStorage.getItem(storageKeys.accessTokenExpiresAt) ?? 0);
	const refreshSkewMs = 30_000;

	if (accessToken && expiresAt - refreshSkewMs > Date.now()) {
		return accessToken;
	}

	return refreshAccessToken();
};

export const logout = async () => {
	const refreshToken = getRefreshToken();

	try {
		if (refreshToken) {
			const { authApi } = await import('./api');
			await authApi.logout({ refresh_token: refreshToken });
		}
	} finally {
		clearTokens();
	}
};

export const redirectToLogin = () => {
	clearTokens();
	window.location.assign('/login');
};

export const authenticatedFetch = async (input: RequestInfo | URL, init: RequestInit = {}) => {
	const accessToken = await getValidAccessToken();

	if (!accessToken) {
		throw new Error('Not authenticated');
	}

	const headers = new Headers(init.headers);
	headers.set('Authorization', `Bearer ${accessToken}`);

	const response = await fetch(input, {
		...init,
		headers,
	});

	if (response.status !== 401) {
		return response;
	}

	const refreshedAccessToken = await refreshAccessToken();

	if (!refreshedAccessToken) {
		return response;
	}

	headers.set('Authorization', `Bearer ${refreshedAccessToken}`);

	return fetch(input, {
		...init,
		headers,
	});
};
