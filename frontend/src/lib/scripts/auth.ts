import type { TokenResponse } from './types';

const keys = {
	access: 'betterMusicAccessToken',
	refresh: 'betterMusicRefreshToken',
	expires: 'betterMusicAccessTokenExpiresAt',
};

export const saveTokens = (tokens: TokenResponse) => {
	localStorage.setItem(keys.access, tokens.access_token);
	localStorage.setItem(keys.refresh, tokens.refresh_token);
	localStorage.setItem(keys.expires, String(Date.now() + tokens.expires_in * 1000));
};

export const clearTokens = () => Object.values(keys).forEach((key) => localStorage.removeItem(key));

const requestRefresh = async () => {
	const refreshToken = localStorage.getItem(keys.refresh);

	if (!refreshToken) return null;

	try {
		const response = await fetch('/api/auth/refresh', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ refresh_token: refreshToken }),
		});

		if (!response.ok) throw new Error('Refresh failed');

		const tokens = (await response.json()) as TokenResponse;
		saveTokens(tokens);

		return tokens.access_token;
	} catch {
		clearTokens();
		return null;
	}
};

export const getValidAccessToken = async () => {
	const token = localStorage.getItem(keys.access);
	const expiresAt = Number(localStorage.getItem(keys.expires) ?? 0);

	return token && expiresAt - 30_000 > Date.now() ? token : requestRefresh();
};

export const authenticatedFetch = async (input: RequestInfo | URL, init: RequestInit = {}) => {
	let token = await getValidAccessToken();
	if (!token) throw new Error('Not authenticated');

	const headers = new Headers(init.headers);
	headers.set('Authorization', `Bearer ${token}`);

	let response = await fetch(input, { ...init, headers });
	if (response.status !== 401) return response;

	token = await requestRefresh();
	if (!token) return response;

	headers.set('Authorization', `Bearer ${token}`);
	response = await fetch(input, { ...init, headers });

	return response;
};

export const logout = async () => {
	const refreshToken = localStorage.getItem(keys.refresh);
	try {
		if (refreshToken)
			await fetch('/api/auth/logout', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ refresh_token: refreshToken }),
			});
	} finally {
		clearTokens();
	}
};
