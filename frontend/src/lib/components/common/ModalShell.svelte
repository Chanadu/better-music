<script lang="ts">
	import type { Snippet } from 'svelte';

	let {
		dialog = $bindable(),
		class: className = '',
		backdropDisabled = false,
		onclose,
		oncancel,
		children,
	}: {
		dialog?: HTMLDialogElement;
		class?: string;
		backdropDisabled?: boolean;
		onclose?: (event: Event) => void;
		oncancel?: (event: Event) => void;
		children: Snippet;
	} = $props();
</script>

<dialog class="modal" bind:this={dialog} {onclose} {oncancel}>
	<div class={`modal-box max-h-[calc(100dvh-2rem)] ${className}`}>
		{@render children()}
	</div>

	<form method="dialog" class="modal-backdrop">
		<button disabled={backdropDisabled}>close</button>
	</form>
</dialog>
