import { writable } from 'svelte/store';

type NewlyAdded = {
	albumIds: ReadonlySet<number>;
	artistIds: ReadonlySet<number>;
};

export const newlyAdded = writable<NewlyAdded>({
	albumIds: new Set(),
	artistIds: new Set(),
});

export function markAlbumAsNew(id: number) {
	newlyAdded.update((value) => ({
		...value,
		albumIds: new Set(value.albumIds).add(id),
	}));
}

export function markArtistAsNew(id: number) {
	newlyAdded.update((value) => ({
		...value,
		artistIds: new Set(value.artistIds).add(id),
	}));
}
