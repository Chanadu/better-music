<script lang="ts">
	import SortControls from '$lib/components/common/SortControls.svelte';
	import SearchBar from '$lib/components/common/SearchBar.svelte';
	import ArtistListItem from '$lib/components/artists/ArtistListItem.svelte';
	import CreateFab from '$lib/components/create/CreateFab.svelte';
	import { database } from '$lib/scripts/database';
	import { useSortPreference } from '$lib/scripts/sort-preferences.svelte';

	let query = $state('');
	const options = [
		{ label: 'Rating', value: 'rating' },
		{ label: 'Name', value: 'name' },
		{ label: 'Added', value: 'added' },
	] as const;
	let sorting = useSortPreference(
		() => 'bettermusic:sort:artists',
		options.map((option) => option.value),
		{ sort: 'rating', reversed: false },
	);

	function averageRating(ratings: number[]) {
		if (ratings.length === 0) return null;
		return ratings.reduce((sum, rating) => sum + rating, 0) / ratings.length;
	}

	let items = $derived.by(() => {
		if (!$database) return [];

		const result = $database.artists
			.map((artist) => {
				const albums = $database!.albums.filter((album) => album.artist_id === artist.id);
				const ratings = albums.flatMap((album) => (typeof album.rating === 'number' ? [album.rating] : []));

				return {
					artist,
					albumCount: albums.length,
					averageRating: averageRating(ratings),
				};
			})
			.filter(({ artist }) => artist.name.toLowerCase().includes(query.trim().toLowerCase()));

		const direction = sorting.reversed ? -1 : 1;

		return result.sort((a, b) => {
			if (sorting.sort === 'name') {
				return direction * a.artist.name.localeCompare(b.artist.name);
			}

			if (sorting.sort === 'added') {
				return direction * (new Date(b.artist.created_at).getTime() - new Date(a.artist.created_at).getTime());
			}

			return direction * ((b.averageRating ?? -1) - (a.averageRating ?? -1));
		});
	});
</script>

<div class="navbar gap-2">
	<SearchBar placeholder="artist name..." bind:value={query} />

	<div class="pt-3">
		<SortControls {options} bind:sort={sorting.sort} bind:reversed={sorting.reversed} name="artist-sort" />
	</div>
</div>

<div class="divider mt-2 mb-0"></div>

{#if !$database}
	<div class="text-base-content/60 mt-4 px-2">Loading artists...</div>
{:else if items.length === 0}
	<div class="text-base-content/60 mt-4 px-2">
		{query ? 'No artists match your search.' : 'No artists yet.'}
	</div>
{:else}
	<ul class="list bg-base-200 rounded-box mt-4 shadow-md">
		{#each items as item}
			<ArtistListItem {...item} />
		{/each}
	</ul>
{/if}

<CreateFab type="artist" />
