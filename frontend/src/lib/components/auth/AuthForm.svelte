<script lang="ts">
	import { authApi, ApiError } from '$lib/scripts/api';
	import { saveTokens } from '$lib/scripts/auth';
	import PasswordField from './PasswordField.svelte';
	import EmailIcon from '../icons/EmailIcon.svelte';

	let { mode }: { mode: 'login' | 'register' } = $props();

	let email = $state('');
	let password = $state('');
	let confirmPassword = $state('');
	let showPassword = $state(false);
	let showConfirm = $state(false);
	let busy = $state(false);
	let error = $state('');
	let copy = $derived.by(() => {
		if (mode === 'login') {
			return {
				button: 'Login',
				loadingClass: 'loading-dots',
				loadingLabel: 'Logging in',
				passwordAutocomplete: 'current-password' as const,
			};
		}

		return {
			button: 'Create',
			loadingClass: 'loading-spinner',
			loadingLabel: 'Creating',
			passwordAutocomplete: 'new-password' as const,
		};
	});

	function isValidEmail(value: string) {
		if ([...value].some((character) => character.trim() === '')) return false;

		const parts = value.split('@');
		if (parts.length !== 2 || !parts[0] || !parts[1]) return false;

		const lastDot = parts[1].lastIndexOf('.');
		return lastDot > 0 && lastDot < parts[1].length - 1;
	}

	let valid = $derived(
		isValidEmail(email) && password.length >= 4 && (mode === 'login' || confirmPassword === password),
	);

	async function submit(event: SubmitEvent) {
		event.preventDefault();

		if (!valid || busy) return;
		busy = true;
		error = '';
		try {
			let tokens;
			if (mode === 'register') {
				tokens = await authApi.register({ email, password });
			} else {
				tokens = await authApi.login({ email, password });
			}

			saveTokens(tokens);
			location.assign('/');
		} catch (e) {
			error = e instanceof ApiError ? e.message : 'Could not reach the server. Please try again.';
		} finally {
			busy = false;
		}
	}
</script>

<form onsubmit={submit}>
	<fieldset class="fieldset flex flex-col gap-4">
		<div class="flex flex-col gap-2">
			<div class="label">Email</div>
			<div class="flex w-full flex-col gap-1">
				<label class="input validator w-full">
					<EmailIcon class="h-[1em] opacity-50" />
					<input
						type="email"
						name="email"
						placeholder="mail@site.com"
						autocomplete="email"
						required
						bind:value={email}
					/>
				</label>
				<div class="validator-hint hidden w-full">Enter valid email address</div>
			</div>
		</div>

		<div class="flex flex-col gap-2">
			<div class="label">Password</div>
			<PasswordField bind:value={password} bind:shown={showPassword} autocomplete={copy.passwordAutocomplete} />
		</div>
		{#if mode === 'register'}
			<div class="flex flex-col gap-2">
				<div class="label">Confirm password</div>
				<PasswordField
					bind:value={confirmPassword}
					bind:shown={showConfirm}
					autocomplete="new-password"
					hint="Passwords must match"
				/>
			</div>
		{/if}

		{#if error}
			<p class="text-error text-sm" class:text-center={mode === 'login'}>
				{error}
			</p>
		{/if}

		<button
			type="submit"
			class="btn btn-primary mx-auto mt-4 w-min"
			class:btn-disabled={!valid || busy}
			disabled={!valid || busy}
		>
			{copy.button}
		</button>
		{#if busy}
			<div class="text-primary flex justify-center" aria-live="polite">
				<span class={`loading ${copy.loadingClass} loading-md`} aria-label={copy.loadingLabel}></span>
			</div>
		{/if}
	</fieldset>
</form>
