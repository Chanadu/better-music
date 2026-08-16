<script lang="ts">
	import DeleteIcon from '$lib/components/icons/DeleteIcon.svelte';

	let {
		dialog = $bindable(),
		title,
		description,
		confirmLabel,
		disabledReason,
		onconfirm,
	}: {
		dialog?: HTMLDialogElement;
		title: string;
		description: string;
		confirmLabel: string;
		disabledReason?: string;
		onconfirm: () => Promise<void>;
	} = $props();

	let deleting = $state(false);
	let error = $state('');

	function formatError(value: unknown) {
		const text = value instanceof Error ? value.message : 'Delete failed';
		return text.charAt(0).toUpperCase() + text.slice(1);
	}

	function handleClose() {
		if (!deleting) error = '';
	}

	async function confirm() {
		if (deleting || disabledReason) return;

		deleting = true;
		error = '';

		try {
			await onconfirm();
			dialog?.close();
		} catch (value) {
			console.error('Delete failed', value);
			error = formatError(value);
		} finally {
			deleting = false;
		}
	}
</script>

<dialog class="modal" bind:this={dialog} onclose={handleClose} oncancel={(event) => deleting && event.preventDefault()}>
	<div class="modal-box max-w-md">
		<div class="bg-error/15 text-error mx-auto mb-4 flex size-14 items-center justify-center rounded-full">
			<DeleteIcon class="size-7" />
		</div>

		<h2 class="text-center text-xl font-semibold">{title}</h2>
		<p class="text-base-content/70 mt-2 text-center">{description}</p>

		{#if disabledReason}
			<div class="alert alert-warning alert-soft mt-5 text-sm" role="status">
				<span>{disabledReason}</span>
			</div>
		{/if}

		{#if error}
			<div class="alert alert-error alert-soft mt-5 text-sm" role="alert">
				<span>{error}</span>
			</div>
		{/if}

		<div class="modal-action mt-6 grid grid-cols-2 gap-3">
			<form method="dialog">
				<button class="btn btn-soft btn-secondary w-full" disabled={deleting}>Cancel</button>
			</form>

			<button
				type="button"
				class="btn btn-error"
				disabled={deleting || Boolean(disabledReason)}
				onclick={confirm}
			>
				{#if deleting}<span class="loading loading-spinner loading-sm"></span>{/if}
				{deleting ? 'Deleting...' : confirmLabel}
			</button>
		</div>
	</div>

	<form method="dialog" class="modal-backdrop"><button disabled={deleting}>close</button></form>
</dialog>
