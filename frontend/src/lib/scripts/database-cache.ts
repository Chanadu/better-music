import type { DatabaseData } from './types';

const databaseName = 'better-music';
const databaseVersion = 1;
const snapshotStore = 'library-snapshots';
let pendingMutation = Promise.resolve<unknown>(undefined);

const runMutation = (operation: () => Promise<unknown>) =>
	(pendingMutation = pendingMutation.then(operation, operation));

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

		return value ?? null;
	} catch (error) {
		console.warn('Could not read the persistent library cache', error);
		return null;
	}
};

export const setStoredDatabaseCache = async (userId: number, value: DatabaseData) => {
	try {
		await runMutation(() => runTransaction('readwrite', (store) => store.put(value, userId)));
	} catch (error) {
		console.warn('Could not persist the library for offline use', error);
	}
};

export const clearStoredDatabaseCaches = async () => {
	try {
		await runMutation(() => runTransaction('readwrite', (store) => store.clear()));
	} catch (error) {
		console.warn('Could not clear the persistent library cache', error);
	}
};
