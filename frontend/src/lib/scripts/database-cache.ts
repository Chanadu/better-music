import type { DatabaseData, LocalDatabaseData, PendingMutation, ServerDatabaseData } from './types';

const databaseName = 'better-music';
const databaseVersion = 3;
const libraryStore = 'library-snapshots';
const outboxStore = 'mutation-outbox';
const userIndex = 'user-id';

type LibraryUpdate = { library: LocalDatabaseData; mutations: PendingMutation[] };

const openDatabase = () =>
	new Promise<IDBDatabase>((resolve, reject) => {
		if (!globalThis.indexedDB) {
			reject(new Error('IndexedDB is not available'));
			return;
		}
		const request = indexedDB.open(databaseName, databaseVersion);
		let blocked = false;
		request.onupgradeneeded = () => {
			const database = request.result;
			if (!database.objectStoreNames.contains(libraryStore)) database.createObjectStore(libraryStore);
			const outbox =
				database.objectStoreNames.contains(outboxStore) ?
					request.transaction!.objectStore(outboxStore)
				:	database.createObjectStore(outboxStore, { keyPath: 'mutationId' });
			if (!outbox.indexNames.contains(userIndex)) outbox.createIndex(userIndex, 'userId');
		};
		request.onblocked = () => {
			blocked = true;
			reject(new Error('Close older app tabs to upgrade local storage, then try again'));
		};
		request.onerror = () => reject(request.error ?? new Error('Could not open IndexedDB'));
		request.onsuccess = () => {
			const database = request.result;
			database.onversionchange = () => database.close();
			if (blocked) database.close();
			else resolve(database);
		};
	});

const readRequest = <T>(request: IDBRequest<T>) =>
	new Promise<T>((resolve, reject) => {
		request.onsuccess = () => resolve(request.result);
		request.onerror = () => reject(request.error ?? new Error('IndexedDB request failed'));
	});

const runTransaction = async <T>(
	mode: IDBTransactionMode,
	operation: (transaction: IDBTransaction) => T | Promise<T>,
): Promise<T> => {
	const database = await openDatabase();
	try {
		const transaction = database.transaction([libraryStore, outboxStore], mode);
		const completed = new Promise<void>((resolve, reject) => {
			transaction.oncomplete = () => resolve();
			transaction.onabort = () => reject(transaction.error ?? new Error('IndexedDB transaction aborted'));
		});
		void completed.catch(() => undefined);
		try {
			const result = await operation(transaction);
			await completed;
			return result;
		} catch (error) {
			try {
				transaction.abort();
			} catch {}
			await completed.catch(() => undefined);
			throw error;
		}
	} finally {
		database.close();
	}
};

const readMutations = (transaction: IDBTransaction, userId: number) =>
	readRequest<PendingMutation[]>(transaction.objectStore(outboxStore).index(userIndex).getAll(userId));

export const getStoredLibrary = (userId: number): Promise<LocalDatabaseData | null> =>
	runTransaction(
		'readonly',
		async (transaction) =>
			(await readRequest<LocalDatabaseData | undefined>(transaction.objectStore(libraryStore).get(userId))) ??
			null,
	);

export const getPendingMutations = async (userId: number): Promise<PendingMutation[]> => {
	const mutations = await runTransaction('readonly', (transaction) => readMutations(transaction, userId));
	return mutations.sort((left, right) => (left.sequence ?? left.createdAt) - (right.sequence ?? right.createdAt));
};

export const updateLibraryAndOutbox = (
	userId: number,
	update: (current: LibraryUpdate) => LibraryUpdate,
): Promise<LibraryUpdate> =>
	runTransaction('readwrite', async (transaction) => {
		const snapshots = transaction.objectStore(libraryStore);
		const [library, mutations] = await Promise.all([
			readRequest<LocalDatabaseData | undefined>(snapshots.get(userId)),
			readMutations(transaction, userId),
		]);
		const updated = update({ library: library ?? { artists: [], albums: [], loadedAt: 0 }, mutations });
		if (updated.mutations.some((mutation) => mutation.userId !== userId)) {
			throw new Error('Mutation user does not match the library owner');
		}
		const outbox = transaction.objectStore(outboxStore);
		for (const mutation of mutations) outbox.delete(mutation.mutationId);
		for (const mutation of updated.mutations) outbox.add(mutation);
		snapshots.put(updated.library, userId);
		return updated;
	});

export const applyOptimisticMutation = async (
	userId: number,
	mutation: PendingMutation,
	update: (library: LocalDatabaseData) => LocalDatabaseData,
) => {
	const result = await updateLibraryAndOutbox(userId, ({ library, mutations }) => ({
		library: update(library),
		mutations: [
			...mutations,
			{
				...mutation,
				sequence: mutations.reduce((last, item) => Math.max(last, item.sequence ?? item.createdAt), 0) + 1,
			},
		],
	}));
	return result;
};

export const updatePendingMutation = async (
	userId: number,
	mutationId: string,
	update: (mutation: PendingMutation) => PendingMutation,
): Promise<PendingMutation | null> => {
	const result = await updateLibraryAndOutbox(userId, ({ library, mutations }) => ({
		library,
		mutations: mutations.map((mutation) => {
			if (mutation.mutationId !== mutationId) return mutation;
			const updated = update(mutation);
			if (updated.mutationId !== mutationId) throw new Error('Cannot change a mutation ID');
			return updated;
		}),
	}));
	return result.mutations.find((mutation) => mutation.mutationId === mutationId) ?? null;
};

export const removePendingMutation = async (userId: number, mutationId: string): Promise<void> => {
	await updateLibraryAndOutbox(userId, ({ library, mutations }) => ({
		library,
		mutations: mutations.filter((mutation) => mutation.mutationId !== mutationId),
	}));
};

export const deleteStoredLibrary = (userId: number): Promise<void> =>
	runTransaction('readwrite', async (transaction) => {
		const mutations = await readMutations(transaction, userId);

		for (const mutation of mutations) {
			transaction.objectStore(outboxStore).delete(mutation.mutationId);
		}

		transaction.objectStore(libraryStore).delete(userId);
	});

export const getStoredDatabaseCache = async (userId: number): Promise<DatabaseData | null> => {
	try {
		const library = await getStoredLibrary(userId);
		if (!library) return null;

		return library;
	} catch (error) {
		console.warn('Could not read the persistent library cache', error);

		return null;
	}
};

export const setStoredDatabaseCache = (userId: number, value: ServerDatabaseData): Promise<boolean> =>
	runTransaction('readwrite', async (transaction) => {
		const pending = await readRequest(transaction.objectStore(outboxStore).index(userIndex).count(userId));
		if (pending) return false;

		transaction.objectStore(libraryStore).put(value, userId);
		return true;
	});
