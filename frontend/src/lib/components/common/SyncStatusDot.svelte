<script lang="ts">
	import type { RecordSyncStatus } from '$lib/scripts/types';

	let { status, label }: { status: RecordSyncStatus; label?: string } = $props();
	const labels = {
		synced: 'Synced',
		pending: 'Pending: waiting to sync',
		syncing: 'Syncing',
		paused: 'Paused: synchronization cannot continue',
		failed: 'Failed: review in Settings',
		conflict: 'Conflicted: the server record changed',
		blocked: 'Blocked: resolve an earlier change or dependency in Settings',
	};
	const colors = {
		synced: '',
		pending: 'bg-warning',
		syncing: 'bg-info animate-pulse',
		paused: 'bg-base-content/45',
		failed: 'bg-error',
		conflict: 'bg-warning',
		blocked: 'bg-base-content/60',
	};
</script>

{#if status !== 'synced'}
	<span
		class={`${colors[status]} inline-block size-2 shrink-0 rounded-full align-middle`}
		role="status"
		aria-label={label ?? labels[status]}
		title={label ?? labels[status]}
	>
	</span>
{/if}
