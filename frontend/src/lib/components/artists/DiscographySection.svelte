<script lang="ts">
	import SortControls from '$lib/components/common/SortControls.svelte';
	import AddAlbumCard from '$lib/components/albums/AddAlbumCard.svelte';
	import AlbumCard from '$lib/components/albums/AlbumCard.svelte';
	import CheckIcon from '$lib/components/icons/CheckIcon.svelte';
	import GridIcon from '$lib/components/icons/GridIcon.svelte';
	import type { Album } from '$lib/scripts/types';
	import { useSortPreference } from '$lib/scripts/sort-preferences.svelte';

	type Filter = 'all' | 'listened' | 'unlistened';

	let { albums, onadd }: { albums: Album[]; onadd?: () => void } = $props();
	let filter = $state<Filter>('all');
	const sortOptions = [
		{ label: 'Year', value: 'year' },
		{ label: 'Album', value: 'album' },
		{ label: 'Rating', value: 'rating' },
		{ label: 'Added', value: 'added' },
	] as const;
	let sorting = useSortPreference(
		() => 'bettermusic:sort:discography',
		sortOptions.map((option) => option.value),
		{ sort: 'year', reversed: false },
	);
	let filteredAlbums = $derived(
		albums
			.filter((album) => {
				if (filter === 'listened') return album.listened;
				if (filter === 'unlistened') return !album.listened;
				return true;
			})
			.sort((a, b) => {
				const direction = sorting.reversed ? -1 : 1;
				let comparison = 0;
				if (sorting.sort === 'album') {
					comparison = a.title.localeCompare(b.title);
				} else if (sorting.sort === 'added') {
					comparison = new Date(b.created_at).getTime() - new Date(a.created_at).getTime();
				} else {
					const left = a[sorting.sort];
					const right = b[sorting.sort];
					if (left == null && right != null) return 1;
					if (left != null && right == null) return -1;
					comparison = (right ?? 0) - (left ?? 0);
				}
				return direction * (comparison || a.title.localeCompare(b.title));
			}),
	);
</script>

<section class="mt-9">
	<div
		class="mb-5 grid grid-cols-[auto_minmax(0.5rem,1fr)_auto] items-center gap-x-2 gap-y-3 lg:grid-cols-[auto_minmax(0.5rem,1fr)_auto_auto]"
	>
		<h2 class="text-secondary text-lg leading-none font-black tracking-tighter sm:text-3xl">Discography</h2>

		<div class="divider my-0 w-full self-center" aria-hidden="true"></div>

		<fieldset class="join grid shrink-0 grid-cols-3" aria-label="Filter discography">
			<label
				class="join-item btn btn-outline btn-primary btn-sm sm:btn-md has-checked:bg-primary has-checked:text-primary-content gap-1 px-1 text-xs sm:gap-1.5 sm:px-4 sm:text-sm"
			>
				<input class="sr-only" type="radio" name="discography-filter" value="all" bind:group={filter} />
				<GridIcon class="size-3.5" />
				All
			</label>

			<label
				class="join-item btn btn-outline btn-primary btn-sm sm:btn-md has-checked:bg-primary has-checked:text-primary-content gap-1 px-1 text-xs sm:gap-1.5 sm:px-4 sm:text-sm"
			>
				<input class="sr-only" type="radio" name="discography-filter" value="listened" bind:group={filter} />
				<CheckIcon class="size-3.5" />
				Listened
			</label>

			<label
				class="join-item btn btn-outline btn-primary btn-sm sm:btn-md has-checked:bg-primary has-checked:text-primary-content gap-1 px-1 text-xs sm:gap-1.5 sm:px-4 sm:text-sm"
			>
				<input class="sr-only" type="radio" name="discography-filter" value="unlistened" bind:group={filter} />
				<CheckIcon class="size-3.5 opacity-40" />
				Not Listened
			</label>
		</fieldset>
		<div class="col-span-3 justify-self-end lg:col-span-1">
			<SortControls
				options={sortOptions}
				bind:sort={sorting.sort}
				bind:reversed={sorting.reversed}
				name="discography-sort"
			/>
		</div>
	</div>

	<div class="grid grid-cols-2 gap-x-3 gap-y-6 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5">
		{#each filteredAlbums as album}
			<AlbumCard {album} subtitle={`${album.year ?? 'Year unknown'}`} showRating />
		{/each}

		<AddAlbumCard subtitle="New release" onclick={onadd} />
	</div>
</section>
