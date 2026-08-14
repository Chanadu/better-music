<script lang="ts">
	import CreateModal from './CreateModal.svelte';

	let { type }: { type: 'artist' | 'album' } = $props();
	let dialog = $state<HTMLDialogElement>();
	let modalOpen = $state(false);

	function openModal() {
		modalOpen = true;
		dialog?.showModal();
	}
</script>

<div
	class="fab pb-[calc(3.5rem+env(safe-area-inset-bottom))] sm:pb-[calc(4.5rem+env(safe-area-inset-bottom))]"
	class:hidden={modalOpen}
>
	<button type="button" class="btn btn-lg btn-circle btn-accent" aria-label={`Create ${type}`} onclick={openModal}>
		+
	</button>
</div>

<CreateModal {type} bind:dialog onclose={() => (modalOpen = false)} />
