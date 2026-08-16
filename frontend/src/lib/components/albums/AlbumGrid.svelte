<script lang="ts">
	import { database } from '$lib/scripts/database';
	import type { Album, Artist } from '$lib/scripts/types';
	import AlbumCard from './AlbumCard.svelte';

	type Item = { album: Album; artist?: Artist };
	const ratingLabels: Record<number, string> = {
		10: 'whoa',
		9: 'incredible',
		8: 'great',
		7: 'good',
		6: 'solid',
		5: 'okay',
		4: 'meh',
		3: 'bad',
		2: 'terrible',
		1: "just don't",
	};
	function ratingHeading(rating: number | undefined) {
		if (rating === undefined) return 'Unrated';
		const label = ratingLabels[rating];
		return label ? `${rating}/10 • ${label}` : `${rating}/10`;
	}

	let {
		mode,
		query = '',
		sort = 'added',
		reversed = false,
	}: {
		mode: 'listened' | 'unlistened';
		query?: string;
		sort?: 'album' | 'artist' | 'added';
		reversed?: boolean;
	} = $props();

	let items = $derived.by(() => {
		if (!$database) return [];
		const artists = new Map($database.artists.map((artist) => [artist.id, artist]));
		const result = $database.albums
			.filter((album) => album.listened === (mode === 'listened'))
			.map((album) => ({ album, artist: artists.get(album.artist_id) }))
			.filter(
				({ album, artist }) =>
					album.title.toLowerCase().includes(query.trim().toLowerCase()) ||
					(artist?.name.toLowerCase() ?? '').includes(query.trim().toLowerCase()),
			);

		const direction = reversed ? -1 : 1;

		return result.sort((a, b) => {
			if (sort === 'artist') {
				return direction * (a.artist?.name ?? '').localeCompare(b.artist?.name ?? '');
			}

			if (sort === 'added') {
				return direction * (new Date(b.album.created_at).getTime() - new Date(a.album.created_at).getTime());
			}

			return direction * a.album.title.localeCompare(b.album.title);
		});
	});

	let groups = $derived.by(() => {
		if (mode === 'unlistened') return [{ rating: undefined, albums: items }];
		const map = new Map<number | undefined, Item[]>();
		for (const item of items) {
			const rating = item.album.rating;
			map.set(rating, [...(map.get(rating) ?? []), item]);
		}
		return [...map].sort(([a], [b]) => (b ?? -1) - (a ?? -1)).map(([rating, albums]) => ({ rating, albums }));
	});
</script>

{#if mode === 'unlistened' || items.length === 0}
	<div class="divider mt-2 mb-0"></div>
{/if}

{#if !$database}
	<div class="text-base-content/60 mt-4 px-2">Loading albums...</div>
{:else if items.length === 0}
	<div class="text-base-content/60 mt-4 px-2">
		{query ? `No ${mode} albums match your search.` : `No ${mode} albums yet.`}
	</div>
{:else}
	<div class={mode === 'unlistened' ? 'mt-4' : ''}>
		{#each groups as group}
			<section class="pt-0 pb-2">
				{#if mode === 'listened'}
					<div class="divider divider-center divider-primary text-xl">
						{ratingHeading(group.rating)} • {group.albums.length}
						{group.albums.length === 1 ? 'album' : 'albums'}
					</div>
				{/if}

				<div class="grid grid-cols-3 gap-3 sm:grid-cols-4 md:grid-cols-5 lg:grid-cols-6">
					{#each group.albums as item}
						<AlbumCard album={item.album} subtitle={item.artist?.name ?? 'Unknown Artist'} compact />
					{/each}
				</div>
			</section>
		{/each}
	</div>
{/if}
