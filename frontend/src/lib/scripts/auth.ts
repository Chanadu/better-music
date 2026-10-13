import type { TokenResponse } from './types';
import { deleteStoredLibrary, getPendingMutations } from './database-cache';
import { hasUnknownOutcome } from './queue-state';
import { syncConfig } from './sync-config';
import { persistentStorage } from './storage';

const keys = {
	access: 'betterMusicAccessToken',
	refresh: 'betterMusicRefreshToken',
	expires: 'betterMusicAccessTokenExpiresAt',
	userId: 'betterMusicUserId',
	generation: 'betterMusicSessionGeneration',
};
const sessionListeners = new Set<() => void>();
const notifySession = () => {
	for (const listener of sessionListeners) listener();
};
export const subscribeSessionChanges = (listener: () => void) => {
	sessionListeners.add(listener);
	return () => {
		sessionListeners.delete(listener);
	};
};
if (typeof window !== 'undefined')
	window.addEventListener('storage', (event) => {
		if (event.key === null || event.key === keys.generation || event.key === keys.userId) notifySession();
	});
const requestLock = (userId: number) => `better-music-requests:${userId}`;
let refreshRequest: Promise<string | null> | null = null;

export class AuthenticationError extends Error {
	constructor(message: string) {
		super(message);
		this.name = 'AuthenticationError';
	}
}

export const getCurrentUserId = () => {
	const userId = Number(persistentStorage.get(keys.userId));

	return Number.isInteger(userId) && userId > 0 ? userId : null;
};

export const getSessionGeneration = () => persistentStorage.get(keys.generation);

// Capture once before awaiting so a later session cannot accept this work.
export const sessionGuard = () => {
	const userId = getCurrentUserId();
	const generation = getSessionGeneration();
	return () => {
		if (getCurrentUserId() !== userId || getSessionGeneration() !== generation)
			throw new AuthenticationError('Account changed during the request');
	};
};

export const saveTokens = (tokens: TokenResponse, preserveSession = false) => {
	const previousSession = getSessionGeneration();
	if (!preserveSession || getCurrentUserId() !== tokens.user_id || !getSessionGeneration())
		persistentStorage.set(keys.generation, crypto.randomUUID());

	persistentStorage.set(keys.access, tokens.access_token);
	persistentStorage.set(keys.refresh, tokens.refresh_token);
	persistentStorage.set(keys.expires, String(Date.now() + tokens.expires_in * 1000));
	persistentStorage.set(keys.userId, String(tokens.user_id));
	if (previousSession !== getSessionGeneration()) notifySession();
};

export const invalidateSession = async () => {
	Object.values(keys).forEach((key) => persistentStorage.remove(key));
	if (Object.values(keys).some((key) => persistentStorage.get(key) !== null))
		throw new Error('Browser storage could not clear authentication. Enable storage and try again.');
	notifySession();
};

const blocksLogout = (queue: Awaited<ReturnType<typeof getPendingMutations>>) =>
	queue.some((item) => item.status === 'syncing' || hasUnknownOutcome(item));

export const canLogout = async () => {
	const userId = getCurrentUserId();
	const checkSession = sessionGuard();
	if (userId === null || !navigator.locks) return false;
	const [locks, queue] = await Promise.all([navigator.locks.query(), getPendingMutations(userId)]);
	checkSession();
	const lockNames = [syncConfig.lockPrefix + userId, requestLock(userId)];
	const active = [...(locks.held ?? []), ...(locks.pending ?? [])].some((lock) =>
		lockNames.includes(lock.name ?? ''),
	);
	return !active && !blocksLogout(queue);
};

// Use the same lock order as sync: account ownership, then HTTP requests.
export const clearSession = async () => {
	const userId = getCurrentUserId();
	const checkSession = sessionGuard();
	if (userId === null) return;
	if (!navigator.locks) throw new Error('This browser does not support safe sign out');
	await navigator.locks.request(syncConfig.lockPrefix + userId, { ifAvailable: true }, async (syncLock) => {
		if (!syncLock) throw new Error('Synchronization is active in another tab. Wait before signing out.');
		await navigator.locks.request(requestLock(userId), { ifAvailable: true }, async (lock) => {
			if (!lock) throw new Error('An account request is active. Wait before signing out.');
			checkSession();
			const queue = await getPendingMutations(userId);
			if (blocksLogout(queue)) throw new Error('A request outcome is unknown. Reconcile it before signing out.');
			await deleteStoredLibrary(userId, checkSession);
			await invalidateSession();
			persistentStorage.remove(`bettermusic:next-album:${userId}`);
		});
	});
};

export const hasStoredSession = () => getCurrentUserId() !== null && persistentStorage.get(keys.refresh) !== null;

const performRefresh = async () => {
	const refreshToken = persistentStorage.get(keys.refresh);
	const session = getSessionGeneration();
	const userId = getCurrentUserId();
	const isCurrent = () => persistentStorage.get(keys.refresh) === refreshToken && getSessionGeneration() === session;

	if (!refreshToken) return null;

	try {
		const response = await fetch('/api/auth/refresh', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ refresh_token: refreshToken }),
		});

		if (!response.ok) {
			if (isCurrent() && (response.status === 400 || response.status === 401)) await invalidateSession();
			return null;
		}

		const tokens = (await response.json()) as TokenResponse;
		if (!isCurrent() || tokens.user_id !== userId) return null;

		saveTokens(tokens, true);

		return tokens.access_token;
	} catch {
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
		await invalidateSession();
		return null;
	}

	return token && expiresAt - 30_000 > Date.now() ? token : requestRefresh();
};

const performAuthenticatedFetch = async (input: RequestInfo | URL, init: RequestInit, checkSession: () => void) => {
	let token = await getValidAccessToken();
	checkSession();

	if (!token) throw new AuthenticationError('Not authenticated');

	const headers = new Headers(init.headers);
	headers.set('Authorization', `Bearer ${token}`);

	let response = await fetch(input, { ...init, headers });
	checkSession();
	if (response.status !== 401) return response;

	token = await requestRefresh();
	checkSession();

	if (!token) {
		if (!hasStoredSession()) return response;
		throw new AuthenticationError('Authentication is temporarily unavailable');
	}

	headers.set('Authorization', `Bearer ${token}`);
	response = await fetch(input, { ...init, headers });

	checkSession();
	if (response.status === 401) await invalidateSession();
	return response;
};

export const authenticatedFetch = async (input: RequestInfo | URL, init: RequestInit = {}) => {
	const userId = getCurrentUserId();
	const checkSession = sessionGuard();
	if (userId === null) throw new AuthenticationError('Not authenticated');
	if (!navigator.locks) throw new AuthenticationError('This browser does not support account request locks');
	return navigator.locks.request(
		requestLock(userId),
		{ mode: 'shared', signal: init.signal ?? undefined },
		async () => {
			checkSession();
			return performAuthenticatedFetch(input, init, checkSession);
		},
	);
};

export const logout = async () => {
	const refreshToken = persistentStorage.get(keys.refresh);
	await clearSession();

	try {
		if (refreshToken)
			await fetch('/api/auth/logout', {
				keepalive: true,
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ refresh_token: refreshToken }),
			});
	} catch {
		// Local sign out succeeded; remote token revocation can fail offline.
	}
};
