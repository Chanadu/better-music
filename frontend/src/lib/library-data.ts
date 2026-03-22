import { apiFetch, getAccessToken, getErrorMessage, getRefreshToken } from './api';

type EntitySyncState = 'synced' | 'pending';
type SyncResult = 'synced' | 'queued';

export type LibraryArtist = {
	id: number;
	name: string;
	cover_url?: string | null;
	spotify_id?: string | null;
	created_at: string;
	local_only?: boolean;
	sync_state?: EntitySyncState;
};

export type LibraryAlbum = {
	id: number;
	artist_id: number;
	title: string;
	cover_url?: string | null;
	year?: number | null;
	spotify_id?: string | null;
	listened: boolean;
	rating?: number | null;
	comment?: string | null;
	listened_at?: string | null;
	created_at: string;
	local_only?: boolean;
	sync_state?: EntitySyncState;
};

export type LibraryData = {
	albums: LibraryAlbum[];
	artists: LibraryArtist[];
};

export type CreateArtistInput = {
	name: string;
	cover_url?: string;
	spotify_id?: string;
};

export type UpdateArtistInput = {
	name: string;
	cover_url?: string;
	spotify_id?: string;
};

export type CreateAlbumInput = {
	title: string;
	artist_id: number;
	cover_url?: string;
	year?: number;
	spotify_id?: string;
	listened: boolean;
	rating?: number;
	listened_at?: string;
};

export type UpdateAlbumInput = {
	artist_id: number;
	title: string;
	cover_url?: string;
	year?: number;
	spotify_id?: string;
	listened: boolean;
	rating?: number;
	listened_at?: string;
};

export type MutationResult<T> = {
	entity: T;
	syncStatus: SyncResult;
};

export type DeleteResult = {
	syncStatus: SyncResult;
};

type ArtistPayload = {
	name: string;
	cover_url?: string;
	spotify_id?: string;
};

type AlbumPayload = {
	title: string;
	artist_id: number;
	cover_url?: string;
	year?: number;
	spotify_id?: string;
	listened: boolean;
	rating?: number;
	listened_at?: string;
};

type QueuedMutation =
	| { id: string; type: 'artist-create'; localId: number; payload: ArtistPayload }
	| { id: string; type: 'artist-update'; artistId: number; payload: UpdateArtistInput }
	| { id: string; type: 'artist-delete'; artistId: number }
	| { id: string; type: 'album-create'; localId: number; payload: AlbumPayload }
	| { id: string; type: 'album-update'; albumId: number; payload: UpdateAlbumInput }
	| { id: string; type: 'album-delete'; albumId: number; artistId: number };

const SNAPSHOT_STORAGE_KEY = 'better-music:library-snapshot';
const QUEUE_STORAGE_KEY = 'better-music:library-queue';
const SYNC_EVENT_NAME = 'library-data-changed';
const SYNC_MESSAGE_TYPE = 'better-music-sync-request';
const SYNC_REQUEST_TAG = 'better-music-library-sync';

let cachedLibraryData: LibraryData | null = null;
let libraryRequest: Promise<LibraryData> | null = null;
let backgroundRefreshRequest: Promise<void> | null = null;
let syncRequest: Promise<boolean> | null = null;
let stateMutationChain: Promise<void> = Promise.resolve();
let offlineSyncInitialized = false;

const isBrowser = (): boolean => typeof window !== 'undefined';
const isOnline = (): boolean => (typeof navigator === 'undefined' ? true : navigator.onLine);
const hasAuthTokens = (): boolean => Boolean(getAccessToken() || getRefreshToken());

const dispatchLibraryChange = (): void => {
	if (!isBrowser()) return;
	window.dispatchEvent(new CustomEvent(SYNC_EVENT_NAME));
};

const readStorageJSON = <T>(key: string, fallback: T): T => {
	if (!isBrowser()) return fallback;
	try {
		const raw = localStorage.getItem(key);
		return raw ? (JSON.parse(raw) as T) : fallback;
	} catch {
		return fallback;
	}
};

const writeStorageJSON = (key: string, value: unknown): void => {
	if (!isBrowser()) return;
	try {
		localStorage.setItem(key, JSON.stringify(value));
	} catch {
		// Ignore storage write failures and keep the in-memory view usable.
	}
};

const removeStorageKey = (key: string): void => {
	if (!isBrowser()) return;
	try {
		localStorage.removeItem(key);
	} catch {
		// Ignore storage remove failures.
	}
};

const normalizeArtist = (
	artist: Partial<LibraryArtist> & Pick<LibraryArtist, 'id' | 'name' | 'created_at'>,
): LibraryArtist => ({
	id: Number(artist.id),
	name: artist.name,
	cover_url: artist.cover_url ?? null,
	spotify_id: artist.spotify_id ?? null,
	created_at: artist.created_at,
	local_only: Boolean(artist.local_only),
	sync_state: artist.sync_state ?? 'synced',
});

const normalizeAlbum = (
	album: Partial<LibraryAlbum> & Pick<LibraryAlbum, 'id' | 'artist_id' | 'title' | 'listened' | 'created_at'>,
): LibraryAlbum => ({
	id: Number(album.id),
	artist_id: Number(album.artist_id),
	title: album.title,
	cover_url: album.cover_url ?? null,
	year: album.year ?? null,
	spotify_id: album.spotify_id ?? null,
	listened: Boolean(album.listened),
	rating: album.listened ? (album.rating ?? null) : null,
	comment: album.comment ?? null,
	listened_at: album.listened ? (album.listened_at ?? null) : null,
	created_at: album.created_at,
	local_only: Boolean(album.local_only),
	sync_state: album.sync_state ?? 'synced',
});

const cloneLibraryData = (data: LibraryData): LibraryData => ({
	artists: data.artists.map((artist) => normalizeArtist(artist)),
	albums: data.albums.map((album) => normalizeAlbum(album)),
});

const normalizeLibraryData = (data: Partial<LibraryData> | null | undefined): LibraryData => ({
	artists: Array.isArray(data?.artists) ? data.artists.map((artist) => normalizeArtist(artist as LibraryArtist)) : [],
	albums: Array.isArray(data?.albums) ? data.albums.map((album) => normalizeAlbum(album as LibraryAlbum)) : [],
});

const mergeLibraryData = (remoteData: LibraryData, localData: LibraryData | null): LibraryData => {
	if (!localData) return cloneLibraryData(remoteData);

	const artistsById = new Map(remoteData.artists.map((artist) => [artist.id, normalizeArtist(artist)]));
	for (const localArtist of localData.artists) {
		if (localArtist.local_only || localArtist.sync_state === 'pending') {
			artistsById.set(localArtist.id, normalizeArtist(localArtist));
		}
	}

	const albumsById = new Map(remoteData.albums.map((album) => [album.id, normalizeAlbum(album)]));
	for (const localAlbum of localData.albums) {
		if (localAlbum.local_only || localAlbum.sync_state === 'pending') {
			albumsById.set(localAlbum.id, normalizeAlbum(localAlbum));
		}
	}

	return {
		artists: [...artistsById.values()],
		albums: [...albumsById.values()],
	};
};

const areLibraryDataEqual = (left: LibraryData | null, right: LibraryData | null): boolean => {
	if (!left && !right) return true;
	if (!left || !right) return false;
	return JSON.stringify(left) === JSON.stringify(right);
};

const readStoredLibraryData = (): LibraryData | null => {
	const stored = readStorageJSON<LibraryData | null>(SNAPSHOT_STORAGE_KEY, null);
	return stored ? normalizeLibraryData(stored) : null;
};

const writeStoredLibraryData = (data: LibraryData | null): void => {
	if (!data) {
		removeStorageKey(SNAPSHOT_STORAGE_KEY);
		cachedLibraryData = null;
		return;
	}

	const normalized = normalizeLibraryData(data);
	writeStorageJSON(SNAPSHOT_STORAGE_KEY, normalized);
	cachedLibraryData = cloneLibraryData(normalized);
};

const readQueuedMutations = (): QueuedMutation[] => {
	return readStorageJSON<QueuedMutation[]>(QUEUE_STORAGE_KEY, []);
};

const writeQueuedMutations = (queue: QueuedMutation[]): void => {
	if (!queue.length) {
		removeStorageKey(QUEUE_STORAGE_KEY);
		return;
	}
	writeStorageJSON(QUEUE_STORAGE_KEY, queue);
};

const withStateLock = async <T>(work: () => T | Promise<T>): Promise<T> => {
	let release = () => undefined;
	const wait = new Promise<void>((resolve) => {
		release = resolve;
	});
	const previous = stateMutationChain;
	stateMutationChain = previous.then(
		() => wait,
		() => wait,
	);
	await previous;
	try {
		return await work();
	} finally {
		release();
	}
};

const nextTempId = (data: LibraryData): number => {
	let smallestID = 0;
	for (const artist of data.artists) smallestID = Math.min(smallestID, artist.id);
	for (const album of data.albums) smallestID = Math.min(smallestID, album.id);
	return smallestID - 1;
};

const isLocalEntity = (id: number, localOnly?: boolean): boolean => localOnly || id < 0;

const createQueueID = (): string => {
	if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
		return crypto.randomUUID();
	}
	return `${Date.now()}-${Math.random().toString(36).slice(2)}`;
};

const findLastIndex = <T>(items: T[], predicate: (item: T) => boolean): number => {
	for (let index = items.length - 1; index >= 0; index -= 1) {
		if (predicate(items[index])) return index;
	}
	return -1;
};

const sanitizeArtistPayload = (input: CreateArtistInput | UpdateArtistInput): ArtistPayload => {
	const name = input.name.trim();
	if (!name) throw new Error('Artist name is required.');

	const payload: ArtistPayload = { name };
	if ('cover_url' in input && input.cover_url?.trim()) payload.cover_url = input.cover_url.trim();
	if ('spotify_id' in input && input.spotify_id?.trim()) payload.spotify_id = input.spotify_id.trim();
	return payload;
};

const sanitizeAlbumPayload = (input: CreateAlbumInput | UpdateAlbumInput): AlbumPayload => {
	const title = input.title.trim();
	const artistID = Number(input.artist_id);
	if (!title || !Number.isFinite(artistID) || artistID === 0) {
		throw new Error('Album title and artist are required.');
	}

	const payload: AlbumPayload = {
		title,
		artist_id: artistID,
		listened: Boolean(input.listened),
	};

	if ('cover_url' in input && input.cover_url?.trim()) payload.cover_url = input.cover_url.trim();
	if ('spotify_id' in input && input.spotify_id?.trim()) payload.spotify_id = input.spotify_id.trim();
	if ('year' in input && typeof input.year === 'number' && Number.isFinite(input.year)) payload.year = input.year;

	if (payload.listened) {
		const rating = Number(input.rating);
		if (!Number.isFinite(rating)) {
			throw new Error('Rating is required when album is marked as listened.');
		}
		payload.rating = Math.min(10, Math.max(0, rating));
		if ('listened_at' in input && input.listened_at) payload.listened_at = input.listened_at;
	}

	return payload;
};

const createLocalArtist = (id: number, payload: ArtistPayload): LibraryArtist =>
	normalizeArtist({
		id,
		name: payload.name,
		cover_url: payload.cover_url ?? null,
		spotify_id: payload.spotify_id ?? null,
		created_at: new Date().toISOString(),
		local_only: true,
		sync_state: 'pending',
	});

const createLocalAlbum = (id: number, payload: AlbumPayload): LibraryAlbum =>
	normalizeAlbum({
		id,
		artist_id: payload.artist_id,
		title: payload.title,
		cover_url: payload.cover_url ?? null,
		year: payload.year ?? null,
		spotify_id: payload.spotify_id ?? null,
		listened: payload.listened,
		rating: payload.listened ? (payload.rating ?? null) : null,
		listened_at: payload.listened ? (payload.listened_at ?? null) : null,
		created_at: new Date().toISOString(),
		local_only: true,
		sync_state: 'pending',
	});

const applyArtistUpdate = (artist: LibraryArtist, payload: ArtistPayload): LibraryArtist =>
	normalizeArtist({
		...artist,
		name: payload.name,
		cover_url: payload.cover_url ?? artist.cover_url ?? null,
		spotify_id: payload.spotify_id ?? artist.spotify_id ?? null,
		sync_state: 'pending',
	});

const applyAlbumUpdate = (album: LibraryAlbum, payload: AlbumPayload): LibraryAlbum =>
	normalizeAlbum({
		...album,
		title: payload.title,
		artist_id: payload.artist_id,
		cover_url: payload.cover_url ?? album.cover_url ?? null,
		year: payload.year ?? album.year ?? null,
		spotify_id: payload.spotify_id ?? album.spotify_id ?? null,
		listened: payload.listened,
		rating: payload.listened ? (payload.rating ?? album.rating ?? null) : null,
		listened_at: payload.listened ? (payload.listened_at ?? album.listened_at ?? null) : null,
		sync_state: 'pending',
	});

const buildAlbumUpdateBody = (payload: AlbumPayload): Record<string, unknown> => {
	const body: Record<string, unknown> = {
		artist_id: payload.artist_id,
		title: payload.title,
		listened: payload.listened,
	};
	if (payload.cover_url) body.cover_url = payload.cover_url;
	if (typeof payload.year === 'number') body.year = payload.year;
	if (payload.spotify_id) body.spotify_id = payload.spotify_id;
	if (payload.listened) {
		body.rating = payload.rating;
		if (payload.listened_at) body.listened_at = payload.listened_at;
	}
	return body;
};

const needsAlbumDetailsUpdate = (payload: AlbumPayload): boolean =>
	Boolean(payload.listened || payload.cover_url || typeof payload.year === 'number');

const registerServiceWorkerSync = async (): Promise<void> => {
	if (!isBrowser() || !('serviceWorker' in navigator)) return;
	try {
		const registration = await navigator.serviceWorker.ready;
		const syncManager = (
			registration as ServiceWorkerRegistration & {
				sync?: { register: (tag: string) => Promise<void> };
			}
		).sync;
		await syncManager?.register(SYNC_REQUEST_TAG);
	} catch {
		// Background sync is best-effort only.
	}
};

const settleMutationSync = async (): Promise<SyncResult> => {
	if (!hasAuthTokens() || !isOnline()) {
		await registerServiceWorkerSync();
		return 'queued';
	}

	try {
		await syncLibraryMutations();
	} catch {
		// Keep the optimistic local state and let the queue retry later.
	}

	if (readQueuedMutations().length > 0) {
		await registerServiceWorkerSync();
		return 'queued';
	}

	return 'synced';
};

const remapQueuedMutation = (
	mutation: QueuedMutation,
	artistMap: Map<number, number>,
	albumMap: Map<number, number>,
): QueuedMutation => {
	switch (mutation.type) {
		case 'artist-create':
			return mutation;
		case 'artist-update':
			return artistMap.has(mutation.artistId) ?
					{ ...mutation, artistId: artistMap.get(mutation.artistId) ?? mutation.artistId }
				:	mutation;
		case 'artist-delete':
			return artistMap.has(mutation.artistId) ?
					{ ...mutation, artistId: artistMap.get(mutation.artistId) ?? mutation.artistId }
				:	mutation;
		case 'album-create':
			return artistMap.has(mutation.payload.artist_id) ?
					{
						...mutation,
						payload: {
							...mutation.payload,
							artist_id: artistMap.get(mutation.payload.artist_id) ?? mutation.payload.artist_id,
						},
					}
				:	mutation;
		case 'album-update': {
			const nextAlbumID = albumMap.get(mutation.albumId) ?? mutation.albumId;
			const nextArtistID = artistMap.get(mutation.payload.artist_id) ?? mutation.payload.artist_id;
			return {
				...mutation,
				albumId: nextAlbumID,
				payload: { ...mutation.payload, artist_id: nextArtistID },
			};
		}
		case 'album-delete':
			return {
				...mutation,
				albumId: albumMap.get(mutation.albumId) ?? mutation.albumId,
				artistId: artistMap.get(mutation.artistId) ?? mutation.artistId,
			};
	}
};

const fetchLibraryDataFromNetwork = async (): Promise<LibraryData> => {
	const [albumsResponse, artistsResponse] = await Promise.all([
		apiFetch('/api/albums', { method: 'GET' }, { auth: true }),
		apiFetch('/api/artists', { method: 'GET' }, { auth: true }),
	]);

	if (!albumsResponse.ok || !artistsResponse.ok) {
		throw new Error('Failed to load library data.');
	}

	const [albums, artists] = await Promise.all([
		albumsResponse.json() as Promise<LibraryAlbum[]>,
		artistsResponse.json() as Promise<LibraryArtist[]>,
	]);

	return normalizeLibraryData({
		albums: albums.map((album) => ({ ...album, local_only: false, sync_state: 'synced' })),
		artists: artists.map((artist) => ({ ...artist, local_only: false, sync_state: 'synced' })),
	});
};

const refreshLibraryFromNetwork = async (): Promise<void> => {
	if (backgroundRefreshRequest || !hasAuthTokens() || !isOnline()) return;

	backgroundRefreshRequest = (async () => {
		const previous = cachedLibraryData ?? readStoredLibraryData();
		const remoteData = await fetchLibraryDataFromNetwork();
		const updated = await withStateLock(() => {
			const localSnapshot = readStoredLibraryData();
			const nextData =
				readQueuedMutations().length > 0 ? mergeLibraryData(remoteData, localSnapshot) : remoteData;
			writeStoredLibraryData(nextData);
			return true;
		});

		const latest = cachedLibraryData ?? readStoredLibraryData();
		if (updated && !areLibraryDataEqual(previous, latest)) {
			dispatchLibraryChange();
		}
	})()
		.catch(() => undefined)
		.finally(() => {
			backgroundRefreshRequest = null;
		});

	await backgroundRefreshRequest;
};

const updateQueueForArtistChange = (queue: QueuedMutation[], oldID: number, newID: number): QueuedMutation[] =>
	queue.map((mutation) => remapQueuedMutation(mutation, new Map([[oldID, newID]]), new Map()));

const updateQueueForAlbumChange = (queue: QueuedMutation[], oldID: number, newID: number): QueuedMutation[] =>
	queue.map((mutation) => remapQueuedMutation(mutation, new Map(), new Map([[oldID, newID]])));

const processArtistCreateMutation = async (
	mutation: Extract<QueuedMutation, { type: 'artist-create' }>,
): Promise<void> => {
	const response = await apiFetch(
		'/api/artists',
		{
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify(mutation.payload),
		},
		{ auth: true },
	);

	if (!response.ok) {
		throw new Error(await getErrorMessage(response, 'Failed to sync artist.'));
	}

	const createdArtist = normalizeArtist(
		((await response.json()) as LibraryArtist) ?? createLocalArtist(mutation.localId, mutation.payload),
	);

	await withStateLock(() => {
		const snapshot = readStoredLibraryData() ?? { albums: [], artists: [] };
		const queue = readQueuedMutations().filter((item) => item.id !== mutation.id);
		const currentArtist = snapshot.artists.find((artist) => artist.id === mutation.localId);
		const syncedArtist = normalizeArtist({
			...currentArtist,
			...createdArtist,
			id: createdArtist.id,
			name: createdArtist.name || currentArtist?.name || mutation.payload.name,
			cover_url: createdArtist.cover_url ?? currentArtist?.cover_url ?? mutation.payload.cover_url ?? null,
			spotify_id: createdArtist.spotify_id ?? currentArtist?.spotify_id ?? mutation.payload.spotify_id ?? null,
			local_only: false,
			sync_state: 'synced',
		});

		snapshot.artists = snapshot.artists.map((artist) => (artist.id === mutation.localId ? syncedArtist : artist));
		snapshot.albums = snapshot.albums.map((album) =>
			album.artist_id === mutation.localId ? normalizeAlbum({ ...album, artist_id: syncedArtist.id }) : album,
		);

		writeStoredLibraryData(snapshot);
		writeQueuedMutations(updateQueueForArtistChange(queue, mutation.localId, syncedArtist.id));
	});
};

const processArtistUpdateMutation = async (
	mutation: Extract<QueuedMutation, { type: 'artist-update' }>,
): Promise<void> => {
	const response = await apiFetch(
		`/api/artists/${mutation.artistId}`,
		{
			method: 'PUT',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify(mutation.payload),
		},
		{ auth: true },
	);

	if (!response.ok) {
		throw new Error(await getErrorMessage(response, 'Failed to sync artist update.'));
	}

	await withStateLock(() => {
		const snapshot = readStoredLibraryData();
		if (snapshot) {
			snapshot.artists = snapshot.artists.map((artist) =>
				artist.id === mutation.artistId ?
					normalizeArtist({ ...artist, local_only: false, sync_state: 'synced' })
				:	artist,
			);
			writeStoredLibraryData(snapshot);
		}
		writeQueuedMutations(readQueuedMutations().filter((item) => item.id !== mutation.id));
	});
};

const processArtistDeleteMutation = async (
	mutation: Extract<QueuedMutation, { type: 'artist-delete' }>,
): Promise<void> => {
	const response = await apiFetch(`/api/artists/${mutation.artistId}`, { method: 'DELETE' }, { auth: true });
	if (!response.ok) {
		throw new Error(await getErrorMessage(response, 'Failed to sync artist delete.'));
	}

	await withStateLock(() => {
		writeQueuedMutations(readQueuedMutations().filter((item) => item.id !== mutation.id));
	});
};

const processAlbumCreateMutation = async (
	mutation: Extract<QueuedMutation, { type: 'album-create' }>,
): Promise<void> => {
	const createResponse = await apiFetch(
		'/api/albums',
		{
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({
				title: mutation.payload.title,
				artist_id: mutation.payload.artist_id,
				...(mutation.payload.spotify_id ? { spotify_id: mutation.payload.spotify_id } : {}),
			}),
		},
		{ auth: true },
	);

	if (!createResponse.ok) {
		throw new Error(await getErrorMessage(createResponse, 'Failed to sync album.'));
	}

	const createdAlbum = (await createResponse.json()) as LibraryAlbum;

	if (needsAlbumDetailsUpdate(mutation.payload)) {
		const updateResponse = await apiFetch(
			`/api/albums/${createdAlbum.id}`,
			{
				method: 'PUT',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(buildAlbumUpdateBody(mutation.payload)),
			},
			{ auth: true },
		);

		if (!updateResponse.ok) {
			throw new Error(await getErrorMessage(updateResponse, 'Failed to sync album details.'));
		}
	}

	await withStateLock(() => {
		const snapshot = readStoredLibraryData() ?? { albums: [], artists: [] };
		const queue = readQueuedMutations().filter((item) => item.id !== mutation.id);
		const localAlbum = snapshot.albums.find((album) => album.id === mutation.localId);
		const syncedAlbum = normalizeAlbum({
			...localAlbum,
			...createdAlbum,
			id: createdAlbum.id,
			artist_id: mutation.payload.artist_id,
			title: mutation.payload.title,
			cover_url: mutation.payload.cover_url ?? localAlbum?.cover_url ?? null,
			year: mutation.payload.year ?? localAlbum?.year ?? null,
			spotify_id: mutation.payload.spotify_id ?? localAlbum?.spotify_id ?? null,
			listened: mutation.payload.listened,
			rating: mutation.payload.listened ? (mutation.payload.rating ?? localAlbum?.rating ?? null) : null,
			listened_at:
				mutation.payload.listened ? (mutation.payload.listened_at ?? localAlbum?.listened_at ?? null) : null,
			local_only: false,
			sync_state: 'synced',
		});

		snapshot.albums = snapshot.albums.map((album) => (album.id === mutation.localId ? syncedAlbum : album));
		writeStoredLibraryData(snapshot);
		writeQueuedMutations(updateQueueForAlbumChange(queue, mutation.localId, syncedAlbum.id));
	});
};

const processAlbumUpdateMutation = async (
	mutation: Extract<QueuedMutation, { type: 'album-update' }>,
): Promise<void> => {
	const response = await apiFetch(
		`/api/albums/${mutation.albumId}`,
		{
			method: 'PUT',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify(buildAlbumUpdateBody(sanitizeAlbumPayload(mutation.payload))),
		},
		{ auth: true },
	);

	if (!response.ok) {
		throw new Error(await getErrorMessage(response, 'Failed to sync album update.'));
	}

	await withStateLock(() => {
		const snapshot = readStoredLibraryData();
		if (snapshot) {
			snapshot.albums = snapshot.albums.map((album) =>
				album.id === mutation.albumId ?
					normalizeAlbum({ ...album, local_only: false, sync_state: 'synced' })
				:	album,
			);
			writeStoredLibraryData(snapshot);
		}
		writeQueuedMutations(readQueuedMutations().filter((item) => item.id !== mutation.id));
	});
};

const processAlbumDeleteMutation = async (
	mutation: Extract<QueuedMutation, { type: 'album-delete' }>,
): Promise<void> => {
	const response = await apiFetch(
		`/api/albums/${mutation.albumId}`,
		{
			method: 'DELETE',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ artist_id: mutation.artistId }),
		},
		{ auth: true },
	);

	if (!response.ok) {
		throw new Error(await getErrorMessage(response, 'Failed to sync album delete.'));
	}

	await withStateLock(() => {
		writeQueuedMutations(readQueuedMutations().filter((item) => item.id !== mutation.id));
	});
};

const processQueuedMutation = async (mutation: QueuedMutation): Promise<void> => {
	switch (mutation.type) {
		case 'artist-create':
			await processArtistCreateMutation(mutation);
			return;
		case 'artist-update':
			await processArtistUpdateMutation(mutation);
			return;
		case 'artist-delete':
			await processArtistDeleteMutation(mutation);
			return;
		case 'album-create':
			await processAlbumCreateMutation(mutation);
			return;
		case 'album-update':
			await processAlbumUpdateMutation(mutation);
			return;
		case 'album-delete':
			await processAlbumDeleteMutation(mutation);
			return;
	}
};

export const loadLibraryData = async (force = false): Promise<LibraryData> => {
	if (!force && cachedLibraryData) {
		void refreshLibraryFromNetwork();
		return cachedLibraryData;
	}

	if (libraryRequest) return libraryRequest;

	libraryRequest = (async () => {
		const stored = readStoredLibraryData();
		if (!force && stored) {
			cachedLibraryData = cloneLibraryData(stored);
			void refreshLibraryFromNetwork();
			void syncLibraryMutations();
			return cachedLibraryData;
		}

		if (stored && readQueuedMutations().length > 0) {
			cachedLibraryData = cloneLibraryData(stored);
			void syncLibraryMutations();
			return cachedLibraryData;
		}

		if (hasAuthTokens() && isOnline()) {
			const remoteData = await fetchLibraryDataFromNetwork();
			await withStateLock(() => {
				const localSnapshot = readStoredLibraryData();
				const nextData =
					readQueuedMutations().length > 0 ? mergeLibraryData(remoteData, localSnapshot) : remoteData;
				writeStoredLibraryData(nextData);
			});
			return cachedLibraryData ?? mergeLibraryData(remoteData, readStoredLibraryData());
		}

		if (stored) {
			cachedLibraryData = cloneLibraryData(stored);
			return cachedLibraryData;
		}

		throw new Error('Library data is not available offline yet.');
	})().finally(() => {
		libraryRequest = null;
	});

	return libraryRequest;
};

export const warmLibraryData = (): void => {
	if (!hasAuthTokens()) return;
	if (!cachedLibraryData) {
		void loadLibraryData().catch(() => undefined);
		return;
	}
	void refreshLibraryFromNetwork();
	void syncLibraryMutations();
};

export const hasCachedLibraryData = (): boolean => Boolean(cachedLibraryData || readStoredLibraryData());

export const invalidateLibraryData = (): void => {
	cachedLibraryData = null;
	libraryRequest = null;
};

export const createArtistRecord = async (input: CreateArtistInput): Promise<MutationResult<LibraryArtist>> => {
	const payload = sanitizeArtistPayload(input);
	const artist = await withStateLock(() => {
		const snapshot = readStoredLibraryData() ?? { albums: [], artists: [] };
		const queue = readQueuedMutations();
		const localID = nextTempId(snapshot);
		const nextArtist = createLocalArtist(localID, payload);
		snapshot.artists = [...snapshot.artists, nextArtist];
		writeStoredLibraryData(snapshot);
		writeQueuedMutations([...queue, { id: createQueueID(), type: 'artist-create', localId: localID, payload }]);
		return nextArtist;
	});

	dispatchLibraryChange();
	return { entity: artist, syncStatus: await settleMutationSync() };
};

export const updateArtistRecord = async (
	id: number,
	input: UpdateArtistInput,
): Promise<MutationResult<LibraryArtist>> => {
	const payload = sanitizeArtistPayload(input);
	const artist = await withStateLock(() => {
		const snapshot = readStoredLibraryData();
		if (!snapshot) throw new Error('Artist not found.');

		const artistIndex = snapshot.artists.findIndex((item) => item.id === id);
		if (artistIndex < 0) throw new Error('Artist not found.');

		const currentArtist = snapshot.artists[artistIndex];
		const nextArtist = applyArtistUpdate(currentArtist, payload);
		const queue = readQueuedMutations();
		const nextQueue = [...queue];
		const localCreateIndex = findLastIndex(
			nextQueue,
			(item) => item.type === 'artist-create' && item.localId === id,
		);

		if (isLocalEntity(id, currentArtist.local_only) && localCreateIndex >= 0) {
			const localCreate = nextQueue[localCreateIndex] as Extract<QueuedMutation, { type: 'artist-create' }>;
			nextQueue[localCreateIndex] = { ...localCreate, payload: { ...localCreate.payload, ...payload } };
		} else if (isLocalEntity(id, currentArtist.local_only)) {
			nextQueue.push({
				id: createQueueID(),
				type: 'artist-create',
				localId: id,
				payload: {
					name: nextArtist.name,
					...(nextArtist.cover_url ? { cover_url: nextArtist.cover_url } : {}),
					...(nextArtist.spotify_id ? { spotify_id: nextArtist.spotify_id } : {}),
				},
			});
		} else {
			const existingUpdateIndex = findLastIndex(
				nextQueue,
				(item) => item.type === 'artist-update' && item.artistId === id,
			);
			if (existingUpdateIndex >= 0) {
				const existing = nextQueue[existingUpdateIndex] as Extract<QueuedMutation, { type: 'artist-update' }>;
				nextQueue[existingUpdateIndex] = { ...existing, payload };
			} else {
				nextQueue.push({ id: createQueueID(), type: 'artist-update', artistId: id, payload });
			}
		}

		snapshot.artists[artistIndex] = nextArtist;
		writeStoredLibraryData(snapshot);
		writeQueuedMutations(nextQueue);
		return nextArtist;
	});

	dispatchLibraryChange();
	return { entity: artist, syncStatus: await settleMutationSync() };
};

export const deleteArtistRecord = async (id: number): Promise<DeleteResult> => {
	await withStateLock(() => {
		const snapshot = readStoredLibraryData();
		if (!snapshot) return;

		const artist = snapshot.artists.find((item) => item.id === id);
		if (!artist) return;

		const removedAlbumIDs = new Set(
			snapshot.albums.filter((album) => album.artist_id === id).map((album) => album.id),
		);
		snapshot.artists = snapshot.artists.filter((item) => item.id !== id);
		snapshot.albums = snapshot.albums.filter((album) => album.artist_id !== id);

		let nextQueue = readQueuedMutations().filter((mutation) => {
			if (mutation.type === 'artist-create' && mutation.localId === id) return false;
			if (mutation.type === 'artist-update' && mutation.artistId === id) return false;
			if (mutation.type === 'artist-delete' && mutation.artistId === id) return false;
			if (mutation.type === 'album-create' && removedAlbumIDs.has(mutation.localId)) return false;
			if (mutation.type === 'album-update' && removedAlbumIDs.has(mutation.albumId)) return false;
			if (mutation.type === 'album-delete' && removedAlbumIDs.has(mutation.albumId)) return false;
			return true;
		});

		if (!isLocalEntity(id, artist.local_only)) {
			nextQueue.push({ id: createQueueID(), type: 'artist-delete', artistId: id });
		}

		writeStoredLibraryData(snapshot);
		writeQueuedMutations(nextQueue);
	});

	dispatchLibraryChange();
	return { syncStatus: await settleMutationSync() };
};

export const createAlbumRecord = async (input: CreateAlbumInput): Promise<MutationResult<LibraryAlbum>> => {
	const payload = sanitizeAlbumPayload(input);
	const album = await withStateLock(() => {
		const snapshot = readStoredLibraryData() ?? { albums: [], artists: [] };
		const queue = readQueuedMutations();
		const localID = nextTempId(snapshot);
		const nextAlbum = createLocalAlbum(localID, payload);
		snapshot.albums = [...snapshot.albums, nextAlbum];
		writeStoredLibraryData(snapshot);
		writeQueuedMutations([...queue, { id: createQueueID(), type: 'album-create', localId: localID, payload }]);
		return nextAlbum;
	});

	dispatchLibraryChange();
	return { entity: album, syncStatus: await settleMutationSync() };
};

export const updateAlbumRecord = async (id: number, input: UpdateAlbumInput): Promise<MutationResult<LibraryAlbum>> => {
	const payload = sanitizeAlbumPayload(input);
	const album = await withStateLock(() => {
		const snapshot = readStoredLibraryData();
		if (!snapshot) throw new Error('Album not found.');

		const albumIndex = snapshot.albums.findIndex((item) => item.id === id);
		if (albumIndex < 0) throw new Error('Album not found.');

		const currentAlbum = snapshot.albums[albumIndex];
		const nextAlbum = applyAlbumUpdate(currentAlbum, payload);
		const queue = readQueuedMutations();
		const nextQueue = [...queue];
		const localCreateIndex = findLastIndex(
			nextQueue,
			(item) => item.type === 'album-create' && item.localId === id,
		);

		if (isLocalEntity(id, currentAlbum.local_only) && localCreateIndex >= 0) {
			const localCreate = nextQueue[localCreateIndex] as Extract<QueuedMutation, { type: 'album-create' }>;
			nextQueue[localCreateIndex] = { ...localCreate, payload: { ...localCreate.payload, ...payload } };
		} else if (isLocalEntity(id, currentAlbum.local_only)) {
			nextQueue.push({
				id: createQueueID(),
				type: 'album-create',
				localId: id,
				payload: {
					title: nextAlbum.title,
					artist_id: nextAlbum.artist_id,
					...(nextAlbum.cover_url ? { cover_url: nextAlbum.cover_url } : {}),
					...(typeof nextAlbum.year === 'number' ? { year: nextAlbum.year } : {}),
					...(nextAlbum.spotify_id ? { spotify_id: nextAlbum.spotify_id } : {}),
					listened: nextAlbum.listened,
					...(nextAlbum.listened && nextAlbum.rating != null ? { rating: nextAlbum.rating } : {}),
					...(nextAlbum.listened && nextAlbum.listened_at ? { listened_at: nextAlbum.listened_at } : {}),
				},
			});
		} else {
			const existingUpdateIndex = findLastIndex(
				nextQueue,
				(item) => item.type === 'album-update' && item.albumId === id,
			);
			if (existingUpdateIndex >= 0) {
				const existing = nextQueue[existingUpdateIndex] as Extract<QueuedMutation, { type: 'album-update' }>;
				nextQueue[existingUpdateIndex] = { ...existing, payload };
			} else {
				nextQueue.push({ id: createQueueID(), type: 'album-update', albumId: id, payload });
			}
		}

		snapshot.albums[albumIndex] = nextAlbum;
		writeStoredLibraryData(snapshot);
		writeQueuedMutations(nextQueue);
		return nextAlbum;
	});

	dispatchLibraryChange();
	return { entity: album, syncStatus: await settleMutationSync() };
};

export const deleteAlbumRecord = async (id: number, artistId: number): Promise<DeleteResult> => {
	await withStateLock(() => {
		const snapshot = readStoredLibraryData();
		if (!snapshot) return;

		const album = snapshot.albums.find((item) => item.id === id);
		if (!album) return;

		snapshot.albums = snapshot.albums.filter((item) => item.id !== id);
		let nextQueue = readQueuedMutations().filter((mutation) => {
			if (mutation.type === 'album-create' && mutation.localId === id) return false;
			if (mutation.type === 'album-update' && mutation.albumId === id) return false;
			if (mutation.type === 'album-delete' && mutation.albumId === id) return false;
			return true;
		});

		if (!isLocalEntity(id, album.local_only)) {
			nextQueue.push({ id: createQueueID(), type: 'album-delete', albumId: id, artistId });
		}

		writeStoredLibraryData(snapshot);
		writeQueuedMutations(nextQueue);
	});

	dispatchLibraryChange();
	return { syncStatus: await settleMutationSync() };
};

export const syncLibraryMutations = async (): Promise<boolean> => {
	if (syncRequest) return syncRequest;
	if (!hasAuthTokens() || !isOnline()) return false;

	syncRequest = (async () => {
		let queue = readQueuedMutations();
		if (!queue.length) {
			await refreshLibraryFromNetwork();
			return true;
		}

		while (queue.length && hasAuthTokens() && isOnline()) {
			const mutation = queue[0];
			try {
				await processQueuedMutation(mutation);
				dispatchLibraryChange();
			} catch (error) {
				console.error('[Offline Sync] Failed to process queued mutation:', error);
				await registerServiceWorkerSync();
				return false;
			}
			queue = readQueuedMutations();
		}

		if (!queue.length) {
			await refreshLibraryFromNetwork().catch(() => undefined);
		}

		return readQueuedMutations().length === 0;
	})().finally(() => {
		syncRequest = null;
	});

	return syncRequest;
};

export const initializeOfflineLibrarySync = (): void => {
	if (!isBrowser() || offlineSyncInitialized) return;
	offlineSyncInitialized = true;

	window.addEventListener('online', () => {
		void syncLibraryMutations();
	});

	window.addEventListener('storage', (event) => {
		if (event.key !== SNAPSHOT_STORAGE_KEY && event.key !== QUEUE_STORAGE_KEY) return;
		cachedLibraryData = readStoredLibraryData();
		dispatchLibraryChange();
	});

	if ('serviceWorker' in navigator) {
		navigator.serviceWorker.addEventListener('message', (event) => {
			if (event.data?.type === SYNC_MESSAGE_TYPE) {
				void syncLibraryMutations();
			}
		});
	}

	void syncLibraryMutations();
	void refreshLibraryFromNetwork();
};
