<script lang="ts">
	import '../app.css';
	import { afterNavigate, beforeNavigate, preloadCode } from '$app/navigation';
	import { page } from '$app/state';
	import { onMount, tick } from 'svelte';
	import StorageErrorAlert from '$lib/components/common/StorageErrorAlert.svelte';
	import BottomNav from '$lib/components/navigation/BottomNav.svelte';
	import { appSettings } from '$lib/scripts/app-settings.svelte';
	import { getValidAccessToken, hasStoredSession } from '$lib/scripts/auth';
	import {
		getDatabaseData,
		markOffline,
		markSyncError,
		refreshDatabaseSafely,
		storageProblem,
		retryLocalStorage,
	} from '$lib/scripts/database';

	let { children } = $props();
	let ready = $state(false);
	let previewPage = $derived(page.url.pathname === '/preview');
	let authPage = $derived(page.url.pathname === '/login' || page.url.pathname === '/create-account');

	async function preloadLibraryPages() {
		if (!navigator.onLine || authPage || previewPage) return;
		const results = await Promise.allSettled(
			['/', '/artists', '/albums', '/artist', '/album', '/listen', '/more'].map((path) => preloadCode(path)),
		);
		for (const result of results) {
			if (result.status === 'rejected') console.warn('Could not preload a library page', result.reason);
		}
	}

	// The preview has no session initialization. Remount the layout when crossing
	// its boundary so normal library routes always run their authentication flow.
	beforeNavigate(({ to, cancel }) => {
		if (to && previewPage !== (to.url.pathname === '/preview')) {
			cancel();
			location.assign(to.url.href);
		}
	});

	afterNavigate(() => {
		if (ready && !authPage && !previewPage) void refreshDatabaseSafely();
	});

	onMount(() => {
		appSettings.load();
		void preloadLibraryPages();

		const visibility = () => {
			if (document.visibilityState === 'visible') void refreshDatabaseSafely();
		};
		const focus = () => void refreshDatabaseSafely();
		const reconnect = () => {
			void preloadLibraryPages();
			void refreshDatabaseSafely();
		};

		void (async () => {
			if (authPage || previewPage) {
				ready = true;
				return;
			}

			const cached = await getDatabaseData();
			const token = await getValidAccessToken();

			if (!token && (!cached || !hasStoredSession()) && !$storageProblem) {
				location.assign('/login');
				return;
			}
			if (!token) {
				if (navigator.onLine) markSyncError();
				else markOffline();
			}

			ready = true;
			await tick();
			if (token) void refreshDatabaseSafely();

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
	<title>
		Better Music
	</title>
</svelte:head>

{#if ready}
	{#if authPage || previewPage}
		{@render children()}
	{:else}

		<main
			class="px-4 pt-[calc(0.5rem+env(safe-area-inset-top))] pb-[calc(4rem+env(safe-area-inset-bottom))] sm:pb-[calc(6rem+env(safe-area-inset-bottom))]"
		>
			{#if $storageProblem}
				<div class="my-4">
					<StorageErrorAlert message={$storageProblem} onretry={retryLocalStorage} />
				</div>
			{/if}

			{@render children()}
		</main>

		<BottomNav />
	{/if}
{/if}
