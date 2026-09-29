<script lang="ts">
	import '../app.css';
	import { page } from '$app/state';
	import { onMount, tick } from 'svelte';
	import BottomNav from '$lib/components/navigation/BottomNav.svelte';
	import { appSettings } from '$lib/scripts/app-settings.svelte';
	import { getValidAccessToken, hasStoredSession } from '$lib/scripts/auth';
	import { loadCachedDatabase, refreshStaleDatabaseData } from '$lib/scripts/database';

	let { children } = $props();
	let ready = $state(false);
	let authPage = $derived(page.url.pathname === '/login' || page.url.pathname === '/create-account');

	onMount(() => {
		appSettings.load();

		const refresh = async () => {
			if (!navigator.onLine) return;

			const token = await getValidAccessToken();
			if (!token) {
				if (!hasStoredSession()) location.assign('/login');
				return;
			}

			await refreshStaleDatabaseData();
		};

		const refreshSafely = () => refresh().catch((error) => console.error('Failed to refresh database data', error));

		const visibility = () => {
			if (document.visibilityState === 'visible') refreshSafely();
		};

		void (async () => {
			if (authPage) {
				ready = true;
				return;
			}

			const cached = await loadCachedDatabase();
			const token = await getValidAccessToken();

			if (!token && (!cached || !hasStoredSession())) {
				location.assign('/login');
				return;
			}

			ready = true;
			await tick();
			if (token)
				refreshStaleDatabaseData().catch((error) => console.error('Failed to load database data', error));

			document.addEventListener('visibilitychange', visibility);
			window.addEventListener('focus', refreshSafely);
			window.addEventListener('online', refreshSafely);
		})();

		return () => {
			document.removeEventListener('visibilitychange', visibility);
			window.removeEventListener('focus', refreshSafely);
			window.removeEventListener('online', refreshSafely);
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
		<main
			class="px-4 pt-[calc(0.5rem+env(safe-area-inset-top))] pb-[calc(4rem+env(safe-area-inset-bottom))] sm:pb-[calc(6rem+env(safe-area-inset-bottom))]"
		>
			{@render children()}
		</main>

		<BottomNav />
	{/if}
{/if}
