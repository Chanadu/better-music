<script>
	import { onMount } from 'svelte';
	import AuthPage from '$lib/components/auth/AuthPage.svelte';
	import { hasRemoteSession } from '$lib/scripts/auth';

	let showLogin = $state(false);

	onMount(() => {
		let active = true;

		void hasRemoteSession({ verifyOwner: true }).then((authenticated) => {
			if (!active) return;
			if (authenticated) location.replace('/');
			else showLogin = true;
		});

		return () => {
			active = false;
		};
	});
</script>

{#if showLogin}
	<AuthPage mode="login" />
{/if}
