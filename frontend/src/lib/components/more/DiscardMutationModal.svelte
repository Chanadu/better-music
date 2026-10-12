<script lang="ts">
	import ConfirmationDialog from '$lib/components/common/ConfirmationDialog.svelte';
	import { mutationLabel, type RecoveryPlan } from '$lib/scripts/queue-recovery';

	let {
		dialog = $bindable(),
		plan,
		busy,
		error,
		onconfirm,
	}: {
		dialog?: HTMLDialogElement;
		plan: RecoveryPlan | null;
		busy: boolean;
		error: string;
		onconfirm: () => void;
	} = $props();
</script>

<ConfirmationDialog
	bind:dialog
	title={`Discard ${plan ? mutationLabel(plan.mutation) : 'local change'}?`}
	messages={plan?.messages ?? []}
	confirmLabel="Discard changes"
	busyLabel="Discarding..."
	{busy}
	{error}
	disabled={!plan}
	{onconfirm}
/>
