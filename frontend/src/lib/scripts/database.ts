import { writable } from 'svelte/store';
import { getCurrentUserId, getSessionGeneration, sessionGuard, subscribeSessionChanges } from './auth';
import { ApiError } from './api-fetch';
import { isAlbumRating, isAlbumYear, isLibraryName } from './library-validation';
import {
	pullLibrary,
	pushLibrary,
	type LibraryChange,
	type LibraryCheckpoint,
	type PushLibraryErrorResponse,
} from './generated/api';
import { cacheLibraryArtwork } from './artwork-cache';
import type { RxDatabase, RxCollection } from 'rxdb/plugins/core';
import type { RxReplicationState } from 'rxdb/plugins/replication';
import type {
	Album,
	Artist,
	DatabaseData,
	EntityId,
	LibraryEntity,
	RecordSyncStatus,
	CreateArtistRequest,
	UpdateArtistRequest,
	CreateAlbumRequest,
	UpdateAlbumRequest,
} from './types';

export type { DatabaseData } from './types';

type Doc = Artist | Album;

type Kind = 'artists' | 'albums';


export type SyncProblem = { id: string; kind: Kind; label: string; message: string; deleted: boolean; href: string };

type Dirty = { id: string; kind: Kind; recordId: string; token: string };

type Collections = {
	artists: RxCollection<Artist>;
	albums: RxCollection<Album>;
	dirty: RxCollection<Dirty>;
	syncstate: RxCollection<{ id: string; epoch: string }>;
};

type Context = {
	userId: string;
	generation: string | null;
	db: RxDatabase<Collections>;
	replications: RxReplicationState<any, LibraryCheckpoint>[];
	subscriptions: { unsubscribe(): void }[];
	active: number;
	activeKinds: Set<Kind>;
	lastError: string | null;
	closed: boolean;
	timer?: ReturnType<typeof setInterval>;
};

export type SyncStatus = {
	state: 'connecting' | 'synced' | 'pending' | 'offline' | 'error';
	lastSyncedAt: number | null;
	error?: string;
};

export const database = writable<DatabaseData | null>(null);
export const syncStatus = writable<SyncStatus>({ state: 'connecting', lastSyncedAt: null });
export const storageProblem = writable<string | null>(null);
export const pendingRecords = writable<Dirty[]>([]);
export const syncProblems = writable<SyncProblem[]>([]);
let context: Context | null = null;
let opening: Promise<Context> | null = null;
let transition: Promise<void> = Promise.resolve();
const isCurrent = (ctx: Context) =>
	!ctx.closed && ctx.userId === getCurrentUserId() && ctx.generation === getSessionGeneration();
const equal = (a: any, b: any) => JSON.stringify(sortObject(a)) === JSON.stringify(sortObject(b));

function sortObject(value: any): any {
	if (!value || typeof value !== 'object') return value;

	return Object.fromEntries(
		Object.keys(value)
			.filter((k) => !['_rev', '_meta', '_attachments'].includes(k))
			.sort()
			.map((k) => [k, value[k]]),
	);
}

const uuid = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/;
export const parseEntityId = (value: string | null): EntityId | null => (value && uuid.test(value) ? value : null);
export const markOffline = () => syncStatus.update((s) => ({ ...s, state: 'offline' }));
export const markConnecting = () => syncStatus.update((s) => ({ ...s, state: 'connecting' }));
export const markSyncError = (error = 'Sign in to the same account or retry synchronization.') =>
	syncStatus.update((s) => ({ ...s, state: 'error', error }));

function storageError(error: unknown): Error {
	const reason = error instanceof Error ? error.message : String(error);
	const message = `Could not save or open your local library. ${reason}`;
	storageProblem.set(message);
	return new Error(message);
}

const lockName = (userId: string) => `better-music-local-library:${userId}`;

async function localLock<T>(ctx: Context, fn: () => Promise<T>): Promise<T> {
	if (!navigator.locks) throw new Error('This browser needs Web Locks for safe local editing.');

	return navigator.locks.request(lockName(ctx.userId), async () => {
		if (!isCurrent(ctx)) throw new Error('Account changed');

		return fn();
	});
}

async function publish(ctx: Context) {
	if (!isCurrent(ctx)) return;

	const [artists, albums, dirty] = await Promise.all([
		ctx.db.artists.find().exec(),
		ctx.db.albums.find().exec(),
		ctx.db.dirty.find().exec(),
	]);
	if (!isCurrent(ctx)) return;

	const data: DatabaseData = {
		artists: artists.map((d) => d.toJSON() as Artist),
		albums: albums.map((d) => d.toJSON() as Album),
		loadedAt: Date.now(),
	};

	database.set(data);
	pendingRecords.set(dirty.map((d) => d.toJSON() as Dirty));
	cacheLibraryArtwork(data);
	syncStatus.update((s) => ({
		...s,
		state:
			!navigator.onLine ? 'offline'
				: ctx.lastError ? 'error'
					: ctx.active ? 'connecting'
						: dirty.length ? 'pending'
							: s.lastSyncedAt ? 'synced'
								: 'connecting',
		error: ctx.lastError ?? undefined,
	}));
}

async function closeContext(ctx: Context) {
	ctx.closed = true;
	clearInterval(ctx.timer);
	ctx.subscriptions.forEach((s) => s.unsubscribe());
	await Promise.allSettled(ctx.replications.map((r) => r.cancel()));
	await ctx.db.close();
}

subscribeSessionChanges(() => {
	const previous = context;
	context = null;
	database.set(null);
	pendingRecords.set([]);
	syncProblems.set([]);
	storageProblem.set(null);
	syncStatus.set({ state: 'connecting', lastSyncedAt: null });
	if (previous) transition = transition.then(() => closeContext(previous)).catch(console.error);
});

const nullableString = { type: ['string', 'null'] };

function schema(kind: Kind) {
	const properties: Record<string, any> = {
		id: { type: 'string', maxLength: 36 },
		created_at: { type: 'string' },
		edited_at: { type: 'string' },
		cover_url: nullableString,
		spotify_id: nullableString,
	};

	if (kind === 'artists') properties.name = { type: 'string' };
	else
		Object.assign(properties, {
			artist_id: { type: 'string', maxLength: 36 },
			title: { type: 'string' },
			year: { type: ['integer', 'null'], minimum: -2147483648, maximum: 2147483647 },
			listened: { type: 'boolean' },
			rating: { type: ['integer', 'null'], minimum: 1, maximum: 10 },
			comment: nullableString,
			listened_at: nullableString,
		});
	return { version: 0, primaryKey: 'id', type: 'object', properties, required: Object.keys(properties) } as any;
}

async function syncRequest<T>(ctx: Context, kind: Kind, request: (init: RequestInit) => Promise<T>): Promise<T> {
	if (!isCurrent(ctx)) throw new Error('Account changed');

	try {
		return await request({ signal: AbortSignal.timeout(30000) }).finally(() => {
			if (!isCurrent(ctx)) throw new Error('Account changed');
		});
	} catch (error) {
		if (!(error instanceof ApiError)) throw error;

		const result = error.body as Partial<PushLibraryErrorResponse> | null;
		const problems: SyncProblem[] = [];
		for (const failure of result?.failures ?? []) {
			const raw = (await ctx.db[kind].storageInstance.findDocumentsById([failure.id], true))[0] as any;
			const label = raw?.name ?? raw?.title ?? 'Library record';
			problems.push({
				id: failure.id,
				kind,
				label,
				message: failure.error,
				deleted: raw?._deleted ?? false,
				href:
					kind === 'artists' ?
						`/artist?id=${failure.id}`
						: `/album?id=${failure.id}&artist_id=${raw?.artist_id ?? ''}`,
			});
		}

		if (isCurrent(ctx))
			syncProblems.update((previous) => [
				...previous.filter((p) => !problems.some((n) => n.id === p.id && n.kind === p.kind)),
				...problems,
			]);
		const details = problems.map((p) => `${p.label}: ${p.message}`).join(' ');
		const message = result ? error.message : 'Synchronization failed';
		throw new Error(`${message}${details ? ' ' + details : ''}`);
	}
}

async function openLibrary(): Promise<Context> {
	const userId = getCurrentUserId();
	if (userId === null) throw new Error('Not authenticated');

	const generation = getSessionGeneration();
	const check = sessionGuard();
	if (context && isCurrent(context)) return context;

	if (opening) {
		await opening;
		check();
		return openLibrary();
	}

	const task = (async () => {
		await transition;
		check();

		const [{ createRxDatabase }, { getRxStorageDexie }, { replicateRxCollection }] = await Promise.all([
			import('rxdb/plugins/core'),
			import('rxdb/plugins/storage-dexie'),
			import('rxdb/plugins/replication'),
		]);
		check();
		const db = await createRxDatabase<Collections>({
			name: `bettermusicrxdb1user${userId}`,
			storage: getRxStorageDexie(),
			multiInstance: true,
		});

		try {
			await db.addCollections({
				artists: {
					schema: schema('artists'),
					conflictHandler: {
						isEqual: equal,
						resolve: async ({ realMasterState, newDocumentState }) =>
							Date.parse(newDocumentState.edited_at) > Date.parse(realMasterState.edited_at) ?
								newDocumentState
								: realMasterState,
					},
				},
				albums: {
					schema: schema('albums'),
					conflictHandler: {
						isEqual: equal,
						resolve: async ({ realMasterState, newDocumentState }) =>
							Date.parse(newDocumentState.edited_at) > Date.parse(realMasterState.edited_at) ?
								newDocumentState
								: realMasterState,
					},
				},
				dirty: {
					schema: {
						version: 0,
						primaryKey: 'id',
						type: 'object',
						properties: {
							id: { type: 'string', maxLength: 44 },
							kind: { type: 'string' },
							recordId: { type: 'string', maxLength: 36 },
							token: { type: 'string' },
						},
						required: ['id', 'kind', 'recordId', 'token'],
					},
				},
				syncstate: {
					schema: {
						version: 0,
						primaryKey: 'id',
						type: 'object',
						properties: { id: { type: 'string', maxLength: 10 }, epoch: { type: 'string' } },
						required: ['id', 'epoch'],
					},
				},
			});

			check();
			const ctx: Context = {
				userId,
				generation,
				db,
				replications: [],
				subscriptions: [],
				active: 0,
				activeKinds: new Set(),
				lastError: null,
				closed: false,
			};

			context = ctx;
			for (const kind of ['artists', 'albums'] as const) {
				let epoch: string | null = (await db.syncstate.findOne(kind).exec())?.epoch ?? null;
				check();
				if (!isCurrent(ctx)) throw new Error('Account changed');

				const replication = replicateRxCollection<Doc, LibraryCheckpoint>({
					collection: db[kind] as RxCollection<Doc>,
					replicationIdentifier: `${location.origin}/api/sync/v1/${userId}/${kind}`,
					live: true,
					retryTime: 2000,
					waitForLeadership: false,
					pull: {
						batchSize: 100,
						handler: async (checkpoint, limit) => {
							const page = await syncRequest(ctx, kind, (init) =>
								pullLibrary(kind, { checkpoint: checkpoint ?? null, limit }, init),
							);

							if (epoch && epoch !== page.checkpoint.epoch)
								throw new Error('Sync history changed. Export your local library before resetting.');
							epoch = page.checkpoint.epoch;
							await db.syncstate.upsert({ id: kind, epoch: epoch! });
							return page;
						},
					},
					push: {
						batchSize: 100,
						handler: async (rows) => {
							if (!epoch) {
								const page = await syncRequest(ctx, kind, (init) =>
									pullLibrary(kind, { checkpoint: null, limit: 1 }, init),
								);

								epoch = page.checkpoint.epoch;
								await db.syncstate.upsert({ id: kind, epoch: epoch! });
							}

							const latestDocs = await db[kind].storageInstance.findDocumentsById(
								rows.map((row) => row.newDocumentState.id),
								true,
							);
							const latestById = new Map(latestDocs.map((doc) => [doc.id, doc]));
							const outbound = rows.map((row) => {
								const latest = latestById.get(row.newDocumentState.id);
								return (
									latest &&
									Date.parse(latest.edited_at) > Date.parse(row.newDocumentState.edited_at)
								) ?
									{ ...row, newDocumentState: sortObject(latest) as Doc & { _deleted: boolean } }
									: row;
							});

							// RxDB's schema requires the wire fields that local UI types mark optional.
							const request = { epoch, rows: outbound as LibraryChange[] };
							const conflicts = await syncRequest(ctx, kind, (init) => pushLibrary(kind, request, init));
							const ids = new Set(conflicts.map((d) => d.id));
							if (isCurrent(ctx))
								syncProblems.update((problems) =>
									problems.filter(
										(p) => p.kind !== kind || !rows.some((row) => row.newDocumentState.id === p.id),
									),
								);
							await localLock(ctx, async () => {
								for (const row of outbound) {
									if (ids.has(row.newDocumentState.id)) continue;
									const latest = (
										await db[kind].storageInstance.findDocumentsById(
											[row.newDocumentState.id],
											true,
										)
									)[0];
									if (latest && equal(latest, row.newDocumentState)) {
										const marker = await db.dirty
											.findOne(`${kind}:${row.newDocumentState.id}`)
											.exec();
										if (marker) await marker.remove();
									}
								}
							});

							return conflicts;
						},
					},
				});

				ctx.replications.push(replication);
				ctx.subscriptions.push(
					replication.error$.subscribe((error) => {
						if (!isCurrent(ctx)) return;

						ctx.lastError =
							(error.parameters.errors as { message?: string }[] | undefined)?.[0]?.message ??
							'Synchronization failed. Your changes remain saved locally.';
						void publish(ctx);
					}),
				);
				ctx.subscriptions.push(
					replication.active$.subscribe((active) => {
						if (!isCurrent(ctx)) return;

						if (active) ctx.activeKinds.add(kind);
						else ctx.activeKinds.delete(kind);
						ctx.active = ctx.activeKinds.size;
						void publish(ctx);
					}),
				);
			}

			for (const collection of [db.artists, db.albums, db.dirty])
				ctx.subscriptions.push(
					(collection as RxCollection<any>).$.subscribe(() => void publish(ctx).catch(storageError)),
				);
			ctx.timer = setInterval(() => {
				if (isCurrent(ctx) && navigator.onLine) void refreshDatabaseSafely();
			}, 30000);
			storageProblem.set(null);
			await publish(ctx);
			return ctx;
		} catch (error) {
			await db.close();
			throw error;
		}
	})();
	opening = task;
	try {
		return await task;
	} catch (error) {
		check();
		throw storageError(error);
	} finally {
		if (opening === task) opening = null;
	}
}

export const getDatabaseData = async (): Promise<DatabaseData | null> => {
	if (getCurrentUserId() === null) return null;

	const check = sessionGuard();
	try {
		const ctx = await openLibrary();
		await publish(ctx);
		const [artists, albums] = await Promise.all([ctx.db.artists.find().exec(), ctx.db.albums.find().exec()]);
		if (!isCurrent(ctx)) return null;

		return {
			artists: artists.map((d) => d.toJSON() as Artist),
			albums: albums.map((d) => d.toJSON() as Album),
			loadedAt: Date.now(),
		};
	} catch (error) {
		try {
			check();
		} catch {
			return null;
		}

		storageError(error);
		return null;
	}
};

export const retryLocalStorage = async () => {
	await openLibrary();
	storageProblem.set(null);
};

export async function refreshDatabaseData(): Promise<DatabaseData> {
	const ctx = await openLibrary();
	const before = await localLock(ctx, async () => (await ctx.db.dirty.find().exec()).map((d) => d.toJSON()));
	ctx.lastError = null;
	markConnecting();
	ctx.replications.forEach((r) => r.reSync());
	await Promise.race([
		Promise.all(ctx.replications.map((r) => r.awaitInSync())),
		new Promise((_, reject) =>
			setTimeout(
				() => reject(new Error(ctx.lastError ?? 'Sync timed out. Your local edits remain saved.')),
				30000,
			),
		),
	]);
	if (!isCurrent(ctx)) throw new Error('Account changed');

	await localLock(ctx, async () => {
		if (ctx.active > 0) return;

		for (const previous of before) {
			const marker = await ctx.db.dirty.findOne(previous.id).exec();
			if (marker?.token === previous.token) await marker.remove();
		}
	});

	ctx.lastError = null;
	syncStatus.set({ state: 'synced', lastSyncedAt: Date.now() });
	await publish(ctx);
	return (await getDatabaseData())!;
}

export const refreshDatabaseSafely = async () => {
	if (!navigator.onLine) {
		markOffline();
		return;
	}

	try {
		await refreshDatabaseData();
	} catch (error) {
		markSyncError(error instanceof Error ? error.message : String(error));
	}
};

export const hasUnsyncedChanges = async (userId: string) => {
	if (userId !== getCurrentUserId()) return true;

	return withAccountLibraryLock(userId, async (pending) => pending);
};

export async function withAccountLibraryLock<T>(userId: string, action: (pending: boolean) => Promise<T>): Promise<T> {
	if (userId !== getCurrentUserId()) throw new Error('Account changed');
	const ctx = await openLibrary();
	return localLock(ctx, async () => {
		const pending = ctx.active > 0 || (await ctx.db.dirty.findOne().exec()) !== null;
		return action(pending);
	});
}

export function getRecordSyncStatus(
	records: readonly Dirty[],
	entity: LibraryEntity,
	id: EntityId,
	state?: SyncStatus['state'],
): RecordSyncStatus {
	if (!records.some((d) => d.kind === `${entity}s` && d.recordId === id)) return 'synced';

	return (
		state === 'error' ? 'failed'
			: state === 'offline' ? 'paused'
				: state === 'connecting' ? 'syncing'
					: 'pending'
	);
}

const requiredName = (value: string, label: string) => {
	const result = value.trim();
	if (!result) throw new Error(`${label} is required`);

	if (!isLibraryName(result)) throw new Error(`${label} is too long`);

	return result;
};

async function edit(kind: Kind, id: string, change: (old: Doc | null) => Doc, includeDeleted = false): Promise<Doc> {
	const ctx = await openLibrary();
	try {
		return await localLock(ctx, async () => {
			const collection = ctx.db[kind] as RxCollection<Doc>;
			const found = await collection.findOne(id).exec();
			const raw =
				!found && includeDeleted ? (await collection.storageInstance.findDocumentsById([id], true))[0] : null;
			const next = change(
				found ? (found.toJSON() as Doc)
					: raw ? (sortObject(raw) as Doc)
						: null,
			);
			if (kind === 'artists') {
				if (next._deleted) {
					if (await ctx.db.albums.findOne({ selector: { artist_id: id } }).exec())
						throw new Error('Remove this artist’s albums first');
				} else {
					const name = (next as Artist).name;
					const records = await ctx.db.artists.find().exec();
					if (records.some((d) => d.id !== id && d.name.toLowerCase() === name.toLowerCase()))
						throw new Error('Artist with this name already exists');
				}
			} else if (!next._deleted) {
				const album = next as Album;
				if (!(await ctx.db.artists.findOne(album.artist_id).exec())) throw new Error('Artist not found');

				const records = await ctx.db.albums.find({ selector: { artist_id: album.artist_id } }).exec();
				if (records.some((d) => d.id !== id && d.title.toLowerCase() === album.title.toLowerCase()))
					throw new Error('Album with this title already exists for this artist');
			}

			await ctx.db.dirty.upsert({ id: `${kind}:${id}`, kind, recordId: id, token: crypto.randomUUID() });
			await collection.upsert({ ...next, _deleted: next._deleted ?? false });
			await publish(ctx);
			return next;
		});
	} catch (error) {
		if (error instanceof Error && /quota|storage|indexeddb/i.test(error.message)) storageError(error);
		throw error;
	}
}

const fields = <T extends object>(value: T) =>
	Object.fromEntries(Object.entries(value).filter(([, v]) => v !== undefined)) as T;
function creationDate(value?: string): string {
	if (value === undefined) return new Date().toISOString();
	if (!Number.isFinite(Date.parse(value))) throw new Error('Invalid creation date');
	return value;
}

export const artistsLibrary = {
	create: async (payload: CreateArtistRequest, createdAt?: string) => {
		const id = crypto.randomUUID();
		return edit('artists', id, () => ({
			id,
			name: requiredName(payload.name, 'Name'),
			cover_url: payload.cover_url ?? null,
			spotify_id: payload.spotify_id ?? null,
			created_at: creationDate(createdAt),
			edited_at: new Date().toISOString(),
			_deleted: false,
		})) as Promise<Artist>;
	},
	update: async (id: EntityId, payload: UpdateArtistRequest) =>
		edit('artists', id, (old) => {
			if (!old || !('name' in old)) throw new Error('Artist not found');

			return {
				...old,
				...fields(payload),
				name: requiredName(payload.name ?? old.name, 'Name'),
				edited_at: new Date().toISOString(),
			};
		}) as Promise<Artist>,
	delete: async (id: EntityId) =>
		edit('artists', id, (old) => {
			if (!old) throw new Error('Artist not found');

			return { ...old, _deleted: true, edited_at: new Date().toISOString() };
		}),
};

function albumFields(old: Album | null, payload: CreateAlbumRequest | UpdateAlbumRequest): Album {
	const data = { ...old, ...fields(payload) };
	if (!data.artist_id || !uuid.test(data.artist_id)) throw new Error('Artist is required');

	if (old && old.artist_id !== data.artist_id) throw new Error('Cannot change the album artist');

	if (data.rating != null && !isAlbumRating(data.rating))
		throw new Error('Rating must be between 1 and 10');
	if (data.year != null && !isAlbumYear(data.year))
		throw new Error('Invalid release year');
	if (data.listened_at != null && !Number.isFinite(Date.parse(data.listened_at)))
		throw new Error('Invalid listening date');
	return {
		...data,
		title: requiredName(data.title ?? '', 'Title'),
		cover_url: data.cover_url ?? null,
		spotify_id: data.spotify_id ?? null,
		year: data.year ?? null,
		comment: data.comment ?? null,
		listened: data.listened ?? false,
		rating: data.listened ? (data.rating ?? null) : null,
		listened_at: data.listened ? (data.listened_at ?? null) : null,
		edited_at: new Date().toISOString(),
		_deleted: false,
	} as Album;
}

export const albumsLibrary = {
	create: async (payload: CreateAlbumRequest, createdAt?: string) => {
		const id = crypto.randomUUID();
		return edit('albums', id, () => ({
			...albumFields(null, payload),
			id,
			created_at: creationDate(createdAt),
		})) as Promise<Album>;
	},
	update: async (id: EntityId, payload: UpdateAlbumRequest) =>
		edit('albums', id, (old) => {
			if (!old || !('artist_id' in old)) throw new Error('Album not found');

			return albumFields(old, payload);
		}) as Promise<Album>,
	delete: async (id: EntityId) =>
		edit('albums', id, (old) => {
			if (!old) throw new Error('Album not found');

			return { ...old, _deleted: true, edited_at: new Date().toISOString() };
		}),
};

export async function resolveSyncProblem(problem: SyncProblem, restore: boolean) {
	await edit(
		problem.kind,
		problem.id,
		(old) => {
			if (!old) throw new Error('Record is unavailable. Export your library before clearing storage.');

			return { ...old, _deleted: !restore, edited_at: new Date().toISOString() };
		},
		true,
	);
	await refreshDatabaseSafely();
}
