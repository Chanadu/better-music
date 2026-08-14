<script lang="ts">
	import TopNav from '../navigation/TopNav.svelte';
	import SearchBar from '../common/SearchBar.svelte';
	import AlbumGrid from './AlbumGrid.svelte';
	import CreateModal from '../create/CreateModal.svelte';
	let { mode }: { mode: 'rated' | 'unrated' } = $props();
	let query = $state('');
	let modal = $state<HTMLDialogElement>();
	let modalOpen = $state(false);
</script>

<TopNav breadcrumbs={[mode === 'rated' ? 'Albums' : 'Listen']} />

<div class="navbar"><SearchBar placeholder="album name..." bind:value={query} /></div>

<AlbumGrid {mode} {query} />

<div class="fab pb-[calc(4.5rem+env(safe-area-inset-bottom))]" class:hidden={modalOpen}>
	<button
		class="btn btn-lg btn-circle btn-accent"
		onclick={() => {
			modalOpen = true;
			modal?.showModal();
		}}
	>
		+
	</button>
</div>

<CreateModal type="album" bind:dialog={modal} onclose={() => (modalOpen = false)} />
