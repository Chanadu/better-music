<script lang="ts">
	import AuthForm from './AuthForm.svelte';
	import { getCurrentUserId, type AuthMode } from '$lib/scripts/auth';
	let { mode }: { mode: AuthMode } = $props();
</script>

<div class="hero min-h-screen">
	<div class="hero-content w-full max-w-lg flex-col">
		<div class="text-center">
			<h1 class="text-2xl font-semibold">{mode === 'login' ? 'Login' : 'Create Account'}</h1>
		</div>

		<div class="card bg-base-200 w-full">
			<div class="card-body">
				<AuthForm {mode} />
				{#if mode === 'login'}
					<p class="text-center text-sm">
						<a href="/forgot-password" class="link link-primary">Forgot password?</a>
					</p>
					{#if getCurrentUserId() !== null}
						<p class="text-base-content/70 text-center text-sm">
							Sign in with the account that owns the library saved on this device.
						</p>
						<p class="text-center text-sm"><a href="/" class="link">Return to your saved library</a></p>
					{/if}
				{/if}

				<p class="text-base-content/70 mt-2 text-center text-sm">
					{#if mode === 'login'}
						Need an account?
						<a href="/create-account" class="link link-primary hover-lift inline-block">Create account</a>
					{:else}
						Already have an account?
						<a href="/login" class="link link-primary hover-lift inline-block">Log in</a>
					{/if}
				</p>
			</div>
		</div>
	</div>
</div>
