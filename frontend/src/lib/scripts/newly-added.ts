import { writable } from 'svelte/store';
import type { EntityId } from './types';

type NewlyAdded = {
	albumIds: ReadonlySet<EntityId>;
	artistIds: ReadonlySet<EntityId>;
};

export const newlyAdded = writable<NewlyAdded>({
	albumIds: new Set(),
	artistIds: new Set(),
});

export function markAlbumAsNew(id: EntityId) {
	newlyAdded.update((value) => ({
		...value,
		albumIds: new Set(value.albumIds).add(id),
	}));
}

export function markArtistAsNew(id: EntityId) {
	newlyAdded.update((value) => ({
		...value,
		artistIds: new Set(value.artistIds).add(id),
	}));
}
