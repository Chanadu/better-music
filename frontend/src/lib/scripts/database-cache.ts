import type { DatabaseData, PendingMutation } from './types';

const databaseName = 'better-music';
const databaseVersion = 2;
const snapshotStore = 'library-snapshots';
const outboxStore = 'mutation-outbox';
const outboxUserIndex = 'user-id';
let pendingWrite = Promise.resolve();

const runWrite = <T>(operation: () => Promise<T>) => {
	const result = pendingWrite.then(operation, operation);
	pendingWrite = result.then(
		() => undefined,
		() => undefined,
	);
	return result;
};

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

			if (!database.objectStoreNames.contains(outboxStore)) {
				const store = database.createObjectStore(outboxStore, { keyPath: 'mutationId' });
				store.createIndex(outboxUserIndex, 'userId');
			}
		};
		request.onsuccess = () => resolve(request.result);
		request.onerror = () => reject(request.error ?? new Error('Could not open IndexedDB'));
	});

const getRequestResult = <T>(request: IDBRequest<T>) =>
	new Promise<T>((resolve, reject) => {
		request.onsuccess = () => resolve(request.result);
		request.onerror = () => reject(request.error ?? new Error('IndexedDB request failed'));
	});

const waitForTransaction = (transaction: IDBTransaction) =>
	new Promise<void>((resolve, reject) => {
		transaction.oncomplete = () => resolve();
		transaction.onabort = () => reject(transaction.error ?? new Error('IndexedDB transaction aborted'));
		transaction.onerror = () => reject(transaction.error ?? new Error('IndexedDB transaction failed'));
	});

const runTransaction = async <T>(
	stores: string | string[],
	mode: IDBTransactionMode,
	operation: (transaction: IDBTransaction) => T | Promise<T>,
) => {
	const database = await openDatabase();
	const transaction = database.transaction(stores, mode);
	const completion = waitForTransaction(transaction);

	try {
		const result = await operation(transaction);
		await completion;
		return result;
	} catch (error) {
		try {
			transaction.abort();
		} catch {
			// The transaction may already have aborted or completed.
		}
		await completion.catch(() => undefined);
		throw error;
	} finally {
		database.close();
	}
};

export const getStoredDatabaseCache = async (userId: number): Promise<DatabaseData | null> => {
	try {
		const value = await runTransaction<DatabaseData | undefined>(snapshotStore, 'readonly', (transaction) =>
			getRequestResult(transaction.objectStore(snapshotStore).get(userId)),
		);

		return value ?? null;
	} catch (error) {
		console.warn('Could not read the persistent library cache', error);
		return null;
	}
};

export const setStoredDatabaseCache = async (userId: number, value: DatabaseData) => {
	try {
		await runWrite(() =>
			runTransaction(snapshotStore, 'readwrite', (transaction) =>
				getRequestResult(transaction.objectStore(snapshotStore).put(value, userId)),
			),
		);
	} catch (error) {
		console.warn('Could not persist the library for offline use', error);
	}
};

/**
 * Updates the visible local library and its outbox in one IndexedDB transaction.
 * The returned snapshot can be published to the Svelte store after the commit.
 */
export const applyOptimisticMutation = async (
	userId: number,
	mutation: PendingMutation,
	update: (data: DatabaseData) => DatabaseData,
) => {
	if (mutation.userId !== userId) throw new Error('Mutation user does not match the library owner');

	return runWrite(() =>
		runTransaction([snapshotStore, outboxStore], 'readwrite', async (transaction) => {
			const snapshots = transaction.objectStore(snapshotStore);
			const data = await getRequestResult<DatabaseData | undefined>(snapshots.get(userId));
			if (!data) throw new Error('Cannot apply a local change before the library has been loaded');

			const updated = update(data);
			snapshots.put(updated, userId);
			transaction.objectStore(outboxStore).add(mutation);
			return updated;
		}),
	);
};

export const getPendingMutations = async (userId: number): Promise<PendingMutation[]> => {
	const mutations = await runTransaction<PendingMutation[]>(outboxStore, 'readonly', (transaction) =>
		getRequestResult(transaction.objectStore(outboxStore).index(outboxUserIndex).getAll(userId)),
	);

	return mutations.sort((left, right) => left.createdAt - right.createdAt);
};

export const clearStoredDatabaseCaches = async () => {
	try {
		await runWrite(() =>
			runTransaction([snapshotStore, outboxStore], 'readwrite', (transaction) => {
				transaction.objectStore(snapshotStore).clear();
				transaction.objectStore(outboxStore).clear();
			}),
		);
	} catch (error) {
		console.warn('Could not clear persistent library data', error);
	}
};
