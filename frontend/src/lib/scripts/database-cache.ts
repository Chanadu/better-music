import { sessionCache } from './storage';
import type { DatabaseData } from './types';

const cachePrefix = 'betterMusicDatabaseData:';
const databaseName = 'better-music';
const databaseVersion = 1;
const snapshotStore = 'library-snapshots';

const databaseCacheKey = (userId: number) => `${cachePrefix}${userId}`;

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

const runTransaction = async <T>(mode: IDBTransactionMode, operation: (store: IDBObjectStore) => IDBRequest<T>) => {
	const database = await openDatabase();

	try {
		return await new Promise<T>((resolve, reject) => {
			const transaction = database.transaction(snapshotStore, mode);
			const request = operation(transaction.objectStore(snapshotStore));

			transaction.oncomplete = () => resolve(request.result);
			transaction.onabort = () => reject(transaction.error ?? new Error('IndexedDB transaction aborted'));
			transaction.onerror = () => reject(transaction.error ?? new Error('IndexedDB transaction failed'));
		});
	} finally {
		database.close();
	}
};

export const getStoredDatabaseCache = async (userId: number): Promise<DatabaseData | null> => {
	try {
		const value = await runTransaction<DatabaseData | undefined>('readonly', (store) => store.get(userId));

		if (value !== undefined) return value;
	} catch (error) {
		console.warn('Could not read the persistent library cache', error);
	}

	// Promote the old per-tab cache after upgrading an existing installation.
	const legacyKey = databaseCacheKey(userId);
	const legacy = sessionCache.getJson<DatabaseData>(legacyKey);
	if (legacy) void setStoredDatabaseCache(userId, legacy);

	return legacy;
};

export const setStoredDatabaseCache = async (userId: number, value: DatabaseData) => {
	try {
		await runTransaction('readwrite', (store) => store.put(value, userId));
	} catch (error) {
		console.warn('Could not persist the library for offline use', error);
	}
};

export const clearStoredDatabaseCaches = async () => {
	sessionCache.keys().forEach((key) => {
		if (key.startsWith(cachePrefix)) sessionCache.remove(key);
	});

	try {
		await runTransaction('readwrite', (store) => store.clear());
	} catch (error) {
		console.warn('Could not clear the persistent library cache', error);
	}
};
