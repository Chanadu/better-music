<script lang="ts">
	import DeleteIcon from '$lib/components/icons/DeleteIcon.svelte';
	import ModalShell from './ModalShell.svelte';

	let {
		dialog = $bindable(),
		title,
		messages = [],
		confirmLabel,
		busyLabel,
		busy = false,
		error = '',
		disabled = false,
		disabledReason,
		onconfirm,
		onclose,
	}: {
		dialog?: HTMLDialogElement;
		title: string;
		messages?: string[];
		confirmLabel: string;
		busyLabel: string;
		busy?: boolean;
		error?: string;
		disabled?: boolean;
		disabledReason?: string;
		onconfirm: () => void;
		onclose?: (event: Event) => void;
	} = $props();
</script>

<ModalShell
	bind:dialog
	class="max-w-md"
	backdropDisabled={busy}
	{onclose}
	oncancel={(event) => busy && event.preventDefault()}
>
	<div class="bg-error/15 text-error mx-auto mb-4 flex size-14 items-center justify-center rounded-full">
		<DeleteIcon class="size-7" />
	</div>

	<h2 class="text-center text-xl font-semibold">
		{title}
	</h2>

	{#if messages.length}
		<div class="text-base-content/70 mt-2 space-y-2 text-center">
			{#each messages as message}
				<p>
					{message}
				</p>
			{/each}
		</div>
	{/if}

	{#if disabledReason}
		<div class="alert alert-warning alert-soft mt-5" role="status">
			<span>
				{disabledReason}
			</span>
		</div>
	{/if}

	{#if error}
		<div class="alert alert-error alert-soft mt-5" role="alert">
			<span>
				{error}
			</span>
		</div>
	{/if}

	<div class="modal-action grid grid-cols-2 gap-3">
		<form method="dialog">
			<button class="btn hover-lift btn-error w-full" disabled={busy}> Cancel </button>
		</form>

		<button
			type="button"
			class="btn hover-lift btn-error"
			disabled={busy || disabled || Boolean(disabledReason)}
			onclick={onconfirm}
		>
			{#if busy}
				<span class="loading loading-sm"> </span>
			{/if}
			{busy ? busyLabel : confirmLabel}
		</button>
	</div>
</ModalShell>
