import type {
	DatabaseData,
	LibraryUpdate,
	LocalDatabaseData,
	PendingMutation,
	ServerDatabaseData,
	SuccessfulMutationResponse,
} from './types';
import { compareMutationOrder, reconcileLibraryUpdate } from './database-reconciliation';
import { notifySyncChange } from './sync-events';

const databaseName = 'better-music';
const databaseVersion = 5;
const libraryStore = 'library-snapshots';
const outboxStore = 'mutation-outbox';
const userIndex = 'user-id';

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
			if (database.objectStoreNames.contains('sync-leases')) database.deleteObjectStore('sync-leases');

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

export const recoverInterruptedMutations = async (userId: number, checkOwnership: () => void) => {
	await updateLibraryAndOutbox(
		userId,
		(current) => ({
			...current,
			mutations: current.mutations.map((mutation) =>
				mutation.status === 'syncing' ? { ...mutation, status: 'pending' } : mutation,
			),
		}),
		checkOwnership,
	);
};

export const getStoredLibraryUpdate = (userId: number): Promise<LibraryUpdate | null> =>
	runTransaction('readonly', async (transaction) => {
		const [library, mutations] = await Promise.all([
			readRequest<LocalDatabaseData | undefined>(transaction.objectStore(libraryStore).get(userId)),
			readMutations(transaction, userId),
		]);
		return library ? { library, mutations: mutations.sort(compareMutationOrder) } : null;
	});

export const getStoredLibrary = (userId: number): Promise<LocalDatabaseData | null> =>
	runTransaction(
		'readonly',
		async (transaction) =>
			(await readRequest<LocalDatabaseData | undefined>(transaction.objectStore(libraryStore).get(userId))) ??
			null,
	);

export const getPendingMutations = async (userId: number): Promise<PendingMutation[]> => {
	const mutations = await runTransaction('readonly', (transaction) => readMutations(transaction, userId));

	return mutations.sort(compareMutationOrder);
};

export const reconcileSuccessfulMutation = async (
	userId: number,
	mutationId: string,
	response: SuccessfulMutationResponse,
	checkOwnership?: () => void,
): Promise<LibraryUpdate | null> => {
	const result = await runTransaction('readwrite', async (transaction) => {
		checkOwnership?.();
		const snapshots = transaction.objectStore(libraryStore);
		const outbox = transaction.objectStore(outboxStore);

		const [library, mutations] = await Promise.all([
			readRequest<LocalDatabaseData | undefined>(snapshots.get(userId)),
			readMutations(transaction, userId),
		]);

		const mutation = mutations.find((item) => item.mutationId === mutationId);

		if (!mutation) return library ? { library, mutations } : null;
		if (!library) throw new Error('Cannot reconcile a mutation without its library');

		const updated = reconcileLibraryUpdate({ library, mutations }, mutation, response);
		checkOwnership?.();

		for (const item of updated.mutations) outbox.put(item);
		updated.library = { ...updated.library, revision: (library?.revision ?? 0) + 1 };

		snapshots.put(updated.library, userId);
		outbox.delete(mutationId);

		return updated;
	});
	notifySyncChange({ userId, type: 'library' });
	return result;
};

export const updateLibraryAndOutbox = async (
	userId: number,
	update: (current: LibraryUpdate) => LibraryUpdate,
	checkOwnership?: () => void,
): Promise<LibraryUpdate> => {
	const result = await runTransaction('readwrite', async (transaction) => {
		checkOwnership?.();
		const snapshots = transaction.objectStore(libraryStore);

		const [library, mutations] = await Promise.all([
			readRequest<LocalDatabaseData | undefined>(snapshots.get(userId)),
			readMutations(transaction, userId),
		]);

		const updated = update({ library: library ?? { artists: [], albums: [], loadedAt: 0 }, mutations });

		if (updated.mutations.some((mutation) => mutation.userId !== userId)) {
			throw new Error('Mutation user does not match the library owner');
		}

		checkOwnership?.();
		const outbox = transaction.objectStore(outboxStore);

		for (const mutation of mutations) outbox.delete(mutation.mutationId);
		for (const mutation of updated.mutations) outbox.add(mutation);

		updated.library = { ...updated.library, revision: (library?.revision ?? 0) + 1 };
		snapshots.put(updated.library, userId);

		return updated;
	});

	notifySyncChange({ userId, type: 'library' });

	return result;
};

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

	notifySyncChange({ userId, type: 'queue' });
	return result;
};

export const updatePendingMutation = async (
	userId: number,
	mutationId: string,
	update: (mutation: PendingMutation) => PendingMutation,
	checkOwnership?: () => void,
): Promise<PendingMutation | null> => {
	const result = await updateLibraryAndOutbox(
		userId,
		({ library, mutations }) => ({
			library,
			mutations: mutations.map((mutation) => {
				if (mutation.mutationId !== mutationId) return mutation;
				const updated = update(mutation);
				if (updated.mutationId !== mutationId) throw new Error('Cannot change a mutation ID');
				return updated;
			}),
		}),
		checkOwnership,
	);

	return result.mutations.find((mutation) => mutation.mutationId === mutationId) ?? null;
};

export const removePendingMutation = async (userId: number, mutationId: string): Promise<void> => {
	await updateLibraryAndOutbox(userId, ({ library, mutations }) => ({
		library,
		mutations: mutations.filter((mutation) => mutation.mutationId !== mutationId),
	}));
};

export const deleteStoredLibrary = async (userId: number): Promise<void> => {
	await runTransaction('readwrite', async (transaction) => {
		const mutations = await readMutations(transaction, userId);

		for (const mutation of mutations) {
			transaction.objectStore(outboxStore).delete(mutation.mutationId);
		}

		transaction.objectStore(libraryStore).delete(userId);
		transaction.objectStore(libraryStore).put({ generation: crypto.randomUUID() }, `sync:${userId}`);
	});

	notifySyncChange({ userId, type: 'library' });
};

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

export const setStoredDatabaseCache = async (userId: number, value: ServerDatabaseData): Promise<boolean> => {
	const saved = await runTransaction('readwrite', async (transaction) => {
		const pending = await readRequest(transaction.objectStore(outboxStore).index(userIndex).count(userId));
		if (pending) return false;

		const store = transaction.objectStore(libraryStore);
		const previous = await readRequest<LocalDatabaseData | undefined>(store.get(userId));
		store.put({ ...value, revision: (previous?.revision ?? 0) + 1 }, userId);
		return true;
	});

	if (saved) notifySyncChange({ userId, type: 'library' });
	return saved;
};

export type SyncMetadata = {
	generation: string;
	lastQueueCompletedAt?: number;
	lastLibraryRefreshedAt?: number;
	lastSyncedAt?: number;
};

export const getSyncMetadata = (userId: number): Promise<SyncMetadata> =>
	runTransaction('readwrite', async (transaction) => {
		const store = transaction.objectStore(libraryStore);
		const key = `sync:${userId}`;
		const existing = await readRequest<SyncMetadata | undefined>(store.get(key));
		const value = existing ?? { generation: crypto.randomUUID() };

		if (!existing) store.put(value, key);
		return value;
	});

export const markQueueCompletion = (userId: number, generation: string, check: () => void) =>
	runTransaction('readwrite', async (transaction) => {
		const store = transaction.objectStore(libraryStore);
		const [metadata, queue] = await Promise.all([
			readRequest<SyncMetadata | undefined>(store.get(`sync:${userId}`)),
			readMutations(transaction, userId),
		]);

		check();

		if (metadata?.generation !== generation) throw new Error('Library cache changed');
		if (!queue.some((item) => item.status === 'pending' || item.status === 'syncing'))
			store.put({ ...metadata, lastQueueCompletedAt: Date.now() }, `sync:${userId}`);
	});

export const mergeLibraryRefresh = (
	userId: number,
	value: ServerDatabaseData,
	baseline: LocalDatabaseData | null,
	generation: string,
	check: () => void,
): Promise<LibraryUpdate> =>
	runTransaction('readwrite', async (transaction) => {
		const store = transaction.objectStore(libraryStore);
		const [library, mutations, metadata] = await Promise.all([
			readRequest<LocalDatabaseData | undefined>(store.get(userId)),
			readMutations(transaction, userId),
			readRequest<SyncMetadata | undefined>(store.get(`sync:${userId}`)),
		]);

		check();

		if (metadata?.generation !== generation) throw new Error('Library cache changed during refresh');

		const local = library ?? { artists: [], albums: [], loadedAt: 0 };
		const protectedArtists = new Set(
			mutations.filter((item) => item.entity === 'artist').map((item) => item.entityId),
		);

		const protectedAlbums = new Set(
			mutations.filter((item) => item.entity === 'album').map((item) => item.entityId),
		);

		for (const item of mutations) if (item.entity === 'album') protectedArtists.add(item.payload.artist_id);

		const merge = <T extends { id: number | string }>(
			server: T[],
			current: T[],
			protectedIds: Set<number | string>,
		): T[] => {
			return [
				...server.filter((item) => !protectedIds.has(item.id)),
				...current.filter((item) => protectedIds.has(item.id)),
			];
		};

		// Every local write increments the revision, including edits later reverted.
		if ((local.revision ?? 0) !== (baseline?.revision ?? 0)) {
			for (const item of [...local.artists, ...(baseline?.artists ?? [])]) protectedArtists.add(item.id);
			for (const item of [...local.albums, ...(baseline?.albums ?? [])]) protectedAlbums.add(item.id);
		}

		const merged = {
			revision: (local.revision ?? 0) + 1,
			artists: merge(value.artists, local.artists, protectedArtists),
			albums: merge(value.albums, local.albums, protectedAlbums),
			loadedAt: value.loadedAt,
		};

		check();

		store.put(merged, userId);
		store.put(
			{
				...metadata,
				lastLibraryRefreshedAt: value.loadedAt,
				...(!mutations.some((item) => item.status === 'pending' || item.status === 'syncing') ?
					{ lastSyncedAt: value.loadedAt }
				:	{}),
			},
			`sync:${userId}`,
		);

		return { library: merged, mutations };
	}).then((result) => {
		notifySyncChange({ userId, type: 'library' });
		return result;
	});
