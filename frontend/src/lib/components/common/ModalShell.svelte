<script lang="ts">
	import { getContext, type Snippet } from 'svelte';
	import { previewContext, type PreviewContext } from '$lib/components/preview/context';
	const preview = getContext<PreviewContext | undefined>(previewContext);

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

{#snippet box()}
	<div class={`modal-box max-h-[calc(100dvh-2rem)] ${className}`}>
		{@render children()}
	</div>
{/snippet}

{#if preview?.inlineDialogs}
	<div class="preview-inline-dialog">
		{@render box()}
	</div>
{:else}

	<dialog class="modal" bind:this={dialog} {onclose} {oncancel}>
		{@render box()}
		<form method="dialog" class="modal-backdrop">
			<button disabled={backdropDisabled}>
				close
			</button>
		</form>
	</dialog>
{/if}
