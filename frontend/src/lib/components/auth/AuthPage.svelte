<script lang="ts">
	import AuthForm from './AuthForm.svelte';
	let { mode }: { mode: 'login' | 'register' } = $props();
</script>

<div class="hero min-h-screen">
	<div class="hero-content w-full max-w-lg flex-col lg:flex-row-reverse">
		<div class="text-center lg:text-left">
			<h1 class="text-2xl font-semibold">{mode === 'login' ? 'Login' : 'Create Account'}</h1>
		</div>

		<div class="card bg-base-200 w-full">
			<div class="card-body">
				<AuthForm {mode} />

				<p class="text-base-content/70 mt-2 text-center text-sm">
					{#if mode === 'login'}
						Need an account?
						<a href="/create-account" class="link link-primary">Create account</a>
					{:else}
						Already have an account?
						<a href="/login" class="link link-primary">Log in</a>
					{/if}
				</p>
			</div>
		</div>
	</div>
</div>
