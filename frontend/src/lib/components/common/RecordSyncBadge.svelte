<script lang="ts">
	import { getContext } from 'svelte';
	import RecordSyncSymbol from './RecordSyncSymbol.svelte';
	import { previewContext, type PreviewContext } from '$lib/components/preview/context';
	const preview = getContext<PreviewContext | undefined>(previewContext);
	import { getCurrentUserId } from '$lib/scripts/auth';
	import { getRecordSyncStatus, pendingMutations, syncStatus } from '$lib/scripts/database';
	import type { EntityId, MutationEntity } from '$lib/scripts/types';

	let { entity, id }: { entity: MutationEntity; id: EntityId } = $props();
	let status = $derived(
		preview?.recordStatus ??
			getRecordSyncStatus($pendingMutations, getCurrentUserId() ?? 0, entity, id, $syncStatus.state),
	);
</script>

<RecordSyncSymbol {status} />
