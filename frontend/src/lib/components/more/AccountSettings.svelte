<script lang="ts">
	import { onMount } from 'svelte';
	import { online } from 'svelte/reactivity/window';
	import AccountSettingRow from './AccountSettingRow.svelte';
	import AccountActionDialog from './AccountActionDialog.svelte';
	import DeleteIcon from '$lib/components/icons/DeleteIcon.svelte';
	import EmailIcon from '$lib/components/icons/EmailIcon.svelte';
	import LockIcon from '$lib/components/icons/LockIcon.svelte';
	import LogoutIcon from '$lib/components/icons/LogoutIcon.svelte';
	import { accountApi } from '$lib/scripts/api';
	import { canLogout, logout } from '$lib/scripts/auth';
	import { subscribeSyncChanges } from '$lib/scripts/sync-events';

	let accountEmail = $state('');
	let loadError = $state('');
	let actionDialog = $state<AccountActionDialog>();
	let logoutError = $state('');
	let loggingOut = $state(false);
	let logoutAllowed = $state(false);
	let actionsDisabled = $derived(!accountEmail || online.current !== true);
	let logoutDisabled = $derived(actionsDisabled || loggingOut || !logoutAllowed);

	onMount(() => {
		let mounted = true;
		let checking = false;
		const checkLogout = async () => {
			if (checking) return;
			checking = true;
			try {
				const allowed = await canLogout();
				if (mounted) logoutAllowed = allowed;
			} catch {
				if (mounted) logoutAllowed = false;
			} finally {
				checking = false;
			}
		};
		void checkLogout();
		const unsubscribe = subscribeSyncChanges(() => void checkLogout());
		// Locks can also belong to requests or tabs that do not publish sync events.
		const timer = setInterval(() => void checkLogout(), 500);
		return () => {
			mounted = false;
			clearInterval(timer);
			unsubscribe();
		};
	});

	onMount(async () => {
		if (online.current !== true) return;

		try {
			accountEmail = (await accountApi.get()).email;
		} catch (value) {
			console.error('Failed to load account', value);
			loadError = 'Could not load your account details.';
		}
	});

	async function signOut() {
		if (logoutDisabled) return;
		loggingOut = true;
		logoutError = '';
		try {
			await logout();
			location.assign('/login');
		} catch (error) {
			try {
				logoutAllowed = await canLogout();
			} catch {
				logoutAllowed = false;
			}
			if (logoutAllowed) logoutError = error instanceof Error ? error.message : 'Could not sign out. Try again.';
		} finally {
			loggingOut = false;
		}
	}
</script>

<section aria-labelledby="account-settings-heading" class="pb-4">
	<div class="mb-4">
		<h2
			id="account-settings-heading"
			class="text-base-content text-lg font-bold tracking-[0.14em] uppercase sm:text-xl"
		>
			Personal details
		</h2>
	</div>

	<div class="bg-base-200 divide-base-300 divide-y rounded-xl shadow-sm">
		<AccountSettingRow
			title="Email address"
			description={loadError || accountEmail || 'Loading...'}
			actionLabel="Change"
			icon={EmailIcon}
			disabled={actionsDisabled}
			onclick={() => actionDialog?.open('email', accountEmail)}
		/>

		<AccountSettingRow
			title="Password"
			description="Change your account password"
			actionLabel="Change"
			icon={LockIcon}
			tone="secondary"
			disabled={actionsDisabled}
			onclick={() => actionDialog?.open('password')}
		/>

		<AccountSettingRow
			title="Sign out"
			description={logoutError || 'Sign out of Better Music on this device'}
			actionLabel={loggingOut ? 'Signing out...' : 'Sign out'}
			icon={LogoutIcon}
			tone="accent"
			disabled={logoutDisabled}
			onclick={signOut}
		/>

		<AccountSettingRow
			title="Delete account"
			description="Permanently remove your library and account"
			actionLabel="Delete"
			icon={DeleteIcon}
			tone="error"
			disabled={actionsDisabled}
			onclick={() => actionDialog?.open('delete')}
		/>
	</div>
</section>

<AccountActionDialog bind:this={actionDialog} />
