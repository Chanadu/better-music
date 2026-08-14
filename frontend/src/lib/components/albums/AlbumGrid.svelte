<script lang="ts">
	import { database } from '$lib/scripts/database';
	import type { Album, Artist } from '$lib/scripts/types';
	import AlbumGridItem from './AlbumGridItem.svelte';

	type Item = { album: Album; artist?: Artist };

	let { mode, query = '' }: { mode: 'rated' | 'unrated'; query?: string } = $props();

	let items = $derived.by(() => {
		if (!$database) return [];
		const artists = new Map($database.artists.map((artist) => [artist.id, artist]));
		return $database.albums
			.filter((album) => (typeof album.rating === 'number') === (mode === 'rated'))
			.map((album) => ({ album, artist: artists.get(album.artist_id) }))
			.filter(
				({ album, artist }) =>
					album.title.toLowerCase().includes(query.trim().toLowerCase()) ||
					(artist?.name.toLowerCase() ?? '').includes(query.trim().toLowerCase()),
			)
			.sort((a, b) => a.album.title.localeCompare(b.album.title));
	});

	let groups = $derived.by(() => {
		if (mode === 'unrated') return [{ rating: undefined, albums: items }];
		const map = new Map<number, Item[]>();
		for (const item of items) map.set(item.album.rating ?? 0, [...(map.get(item.album.rating ?? 0) ?? []), item]);
		return [...map].sort(([a], [b]) => b - a).map(([rating, albums]) => ({ rating, albums }));
	});
</script>

{#if mode === 'unrated' || items.length === 0}
	<div class="divider mt-2 mb-0"></div>
{/if}

{#if !$database}
	<div class="text-base-content/60 mt-4 px-2">Loading albums...</div>
{:else if items.length === 0}
	<div class="text-base-content/60 mt-4 px-2">
		{query ? `No ${mode} albums match your search.` : `No ${mode} albums yet.`}
	</div>
{:else}
	<div class={mode === 'unrated' ? 'mt-4' : ''}>
		{#each groups as group}
			<section class="pt-0 pb-2">
				{#if group.rating !== undefined}
					<div class="divider divider-center divider-primary text-xl">
						{group.rating}/10 • {group.albums.length}
						{group.albums.length === 1 ? 'album' : 'albums'}
					</div>
				{/if}

				<div class="grid grid-cols-3 gap-3 sm:grid-cols-4 md:grid-cols-5 lg:grid-cols-6">
					{#each group.albums as item}
						<AlbumGridItem album={item.album} subtitle={item.artist?.name ?? 'Unknown Artist'} compact />
					{/each}
				</div>
			</section>
		{/each}
	</div>
{/if}
