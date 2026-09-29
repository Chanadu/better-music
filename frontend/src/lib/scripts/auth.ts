import type { TokenResponse } from './types';
import { clearStoredDatabaseCaches } from './database-cache';
import { persistentStorage } from './storage';

const keys = {
	access: 'betterMusicAccessToken',
	refresh: 'betterMusicRefreshToken',
	expires: 'betterMusicAccessTokenExpiresAt',
	userId: 'betterMusicUserId',
};
let refreshRequest: Promise<string | null> | null = null;

export const getCurrentUserId = () => {
	const userId = Number(persistentStorage.get(keys.userId));

	return Number.isInteger(userId) && userId > 0 ? userId : null;
};

export const saveTokens = (tokens: TokenResponse) => {
	persistentStorage.set(keys.access, tokens.access_token);
	persistentStorage.set(keys.refresh, tokens.refresh_token);
	persistentStorage.set(keys.expires, String(Date.now() + tokens.expires_in * 1000));
	persistentStorage.set(keys.userId, String(tokens.user_id));
};

export const clearTokens = () => {
	Object.values(keys).forEach((key) => persistentStorage.remove(key));
	void clearStoredDatabaseCaches();
};

export const hasStoredSession = () => getCurrentUserId() !== null && persistentStorage.get(keys.refresh) !== null;

const performRefresh = async () => {
	const refreshToken = persistentStorage.get(keys.refresh);

	if (!refreshToken) return null;

	try {
		const response = await fetch('/api/auth/refresh', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ refresh_token: refreshToken }),
		});

		if (!response.ok) {
			const refreshIsCurrent = persistentStorage.get(keys.refresh) === refreshToken;
			if (refreshIsCurrent && (response.status === 400 || response.status === 401)) clearTokens();
			return null;
		}

		const tokens = (await response.json()) as TokenResponse;
		if (persistentStorage.get(keys.refresh) !== refreshToken) return null;

		saveTokens(tokens);

		return tokens.access_token;
	} catch {
		// A connection failure or temporary server problem is not a logout.
		// Preserve the session so its cached library can still be opened.
		return null;
	}
};

const requestRefresh = () => {
	if (!refreshRequest) {
		refreshRequest = performRefresh().finally(() => {
			refreshRequest = null;
		});
	}

	return refreshRequest;
};

export const getValidAccessToken = async () => {
	const token = persistentStorage.get(keys.access);
	const expiresAt = Number(persistentStorage.get(keys.expires));

	if (token && getCurrentUserId() === null) {
		clearTokens();
		return null;
	}

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
	if (!token) {
		if (!hasStoredSession()) return response;
		throw new Error('Authentication is temporarily unavailable');
	}

	headers.set('Authorization', `Bearer ${token}`);
	response = await fetch(input, { ...init, headers });

	return response;
};

export const logout = async () => {
	const refreshToken = persistentStorage.get(keys.refresh);
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
