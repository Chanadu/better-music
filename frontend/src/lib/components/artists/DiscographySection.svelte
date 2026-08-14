<script lang="ts">
	import AddAlbumCard from '$lib/components/albums/AddAlbumCard.svelte';
	import AlbumCard from '$lib/components/albums/AlbumCard.svelte';
	import StarIcon from '$lib/components/icons/StarIcon.svelte';
	import GridIcon from '$lib/components/icons/GridIcon.svelte';
	import type { Album } from '$lib/scripts/types';

	type Filter = 'all' | 'rated' | 'unrated';

	let { albums, onadd }: { albums: Album[]; onadd?: () => void } = $props();
	let filter = $state<Filter>('all');
	let filteredAlbums = $derived(
		albums.filter((album) => {
			if (filter === 'rated') return typeof album.rating === 'number';
			if (filter === 'unrated') return typeof album.rating !== 'number';
			return true;
		}),
	);
</script>

<section class="mt-9">
	<div class="mb-5 flex items-center gap-2">
		<h2 class="text-secondary text-2xl leading-none font-black tracking-tighter sm:text-3xl">Discography</h2>

		<div class="divider mx-2 my-0 min-w-4 flex-1 self-center" aria-hidden="true"></div>

		<fieldset class="join grid shrink-0 grid-cols-3" aria-label="Filter discography">
			<label
				class="join-item btn btn-outline btn-primary btn-sm sm:btn-md has-checked:bg-primary has-checked:text-primary-content gap-1.5 px-3 sm:px-4"
			>
				<input class="sr-only" type="radio" name="discography-filter" value="all" bind:group={filter} />
				<GridIcon class="size-3.5" />
				All
			</label>

			<label
				class="join-item btn btn-outline btn-primary btn-sm sm:btn-md has-checked:bg-primary has-checked:text-primary-content gap-1.5 px-3 sm:px-4"
			>
				<input class="sr-only" type="radio" name="discography-filter" value="rated" bind:group={filter} />
				<StarIcon class="size-3.5" filled />
				Rated
			</label>

			<label
				class="join-item btn btn-outline btn-primary btn-sm sm:btn-md has-checked:bg-primary has-checked:text-primary-content gap-1.5 px-3 sm:px-4"
			>
				<input class="sr-only" type="radio" name="discography-filter" value="unrated" bind:group={filter} />
				<StarIcon class="size-3.5" />
				Not Rated
			</label>
		</fieldset>
	</div>

	<div class="grid grid-cols-2 gap-x-3 gap-y-6 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5">
		{#each filteredAlbums as album}
			<AlbumCard {album} subtitle={`${album.year ?? 'Year unknown'}`} showRating />
		{/each}

		<AddAlbumCard subtitle="New release" onclick={onadd} />
	</div>
</section>
