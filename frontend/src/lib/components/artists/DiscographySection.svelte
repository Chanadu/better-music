<script lang="ts">
	import SortControls from '$lib/components/common/SortControls.svelte';
	import AddAlbumCard from '$lib/components/albums/AddAlbumCard.svelte';
	import AlbumCard from '$lib/components/albums/AlbumCard.svelte';
	import type { Album } from '$lib/scripts/types';
	import { useSortPreference } from '$lib/scripts/sort-preferences.svelte';

	type Filter = 'all' | 'listened' | 'unlistened';

	let { albums, onadd }: { albums: Album[]; onadd?: () => void } = $props();
	let filter = $state<Filter>('all');
	const filters = [
		{ value: 'all', label: 'All' },
		{ value: 'listened', label: 'Listened' },
		{ value: 'unlistened', label: 'Unlistened' },
	] as const;
	const sortOptions = [
		{ label: 'Release year', value: 'year' },
		{ label: 'Album title', value: 'album' },
		{ label: 'Rating', value: 'rating' },
		{ label: 'Date added', value: 'added' },
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
	<div class="mb-5 grid grid-cols-[minmax(0,1fr)_auto] items-center gap-x-2 gap-y-3 sm:gap-x-4">
		<div class="col-span-2 col-start-1 row-start-1 flex min-w-0 items-center gap-2 sm:gap-4">
			<h2 class="text-secondary text-3xl leading-none font-black tracking-tighter">Discography</h2>
			<div class="divider my-0 min-w-4 flex-1 self-center" aria-hidden="true"></div>
		</div>
		<div class="col-start-2 row-start-3 w-44 sm:w-48 md:row-start-2">
			<SortControls
				options={sortOptions}
				bind:sort={sorting.sort}
				bind:reversed={sorting.reversed}
				name="discography-sort"
				fullWidth
			/>
		</div>
		<fieldset
			class="join col-span-2 col-start-1 row-start-2 grid w-full min-w-0 grid-cols-3 md:col-span-1"
			aria-label="Filter discography"
		>
			{#each filters as option}
				<label
					class="join-item btn btn-outline btn-primary btn-md has-checked:bg-primary has-checked:text-primary-content px-2 text-sm has-focus-visible:outline-2 has-focus-visible:outline-offset-2 sm:px-6"
				>
					<input
						class="sr-only"
						type="radio"
						name="discography-filter"
						value={option.value}
						bind:group={filter}
					/>
					{option.label}
				</label>
			{/each}
		</fieldset>
	</div>

	<div class="grid grid-cols-2 gap-x-3 gap-y-6 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6">
		{#each filteredAlbums as album}
			<AlbumCard {album} subtitle={`${album.year ?? 'Year unknown'}`} showRating />
		{/each}

		<AddAlbumCard subtitle="New release" onclick={onadd} />
	</div>
</section>
