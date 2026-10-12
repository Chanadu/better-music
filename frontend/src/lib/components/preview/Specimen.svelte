<script lang="ts">
	import { setContext } from 'svelte';
	import type { Snippet } from 'svelte';
	import { previewContext, type PreviewContext } from './context';

	let {
		label,
		children,
		recordStatus,
		accountAction,
		online = true,
		newlyAdded = false,
		nativeDropdowns = false,
		library,
	}: {
		label: string;
		children: Snippet;
		recordStatus?: PreviewContext['recordStatus'];
		accountAction?: PreviewContext['accountAction'];
		online?: boolean;
		newlyAdded?: boolean;
		nativeDropdowns?: boolean;
		library?: PreviewContext['library'];
	} = $props();
	setContext<PreviewContext>(previewContext, {
		inlineDialogs: true,
		get library() {
			return library;
		},
		get nativeDropdowns() {
			return nativeDropdowns;
		},
		get online() {
			return online;
		},
		get newlyAdded() {
			return newlyAdded;
		},
		get recordStatus() {
			return recordStatus;
		},
		get accountAction() {
			return accountAction;
		},
	});
</script>

<article class="specimen min-w-0">
	<h3 class="text-base-content/60 mb-3 text-xs font-bold tracking-wide uppercase">
		{label}
	</h3>

	<div class="min-w-0">
		{@render children()}
	</div>
</article>

<style>
	.specimen :global(.preview-inline-dialog) {
		position: relative;
		display: block;
		visibility: visible;
		opacity: 1;
		pointer-events: auto;
		width: 100%;
		height: auto;
		background: transparent;
		color: inherit;
	}
	.specimen :global(.modal-box) {
		position: relative;
		width: 100%;
		max-width: none;
		max-height: none;
		overflow: visible;
		overscroll-behavior: auto;
		transform: none;
		opacity: 1;
	}
	.specimen :global(.tab-content) {
		display: block;
	}
	.specimen :global(.dropdown-content) {
		position: relative;
		transform: none;
	}
	.specimen :global(.modal-backdrop) {
		display: none;
	}
	.specimen :global(.dock),
	.specimen :global(.fab) {
		position: relative;
		inset: auto;
	}
</style>
