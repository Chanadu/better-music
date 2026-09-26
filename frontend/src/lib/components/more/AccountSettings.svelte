<script lang="ts">
	import { onMount } from 'svelte';
	import AccountSettingRow from './AccountSettingRow.svelte';
	import AccountActionDialog from './AccountActionDialog.svelte';
	import DeleteIcon from '$lib/components/icons/DeleteIcon.svelte';
	import EmailIcon from '$lib/components/icons/EmailIcon.svelte';
	import LockIcon from '$lib/components/icons/LockIcon.svelte';
	import LogoutIcon from '$lib/components/icons/LogoutIcon.svelte';
	import { accountApi } from '$lib/scripts/api';
	import { logout } from '$lib/scripts/auth';

	let accountEmail = $state('');
	let loadError = $state('');
	let actionDialog = $state<AccountActionDialog>();
	let loggingOut = $state(false);

	onMount(async () => {
		try {
			accountEmail = (await accountApi.get()).email;
		} catch (value) {
			console.error('Failed to load account', value);
			loadError = 'Could not load your account details.';
		}
	});

	async function signOut() {
		loggingOut = true;
		await logout();
		location.assign('/login');
	}
</script>

<section aria-labelledby="account-settings-heading" class="pb-4">
	<div class="mb-4">
		<h2 id="account-settings-heading" class="text-accent text-lg font-bold tracking-[0.14em] uppercase sm:text-xl">
			Personal details
		</h2>
	</div>

	<div class="bg-base-200 divide-base-300 divide-y rounded-xl shadow-sm">
		<AccountSettingRow
			title="Email address"
			description={loadError || accountEmail || 'Loading...'}
			actionLabel="Change"
			icon={EmailIcon}
			disabled={!accountEmail}
			onclick={() => actionDialog?.open('email', accountEmail)}
		/>
		<AccountSettingRow
			title="Password"
			description="Change your account password"
			actionLabel="Change"
			icon={LockIcon}
			tone="accent"
			onclick={() => actionDialog?.open('password')}
		/>
		<AccountSettingRow
			title="Sign out"
			description="Sign out of Better Music on this device"
			actionLabel={loggingOut ? 'Signing out...' : 'Sign out'}
			icon={LogoutIcon}
			tone="accent"
			disabled={loggingOut}
			onclick={signOut}
		/>
		<AccountSettingRow
			title="Delete account"
			description="Permanently remove your library and account"
			actionLabel="Delete"
			icon={DeleteIcon}
			tone="error"
			onclick={() => actionDialog?.open('delete')}
		/>
	</div>
</section>

<AccountActionDialog bind:this={actionDialog} />
