import type { TokenResponse } from './api-types';

export type { TokenResponse } from './api-types';

const accessTokenKey = 'betterMusicAccessToken';
const refreshTokenKey = 'betterMusicRefreshToken';
const tokenTypeKey = 'betterMusicTokenType';
const accessTokenExpiresAtKey = 'betterMusicAccessTokenExpiresAt';
const refreshSkewMs = 30_000;

export const saveTokens = (tokens: TokenResponse) => {
	const expiresAt = Date.now() + tokens.expires_in * 1000;

	localStorage.setItem(accessTokenKey, tokens.access_token);
	localStorage.setItem(refreshTokenKey, tokens.refresh_token);
	localStorage.setItem(tokenTypeKey, tokens.token_type);
	localStorage.setItem(accessTokenExpiresAtKey, String(expiresAt));
};

export const clearTokens = () => {
	localStorage.removeItem(accessTokenKey);
	localStorage.removeItem(refreshTokenKey);
	localStorage.removeItem(tokenTypeKey);
	localStorage.removeItem(accessTokenExpiresAtKey);
};

export const getRefreshToken = () => localStorage.getItem(refreshTokenKey);

export const refreshAccessToken = async () => {
	const refreshToken = getRefreshToken();

	if (!refreshToken) return null;

	const response = await fetch('/api/auth/refresh', {
		method: 'POST',
		headers: {
			'Content-Type': 'application/json',
		},
		body: JSON.stringify({ refresh_token: refreshToken }),
	});

	if (!response.ok) {
		clearTokens();
		return null;
	}

	const tokens = (await response.json()) as TokenResponse;
	saveTokens(tokens);

	return tokens.access_token;
};

export const getValidAccessToken = async () => {
	const accessToken = localStorage.getItem(accessTokenKey);
	const expiresAt = Number(localStorage.getItem(accessTokenExpiresAtKey) ?? 0);

	if (accessToken && expiresAt - refreshSkewMs > Date.now()) {
		return accessToken;
	}

	return refreshAccessToken();
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
