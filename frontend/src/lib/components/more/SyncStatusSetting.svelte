<script lang="ts">
	import CheckIcon from '$lib/components/icons/CheckIcon.svelte';
	import { pendingMutations, refreshDatabaseData, syncStatus } from '$lib/scripts/database';
	import SettingRow from './SettingRow.svelte';

	const formatter = new Intl.DateTimeFormat(undefined, {
		dateStyle: 'medium',
		timeStyle: 'short',
	});

	let lastSynced = $derived(
		$syncStatus.lastSyncedAt ? `Last synced ${formatter.format($syncStatus.lastSyncedAt)}` : 'Not synced yet',
	);
	let description = $derived.by(() => {
		const queued = $pendingMutations.length ? `${$pendingMutations.length} changes queued · ` : '';
		if ($syncStatus.state === 'offline') return `Offline · ${queued}${lastSynced}`;
		if ($syncStatus.state === 'pending') return `${queued}${lastSynced}`;
		if ($syncStatus.state === 'connecting') return 'Connecting to the server…';
		if ($syncStatus.state === 'error') return `Could not sync · ${lastSynced}`;
		return lastSynced;
	});

	async function syncNow() {
		try {
			await refreshDatabaseData();
		} catch (error) {
			console.error('Failed to sync the library', error);
		}
	}
</script>

<SettingRow title="Offline and sync status" {description} icon={CheckIcon} tone="accent">
	<button
		type="button"
		class="btn btn-accent btn-md shrink-0 rounded-full shadow-sm"
		disabled={$syncStatus.state === 'connecting' || $syncStatus.state === 'offline'}
		onclick={syncNow}
	>
		{#if $syncStatus.state === 'connecting'}
			<span class="loading loading-sm" aria-hidden="true"></span>
		{/if}
		{$syncStatus.state === 'connecting' ? 'Syncing…' : 'Sync now'}
	</button>
</SettingRow>
