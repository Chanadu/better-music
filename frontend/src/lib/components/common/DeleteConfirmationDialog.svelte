<script lang="ts">
	import ConfirmationDialog from './ConfirmationDialog.svelte';

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

<ConfirmationDialog
	bind:dialog
	{title}
	messages={[description]}
	{confirmLabel}
	busyLabel="Deleting..."
	busy={deleting}
	{error}
	{disabledReason}
	onconfirm={confirm}
	onclose={handleClose}
/>
