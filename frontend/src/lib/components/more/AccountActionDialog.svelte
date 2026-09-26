<script lang="ts">
	import ModalShell from '$lib/components/common/ModalShell.svelte';
	import PasswordField from '$lib/components/auth/PasswordField.svelte';
	import DeleteIcon from '$lib/components/icons/DeleteIcon.svelte';
	import EmailIcon from '$lib/components/icons/EmailIcon.svelte';
	import { accountApi } from '$lib/scripts/api';
	import { accountErrorMessage, finishAccountAction } from '$lib/scripts/account';

	type Action = 'email' | 'password' | 'delete';

	let dialog = $state<HTMLDialogElement>();
	let action = $state<Action>('email');
	let currentEmail = $state('');
	let email = $state('');
	let currentPassword = $state('');
	let newPassword = $state('');
	let confirmPassword = $state('');
	let shown = $state({ current: false, new: false, confirm: false });
	let busy = $state(false);
	let error = $state('');

	let valid = $derived(
		action === 'email' ? Boolean(email && email !== currentEmail && currentPassword)
		: action === 'password' ? Boolean(currentPassword && newPassword.length >= 4 && newPassword === confirmPassword)
		: Boolean(currentPassword),
	);

	export function open(nextAction: Action, accountEmail = '') {
		reset();
		action = nextAction;
		currentEmail = accountEmail;
		email = accountEmail;
		dialog?.showModal();
	}

	function reset() {
		if (busy) return;
		email = '';
		currentPassword = '';
		newPassword = '';
		confirmPassword = '';
		shown = { current: false, new: false, confirm: false };
		error = '';
	}

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		if (busy || !valid) return;
		busy = true;
		error = '';

		try {
			if (action === 'email') await accountApi.updateEmail({ email, password: currentPassword });
			else if (action === 'password') {
				await accountApi.updatePassword({ current_password: currentPassword, new_password: newPassword });
			} else await accountApi.delete({ password: currentPassword });

			finishAccountAction();
		} catch (value) {
			error = accountErrorMessage(value);
			busy = false;
		}
	}
</script>

<ModalShell bind:dialog class="max-w-md" backdropDisabled={busy} onclose={reset}>
	{#if action === 'delete'}
		<div class="bg-error/15 text-error mx-auto mb-4 flex size-14 items-center justify-center rounded-full">
			<DeleteIcon class="size-7" />
		</div>
		<h2 class="text-center text-xl font-bold">Delete your account?</h2>
		<p class="text-base-content/65 mt-2 text-center text-sm">
			This permanently deletes your library and account. This cannot be undone.
		</p>
	{:else}
		<h2 class="text-xl font-bold">Change {action}</h2>
		<p class="text-base-content/60 mt-1 text-sm">You’ll be signed out after this change.</p>
	{/if}

	<form class="mt-5 flex flex-col gap-4" onsubmit={submit}>
		{#if action === 'email'}
			<div class="fieldset">
				<span class="label">New email</span>
				<label class="input w-full">
					<EmailIcon class="h-[1em] opacity-50" />
					<input type="email" required autocomplete="email" bind:value={email} />
				</label>
			</div>
		{/if}

		<div class="fieldset">
			<span class="label">Current password</span>
			<PasswordField bind:value={currentPassword} bind:shown={shown.current} validate={action === 'delete'} />
		</div>

		{#if action === 'password'}
			<div class="fieldset">
				<span class="label">New password</span>
				<PasswordField bind:value={newPassword} bind:shown={shown.new} autocomplete="new-password" />
			</div>
			<div class="fieldset">
				<span class="label">Confirm new password</span>
				<PasswordField
					bind:value={confirmPassword}
					bind:shown={shown.confirm}
					autocomplete="new-password"
					hint="Passwords must match"
				/>
			</div>
		{/if}

		{#if error}<p class="text-error text-sm" role="alert">{error}</p>{/if}

		<div
			class:modal-action={true}
			class:mt-2={true}
			class:grid={action === 'delete'}
			class:grid-cols-2={action === 'delete'}
			class:gap-3={action === 'delete'}
		>
			<button type="button" class="btn btn-ghost" disabled={busy} onclick={() => dialog?.close()}>Cancel</button>
			<button
				type="submit"
				class:btn={true}
				class:btn-error={action === 'delete'}
				class:btn-primary={action === 'email'}
				class:btn-accent={action === 'password'}
				disabled={busy || !valid}
			>
				{#if busy}<span class="loading loading-sm"></span>{/if}
				{action === 'delete' ? 'Delete account' : 'Save'}
			</button>
		</div>
	</form>
</ModalShell>
