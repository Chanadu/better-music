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
	import { canChangeAccount, logout } from '$lib/scripts/auth';
	import { syncStatus } from '$lib/scripts/database';

	let accountEmail = $state('');
	let loadError = $state('');
	let actionDialog = $state<AccountActionDialog>();
	let logoutError = $state('');
	let loggingOut = $state(false);
	let actionsAllowed = $state(false);
	let actionsDisabled = $derived(!accountEmail || online.current !== true || !actionsAllowed);
	let logoutDisabled = $derived(actionsDisabled || loggingOut);

	const checkAccountActions = () => canChangeAccount().catch(() => false);

	onMount(() => {
		let mounted = true;
		let checking = false;
		let loadingAccount = false;
		let lastSync: number | null = null;

		const loadAccount = async () => {
			if (accountEmail || loadingAccount || online.current !== true) return;

			loadingAccount = true;
			try {
				const { email } = await accountApi.get();
				if (mounted) {
					accountEmail = email;
					loadError = '';
				}
			} catch {
				if (mounted) loadError = 'Could not load your account details.';
			} finally {
				loadingAccount = false;
			}
		};

		const checkActions = async () => {
			if (checking) return;

			checking = true;
			try {
				const allowed = await checkAccountActions();
				if (mounted) actionsAllowed = allowed;
			} finally {
				checking = false;
			}
		};

		void loadAccount();

		const unsubscribe = syncStatus.subscribe(({ lastSyncedAt }) => {
			void checkActions();

			if (lastSyncedAt !== null && lastSyncedAt !== lastSync) {
				lastSync = lastSyncedAt;
				void loadAccount();
			}
		});

		const timer = setInterval(() => void checkActions(), 500);

		return () => {
			mounted = false;
			clearInterval(timer);
			unsubscribe();
		};
	});

	async function signOut() {
		if (logoutDisabled) return;

		loggingOut = true;
		logoutError = '';
		try {
			await logout();
			location.assign('/login');
		} catch (error) {
			actionsAllowed = await checkAccountActions();

			if (actionsAllowed) logoutError = error instanceof Error ? error.message : 'Could not sign out. Try again.';
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

	{#if !actionsAllowed}
		<p class="text-base-content/65 mb-4 text-sm" role="status">
			Sync your library and wait for active requests before changing account information or signing out. Use Sync
			now in Library sync.
		</p>
	{/if}

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
