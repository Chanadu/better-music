<script lang="ts">
	import { resetPassword, getCurrentUserId } from '$lib/scripts/auth';
	import PasswordField from './PasswordField.svelte';

	let password = $state('');
	let confirm = $state('');
	let shown = $state(false);
	let busy = $state(false);
	let complete = $state(false);
	let error = $state('');

	async function submit(event: SubmitEvent) {
		event.preventDefault();

		if (busy) return;
		busy = true;
		error = '';

		try {
			if (password !== confirm) throw new Error('Passwords must match.');

			await resetPassword(password);
			complete = true;
		} catch (value) {
			error = value instanceof Error ? value.message : 'Could not reach the server. Try again.';
		} finally {
			busy = false;
		}
	}
</script>

<div class="hero min-h-screen">
	<div class="hero-content w-full max-w-lg flex-col">
		<h1 class="text-2xl font-semibold">Choose a new password</h1>
		<div class="card bg-base-200 w-full">
			<div class="card-body">
				{#if complete}
					<p role="status">Password updated. Sign in to resume syncing your saved library.</p>
				{:else}
					<p class="text-sm">Use at least 8 characters, including a letter and a number.</p>
					<form onsubmit={submit} class="flex flex-col gap-4">
						<div>
							<span>New password</span>
							<PasswordField
								label="New password"
								minimumLength={8}
								bind:value={password}
								bind:shown
								autocomplete="new-password"
								hint="Use at least 8 characters, including a letter and a number"
							/>
						</div>
						<label
							>Confirm password<input
								class="input w-full"
								type="password"
								autocomplete="new-password"
								required
								minlength="8"
								bind:value={confirm}
							/></label
						>
						{#if error}<p class="text-error text-sm" role="alert">{error}</p>{/if}
						<button
							class="btn btn-primary"
							type="submit"
							disabled={busy || password.length < 8 || password !== confirm}
							>{busy ? 'Please wait...' : 'Save password'}</button
						>
					</form>
				{/if}
				<a class="link link-primary text-center" href="/login">Back to login</a>
				{#if getCurrentUserId() !== null}<a class="link text-center" href="/">Return to your saved library</a
					>{/if}
			</div>
		</div>
	</div>
</div>
