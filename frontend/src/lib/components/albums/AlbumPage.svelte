<script lang="ts">
	import SearchBar from '../common/SearchBar.svelte';
	import AlbumGrid from './AlbumGrid.svelte';
	import CreateFab from '../create/CreateFab.svelte';
	let { mode }: { mode: 'rated' | 'unrated' } = $props();
	let query = $state('');
</script>

<div class="navbar"><SearchBar placeholder="album name..." bind:value={query} /></div>

<AlbumGrid {mode} {query} />

<CreateFab type="album" />
