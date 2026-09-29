<script lang="ts">
	import '../app.css';
	import { page } from '$app/state';
	import { onMount, tick } from 'svelte';
	import BottomNav from '$lib/components/navigation/BottomNav.svelte';
	import { appSettings } from '$lib/scripts/app-settings.svelte';
	import { getValidAccessToken, hasStoredSession } from '$lib/scripts/auth';
	import {
		loadCachedDatabase,
		markConnecting,
		markOffline,
		markSyncError,
		refreshDatabaseData,
		refreshStaleDatabaseData,
	} from '$lib/scripts/database';

	let { children } = $props();
	let ready = $state(false);
	let authPage = $derived(page.url.pathname === '/login' || page.url.pathname === '/create-account');

	onMount(() => {
		appSettings.load();

		const refreshDatabaseSafely = async (force = false) => {
			try {
				if (!navigator.onLine) {
					markOffline();
					return;
				}

				markConnecting();
				const token = await getValidAccessToken();
				if (!token) {
					if (!hasStoredSession()) location.assign('/login');
					else markSyncError();
					return;
				}

				if (force) await refreshDatabaseData();
				else await refreshStaleDatabaseData();
			} catch (error) {
				if (navigator.onLine) markSyncError();
				else markOffline();
				console.error('Failed to refresh database data', error);
			}
		};

		const visibility = () => {
			if (document.visibilityState === 'visible') void refreshDatabaseSafely();
		};
		const focus = () => void refreshDatabaseSafely();
		const reconnect = () => void refreshDatabaseSafely(true);

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
			if (!token) {
				if (navigator.onLine) markSyncError();
				else markOffline();
			}

			ready = true;
			await tick();
			if (token) void refreshDatabaseSafely(true);

			document.addEventListener('visibilitychange', visibility);
			window.addEventListener('focus', focus);
			window.addEventListener('online', reconnect);
			window.addEventListener('offline', markOffline);
		})();

		return () => {
			document.removeEventListener('visibilitychange', visibility);
			window.removeEventListener('focus', focus);
			window.removeEventListener('online', reconnect);
			window.removeEventListener('offline', markOffline);
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
