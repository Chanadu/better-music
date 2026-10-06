import type {
	EntityId,
	LibraryUpdate,
	LocalAlbum,
	LocalArtist,
	PendingMutation,
	SuccessfulMutationResponse,
} from './types';

export const compareMutationOrder = (left: PendingMutation, right: PendingMutation) =>
	(left.sequence ?? left.createdAt) - (right.sequence ?? right.createdAt);

const definedPayload = (payload: object) =>
	Object.fromEntries(Object.entries(payload).filter(([, value]) => value !== undefined));

const replay = <T extends LocalArtist | LocalAlbum>(record: T, mutations: PendingMutation[]): T | null => {
	let result = record;

	for (const mutation of mutations) {
		if (mutation.operation === 'delete') return null;
		if (mutation.operation !== 'update') throw new Error('Unexpected later create for the same record');

		result = { ...result, ...definedPayload(mutation.payload) };

		if (mutation.entity === 'album' && mutation.payload.listened === false) {
			result = { ...result, rating: null, listened_at: null };
		}
	}
	return result;
};

export const reconcileLibraryUpdate = (
	current: LibraryUpdate,
	mutation: PendingMutation,
	response: SuccessfulMutationResponse,
): LibraryUpdate => {
	if (response.entity !== mutation.entity || response.operation !== mutation.operation) {
		throw new Error('Response does not match the queued operation');
	}

	const ordered = [...current.mutations].sort(compareMutationOrder);
	const position = ordered.findIndex((item) => item.mutationId === mutation.mutationId);

	if (position < 0) return current;

	const oldId = mutation.entityId;
	const newId: EntityId = response.operation === 'delete' ? oldId : response.record.id;

	if (response.operation !== 'delete') {
		if (
			!Number.isSafeInteger(response.record.id) ||
			response.record.id <= 0 ||
			!Number.isSafeInteger(response.record.version) ||
			response.record.version <= 0
		) {
			throw new Error('Server record must have a positive ID and version');
		}

		if (mutation.operation === 'update' && newId !== oldId) {
			throw new Error('Server record ID does not match the queued update');
		}
	}

	const laterIds = new Set(ordered.slice(position + 1).map((item) => item.mutationId));
	const sameRecord = (item: PendingMutation) => item.entity === mutation.entity && item.entityId === oldId;

	if (ordered.slice(0, position).some(sameRecord)) {
		throw new Error('Earlier operations for this record must be reconciled first');
	}

	const mutations = current.mutations
		.filter((item) => item.mutationId !== mutation.mutationId)
		.map((item): PendingMutation => {
			const replaceEntityId = sameRecord(item) && oldId !== newId;

			const replaceArtistId =
				mutation.entity === 'artist' &&
				oldId !== newId &&
				item.entity === 'album' &&
				item.payload.artist_id === oldId;

			const replaceVersion =
				sameRecord(item) &&
				laterIds.has(item.mutationId) &&
				response.operation !== 'delete' &&
				item.baseVersion !== response.record.version;

			if ((replaceEntityId || replaceArtistId || replaceVersion) && item.attempts > 0) {
				throw new Error('Cannot rewrite an already attempted dependent operation');
			}

			let updated = { ...item };
			if (replaceEntityId) updated.entityId = newId;

			if (replaceVersion) updated.baseVersion = response.record.version;

			if (replaceArtistId && updated.entity === 'album') {
				if (updated.operation === 'create') {
					updated = { ...updated, payload: { ...updated.payload, artist_id: newId } };
				} else if (updated.operation === 'update') {
					updated = { ...updated, payload: { ...updated.payload, artist_id: newId } };
				} else {
					updated = { ...updated, payload: { artist_id: newId } };
				}
			}

			return updated;
		});

	const later = mutations
		.filter((item) => laterIds.has(item.mutationId) && item.entity === mutation.entity && item.entityId === newId)
		.sort(compareMutationOrder);

	let library = { ...current.library };

	if (mutation.entity === 'artist') {
		library.albums = library.albums.map((album) =>
			album.artist_id === oldId ? { ...album, artist_id: newId } : album,
		);

		const remaining = library.artists.filter((artist) => artist.id !== oldId && artist.id !== newId);

		if (response.operation === 'delete') {
			library.artists = remaining;
			library.albums = library.albums.filter((album) => album.artist_id !== oldId);
		} else if (response.entity === 'artist') {
			const record = replay(response.record, later);
			library.artists = record ? [record, ...remaining] : remaining;
		}
	} else {
		const remaining = library.albums.filter((album) => album.id !== oldId && album.id !== newId);

		if (response.operation === 'delete') library.albums = remaining;
		else if (response.entity === 'album') {
			const record = replay(response.record, later);

			library.albums = record ? [record, ...remaining] : remaining;
		}
	}

	return { library, mutations };
};
