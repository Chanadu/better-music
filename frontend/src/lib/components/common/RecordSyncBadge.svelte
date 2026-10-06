<script lang="ts">
	import { getCurrentUserId } from '$lib/scripts/auth';
	import { getRecordSyncStatus, pendingMutations } from '$lib/scripts/database';
	import type { EntityId, MutationEntity } from '$lib/scripts/types';
	let {
		entity,
		id,
		variant = 'badge',
	}: { entity: MutationEntity; id: EntityId; variant?: 'badge' | 'dot' } = $props();
	let status = $derived(getRecordSyncStatus($pendingMutations, getCurrentUserId() ?? 0, entity, id));
</script>

{#if status === 'pending' && variant === 'dot'}
	<span
		class="bg-warning inline-block size-2 shrink-0 rounded-full align-middle"
		role="status"
		aria-label="Pending: waiting to sync"
		title="Pending: waiting to sync"
	></span>
{:else if status !== 'synced'}
	<span class="badge badge-outline badge-sm" role="status"
		>{status === 'pending' ? 'Pending'
		: status === 'syncing' ? 'Syncing'
		: status === 'failed' ? 'Failed'
		: 'Conflict'}</span
	>
{/if}
