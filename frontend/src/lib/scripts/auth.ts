import { browser } from '$app/environment';
import SuperTokens from 'supertokens-web-js';
import EmailPassword from 'supertokens-web-js/recipe/emailpassword';
import Session from 'supertokens-web-js/recipe/session';
import { persistentStorage } from './storage';

let initialized = false;

function withAccountHeader(init: RequestInit, userId: string | null): RequestInit {
	if (userId === null) return init;
	const headers = new Headers(init.headers);
	headers.set('X-Better-Music-Account', userId);
	return { ...init, headers };
}

export function initializeAuthentication(): void {
	if (!browser || initialized) return;
	SuperTokens.init({
		appInfo: { appName: 'Better Music', apiDomain: window.location.origin, apiBasePath: '/api/auth' },
		recipeList: [
			Session.init({
				tokenTransferMethod: 'cookie',
				preAPIHook: async ({ action, url, requestInit, userContext }) => {
					if (action !== 'SIGN_OUT') return { url, requestInit };

					const owner = userContext?.expectedOwner;
					return {
						url,
						requestInit: withAccountHeader(requestInit, typeof owner === 'string' ? owner : null),
					};
				},
			}),
			EmailPassword.init(),
		],
	});
	initialized = true;
	// Retire credentials without touching cached ownership or unsynced records.
	for (const key of [
		'betterMusicAccessToken',
		'betterMusicRefreshToken',
		'betterMusicAccessTokenExpiresAt',
		'betterMusicAuthProvider',
	])
		persistentStorage.remove(key);
}

const keys = {
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
export const getCurrentUserId = () => {
	const id = persistentStorage.get(keys.userId);
	return id && !/^\d+$/.test(id) ? id : null;
};

const requestLock = (userId: string) => `better-music-requests:${userId}`;

export const getSessionGeneration = () => persistentStorage.get(keys.generation);

// Capture once before awaiting so a later session cannot accept this work.
export const sessionGuard = () => {
	const userId = getCurrentUserId();
	const generation = getSessionGeneration();
	return () => {
		if (getCurrentUserId() !== userId || getSessionGeneration() !== generation)
			throw new Error('Account changed during the request');
	};
};

const invalidateSession = () => {
	Object.values(keys).forEach((key) => persistentStorage.remove(key));
	if (Object.values(keys).some((key) => persistentStorage.get(key) !== null))
		throw new Error('Browser storage could not clear authentication. Enable storage and try again.');
	notifySession();
};

export const canChangeAccount = async () => {
	const userId = getCurrentUserId();
	const checkSession = sessionGuard();
	if (userId === null || !navigator.locks) return false;

	const { hasUnsyncedChanges } = await import('./database');
	if (await hasUnsyncedChanges(userId)) return false;

	const locks = await navigator.locks.query();
	checkSession();
	return ![...(locks.held ?? []), ...(locks.pending ?? [])].some((lock) => lock.name === requestLock(userId));
};

// All lifecycle operations exclude requests and local edits until identity state
// is updated. The global lock also serializes signin across tabs/accounts.
async function withLifecycle<T>(action: () => Promise<T>, requireSynced = true): Promise<T> {
	if (!navigator.locks) throw new Error('This browser does not support safe account changes.');
	const check = sessionGuard();
	return navigator.locks.request('better-music-auth-transition', { ifAvailable: true }, async (transition) => {
		if (!transition) throw new Error('Another account action is active. Wait and try again.');
		check();
		const userId = getCurrentUserId();
		if (userId === null) return action();
		return navigator.locks.request(requestLock(userId), { ifAvailable: true }, async (lock) => {
			if (!lock) throw new Error('An account request is active. Wait and try again.');
			const { withAccountLibraryLock } = await import('./database');
			return withAccountLibraryLock(userId, async (pending) => {
				check();
				if (pending && requireSynced)
					throw new Error('Sync your library before changing account information or signing out.');
				return action();
			});
		});
	});
}

export async function runAccountChange<T>(action: (fetcher: typeof fetch) => Promise<T>): Promise<T> {
	return withLifecycle(async () => {
		const expectedOwner = getCurrentUserId();
		if (expectedOwner === null) throw new Error('Not authenticated');
		const check = sessionGuard();
		const result = await action((input, init = {}) => performAuthenticatedFetch(input, init, expectedOwner, check));
		// The backend revokes all sessions after these operations.
		await Session.signOut({ userContext: { expectedOwner } }).catch(() => {});
		invalidateSession();
		return result;
	});
}

export const hasStoredSession = () => getCurrentUserId() !== null;

export async function hasRemoteSession({ verifyOwner = false } = {}): Promise<boolean> {
	initializeAuthentication();
	try {
		if (!(await Session.doesSessionExist())) return false;
		const previousOwner = getCurrentUserId();
		if (previousOwner === null) {
			await withLifecycle(() => loadSessionOwner(null), false);
		} else if (verifyOwner) {
			const response = await authenticatedFetch('/api/session', { cache: 'no-store' });
			return response.ok;
		}
		return true;
	} catch {
		return false;
	}
}

const performAuthenticatedFetch = async (
	input: RequestInfo | URL,
	init: RequestInit,
	userId: string,
	checkSession: () => void,
) => {
	initializeAuthentication();
	checkSession();
	const response = await fetch(input, withAccountHeader({ ...init, credentials: 'include' }, userId));
	checkSession();
	return response;
};

export const authenticatedFetch = async (input: RequestInfo | URL, init: RequestInit = {}) => {
	const userId = getCurrentUserId();
	const checkSession = sessionGuard();
	if (userId === null) throw new Error('Not authenticated');

	if (!navigator.locks) throw new Error('This browser does not support account request locks');

	return navigator.locks.request(requestLock(userId), { mode: 'shared', signal: init.signal ?? undefined }, () =>
		performAuthenticatedFetch(input, init, userId, checkSession),
	);
};

export const logout = async () => {
	initializeAuthentication();
	await withLifecycle(async () => {
		if (!navigator.onLine) throw new Error('Connect to the server before signing out.');
		await Session.signOut({ userContext: { expectedOwner: getCurrentUserId() } });
		invalidateSession();
	});
};

function sdkError(result: { status: string; formFields?: { error: string }[] }): Error {
	return new Error(
		result.status === 'FIELD_ERROR' ?
			result.formFields?.map((field) => field.error).join(' ')
		:	'Email or password is incorrect.',
	);
}

export type AuthMode = 'login' | 'register';

export async function authenticate(mode: AuthMode, email: string, password: string): Promise<void> {
	initializeAuthentication();
	if (!navigator.onLine) throw new Error('Connect to the server to sign in.');
	const previousOwner = getCurrentUserId();
	await withLifecycle(async () => {
		if (mode === 'register' && previousOwner !== null) throw new Error('Sign out before creating another account.');
		const formFields = [
			{ id: 'email', value: email.trim().toLowerCase() },
			{ id: 'password', value: password },
		];
		const result =
			mode === 'register' ?
				await EmailPassword.signUp({ formFields })
			:	await EmailPassword.signIn({
					formFields,
					options: {
						preAPIHook: async ({ url, requestInit }) => ({
							url,
							requestInit: withAccountHeader(requestInit, previousOwner),
						}),
					},
				});
		if (result.status !== 'OK') throw sdkError(result);
		await loadSessionOwner(previousOwner);
	}, false);
}

async function loadSessionOwner(previousOwner: string | null): Promise<void> {
	const response = await fetch(
		'/api/session',
		withAccountHeader({ credentials: 'include', cache: 'no-store' }, previousOwner),
	);
	if (!response.ok) throw new Error('Could not load your account. Retry signing in.');
	const { user_id: userId } = await response.json();
	if (typeof userId !== 'string' || !userId) throw new Error('Invalid account identity.');
	if (previousOwner !== null && previousOwner !== userId) throw new Error('Sign out before switching accounts.');

	const previousGeneration = getSessionGeneration();
	if (previousOwner !== userId || !previousGeneration) persistentStorage.set(keys.generation, crypto.randomUUID());
	persistentStorage.set(keys.userId, userId);
	if (getCurrentUserId() !== userId) throw new Error('Browser storage could not save your account.');
	if (previousOwner !== userId || previousGeneration !== getSessionGeneration()) notifySession();
}

export async function resetPassword(password: string): Promise<void> {
	initializeAuthentication();
	const result = await EmailPassword.submitNewPassword({ formFields: [{ id: 'password', value: password }] });
	if (result.status === 'RESET_PASSWORD_INVALID_TOKEN_ERROR')
		throw new Error('This reset link has expired or was already used. Ask your administrator for a new link.');
	if (result.status !== 'OK') throw sdkError(result);
	// A recovery flow never clears cached ownership or pending library changes.
}
