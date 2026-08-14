<script lang="ts">
	import type { Snippet } from 'svelte';

	let {
		dialog = $bindable(),
		title,
		error = '',
		saving = false,
		canSave,
		onsave,
		onclose,
		children,
	}: {
		dialog?: HTMLDialogElement;
		title: string;
		error?: string;
		saving?: boolean;
		canSave: boolean;
		onsave: () => void;
		onclose?: () => void;
		children: Snippet;
	} = $props();
</script>

<dialog class="modal" bind:this={dialog} {onclose}>
	<div class="modal-box">
		<form method="dialog">
			<button class="btn btn-sm btn-circle btn-ghost absolute top-2 right-2">✕</button>
		</form>

		<h2 class="mb-2 text-center text-xl font-semibold">Create {title}</h2>

		{@render children()}

		{#if error}
			<div class="alert alert-error alert-soft mt-4 justify-center text-center" role="alert">
				<span>{error}</span>
			</div>
		{/if}

		<div class="modal-action mt-6 flex gap-4 px-2">
			<form method="dialog" class="flex-1">
				<button class="btn btn-soft btn-secondary w-full">Cancel</button>
			</form>

			<button type="button" class="btn btn-primary flex-1" disabled={!canSave} onclick={onsave}>
				{saving ? 'Saving...' : `Save ${title}`}
			</button>
		</div>
	</div>

	<form method="dialog" class="modal-backdrop"><button>close</button></form>
</dialog>
