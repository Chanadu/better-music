import { sessionCache } from './storage';

const cachePrefix = 'betterMusicDatabaseData:';
const databaseName = 'better-music';
const databaseVersion = 1;
const snapshotStore = 'library-snapshots';

export const databaseCacheKey = (userId: number) => `${cachePrefix}${userId}`;

const openDatabase = () =>
	new Promise<IDBDatabase>((resolve, reject) => {
		if (!('indexedDB' in globalThis)) {
			reject(new Error('IndexedDB is not available'));
			return;
		}

		const request = indexedDB.open(databaseName, databaseVersion);
		request.onupgradeneeded = () => {
			const database = request.result;
			if (!database.objectStoreNames.contains(snapshotStore)) database.createObjectStore(snapshotStore);
		};
		request.onsuccess = () => resolve(request.result);
		request.onerror = () => reject(request.error ?? new Error('Could not open IndexedDB'));
	});

const runTransaction = async <T>(
	mode: IDBTransactionMode,
	operation: (store: IDBObjectStore, resolve: (value: T) => void, reject: (reason?: unknown) => void) => void,
) => {
	const database = await openDatabase();

	try {
		return await new Promise<T>((resolve, reject) => {
			const transaction = database.transaction(snapshotStore, mode);
			transaction.onabort = () => reject(transaction.error ?? new Error('IndexedDB transaction aborted'));
			transaction.onerror = () => reject(transaction.error ?? new Error('IndexedDB transaction failed'));
			operation(transaction.objectStore(snapshotStore), resolve, reject);
		});
	} finally {
		database.close();
	}
};

export const getStoredDatabaseCache = async <T>(userId: number): Promise<T | null> => {
	try {
		const value = await runTransaction<T | undefined>('readonly', (store, resolve, reject) => {
			const request = store.get(userId);
			request.onsuccess = () => resolve(request.result as T | undefined);
			request.onerror = () => reject(request.error);
		});

		if (value !== undefined) return value;
	} catch (error) {
		console.warn('Could not read the persistent library cache', error);
	}

	// Promote the old per-tab cache after upgrading an existing installation.
	const legacyKey = databaseCacheKey(userId);
	const legacy = sessionCache.getJson<T>(legacyKey);
	if (legacy) void setStoredDatabaseCache(userId, legacy);

	return legacy;
};

export const setStoredDatabaseCache = async (userId: number, value: unknown) => {
	try {
		await runTransaction<void>('readwrite', (store, resolve, reject) => {
			const request = store.put(value, userId);
			request.onsuccess = () => resolve();
			request.onerror = () => reject(request.error);
		});
	} catch (error) {
		console.warn('Could not persist the library for offline use', error);
	}
};

export const clearStoredDatabaseCaches = async () => {
	sessionCache.keys().forEach((key) => {
		if (key.startsWith(cachePrefix)) sessionCache.remove(key);
	});

	try {
		await runTransaction<void>('readwrite', (store, resolve, reject) => {
			const request = store.clear();
			request.onsuccess = () => resolve();
			request.onerror = () => reject(request.error);
		});
	} catch (error) {
		console.warn('Could not clear the persistent library cache', error);
	}
};
