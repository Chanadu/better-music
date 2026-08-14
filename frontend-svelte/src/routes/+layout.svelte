<script lang="ts">
	import '../app.css';
	import { page } from '$app/state';
	import { onMount } from 'svelte';
	import BottomNav from '$lib/components/navigation/BottomNav.svelte';
	import { getValidAccessToken } from '$lib/scripts/auth';
	import { fetchDatabaseData, refreshStaleDatabaseData } from '$lib/scripts/database';

	let { children } = $props();
	let ready = $state(false);
	let authPage = $derived(page.url.pathname === '/login' || page.url.pathname === '/create-account');

	onMount(() => {
		const refresh = () =>
			refreshStaleDatabaseData().catch((error) => console.error('Failed to refresh database data', error));

		const visibility = () => {
			if (document.visibilityState === 'visible') refresh();
		};

		void (async () => {
			if (authPage) {
				ready = true;
				return;
			}

			const token = await getValidAccessToken();

			if (!token) {
				location.assign('/login');
				return;
			}

			ready = true;
			fetchDatabaseData().catch((error) => console.error('Failed to load database data', error));

			refresh();

			document.addEventListener('visibilitychange', visibility);
			window.addEventListener('focus', refresh);
		})();

		return () => {
			document.removeEventListener('visibilitychange', visibility);
			window.removeEventListener('focus', refresh);
		};
	});
</script>

<svelte:head>
	<title>Better Music</title>
</svelte:head>

{#if ready}
	{#if authPage}
		{@render children()}
	{:else}
		<main class="px-4 pt-2 pb-[calc(6rem+env(safe-area-inset-bottom))]">
			{@render children()}
		</main>

		<BottomNav />
	{/if}
{/if}
