<script lang="ts">
	import SortControls from '$lib/components/common/SortControls.svelte';
	import { useSortPreference } from '$lib/scripts/sort-preferences.svelte';
	import SearchBar from '../common/SearchBar.svelte';
	import AlbumGrid from './AlbumGrid.svelte';
	import CreateFab from '../create/CreateFab.svelte';
	let { mode }: { mode: 'listened' | 'unlistened' } = $props();
	let query = $state('');

	const options = [
		{ label: 'Album', value: 'album' },
		{ label: 'Artist', value: 'artist' },
		{ label: 'Added', value: 'added' },
	] as const;
	let sorting = useSortPreference(
		() => `bettermusic:sort:${mode === 'listened' ? 'albums' : 'listen'}`,
		options.map((option) => option.value),
		{ sort: 'added', reversed: false },
	);
</script>

<div class="navbar gap-2">
	<SearchBar placeholder="album name..." bind:value={query} />

	<div class="pt-3">
		<SortControls {options} bind:sort={sorting.sort} bind:reversed={sorting.reversed} name={`album-sort-${mode}`} />
	</div>
</div>

<AlbumGrid {mode} {query} sort={sorting.sort} reversed={sorting.reversed} />

<CreateFab type="album" />
