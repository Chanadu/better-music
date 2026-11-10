<script lang="ts">
	import { getContext } from 'svelte';
	import RecordSyncSymbol from './RecordSyncSymbol.svelte';
	import { previewContext, type PreviewContext } from '$lib/components/preview/context';

	const preview = getContext<PreviewContext | undefined>(previewContext);
	import { getRecordSyncStatus, pendingRecords, syncStatus } from '$lib/scripts/database';
	import type { EntityId, LibraryEntity } from '$lib/scripts/types';

	let { entity, id }: { entity: LibraryEntity; id: EntityId } = $props();
	let status = $derived(preview?.recordStatus ?? getRecordSyncStatus($pendingRecords, entity, id, $syncStatus.state));
</script>

<RecordSyncSymbol {status} />
