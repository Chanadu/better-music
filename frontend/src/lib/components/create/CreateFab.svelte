<script lang="ts">
	import CreateAlbumModal from './CreateAlbumModal.svelte';
	import CreateArtistModal from './CreateArtistModal.svelte';

	let { type }: { type: 'artist' | 'album' } = $props();
	let dialog = $state<HTMLDialogElement>();
	let modalOpen = $state(false);

	function openModal() {
		modalOpen = true;
		dialog?.showModal();
	}
</script>

<div
	data-create-fab
	class="pointer-events-none fixed inset-x-0 bottom-[calc(4.5rem+env(safe-area-inset-bottom))] z-[999] flex justify-end px-4 sm:bottom-[calc(5.5rem+env(safe-area-inset-bottom))]"
	class:hidden={modalOpen}
>
	<button
		type="button"
		class="btn btn-lg btn-circle btn-accent hover-lift pointer-events-auto"
		aria-label={`Create ${type}`}
		onclick={openModal}
	>
		+
	</button>
</div>

{#if type === 'artist'}
	<CreateArtistModal bind:dialog onclose={() => (modalOpen = false)} />
{:else}
	<CreateAlbumModal bind:dialog onclose={() => (modalOpen = false)} />
{/if}
