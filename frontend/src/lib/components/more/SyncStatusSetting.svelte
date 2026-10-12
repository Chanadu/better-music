<script lang="ts">
	import { tick } from 'svelte';
	import CheckIcon from '$lib/components/icons/CheckIcon.svelte';
	import SyncIcon from '$lib/components/icons/SyncIcon.svelte';
	import EditArtistModal from '$lib/components/artists/EditArtistModal.svelte';
	import EditAlbumModal from '$lib/components/albums/EditAlbumModal.svelte';
	import SyncQueueList from './SyncQueueList.svelte';
	import DiscardMutationModal from './DiscardMutationModal.svelte';
	import { database, pendingMutations, refreshDatabaseData, syncStatus, storageProblem } from '$lib/scripts/database';
	import { getChangesNeedingAttention } from '$lib/scripts/queue-state';
	import {
		applyRecovery,
		correctionRecord,
		prepareRecovery,
		reconcileUnknownMutation,
		retryFailedMutation,
		type RecoveryPlan,
	} from '$lib/scripts/queue-recovery';
	import type {
		Artist,
		Album,
		PendingMutation,
		UpdateArtistRequest,
		UpdateAlbumMutationPayload,
	} from '$lib/scripts/types';
	import SettingRow from './SettingRow.svelte';

	let attention = $derived(getChangesNeedingAttention($pendingMutations));
	let busy = $state(false);
	let recoveryDisabled = $derived(busy || $syncStatus.state === 'connecting');
	let sendingDisabled = $derived(recoveryDisabled || $syncStatus.state === 'offline');
	let error = $state('');
	let notice = $state('');
	let syncMessage = $derived(
		error ||
			$storageProblem ||
			($syncStatus.state !== 'offline' ? $syncStatus.pauseReason || $syncStatus.error : undefined),
	);
	let discardPlan = $state.raw<RecoveryPlan | null>(null);
	let correctionPlan = $state.raw<RecoveryPlan | null>(null);
	let discardDialog = $state<HTMLDialogElement>();
	let editDialog = $state<HTMLDialogElement>();
	let editedRecord = $derived(correctionPlan ? correctionRecord(correctionPlan) : null);
	let artistForAlbum = $derived(
		editedRecord && 'artist_id' in editedRecord ?
			$database?.artists.find((item) => item.id === editedRecord.artist_id)
		:	undefined,
	);
	let waiting = $derived(
		$pendingMutations.filter((item) => ['pending', 'syncing'].includes(item.status) && !attention.includes(item))
			.length,
	);
	let needsAttention = $derived(attention.length > 0);
	let statusDescription = $derived(
		$syncStatus.state === 'connecting' ? 'Syncing your library…'
		: syncMessage ? `Sync paused · ${syncMessage}`
		: $syncStatus.state === 'offline' ? 'Offline'
		: needsAttention ?
			`${attention.length} change${attention.length === 1 ? '' : 's'} need${attention.length === 1 ? 's' : ''} attention`
		: waiting ? `${waiting} change${waiting === 1 ? '' : 's'} waiting to sync`
		: 'Your library is up to date',
	);

	const formatter = new Intl.DateTimeFormat(undefined, { dateStyle: 'medium', timeStyle: 'short' });
	let description = $derived(
		`${statusDescription} · Last synced: ${$syncStatus.lastSyncedAt ? formatter.format($syncStatus.lastSyncedAt) : 'Not yet'}`,
	);

	async function perform(action: () => Promise<unknown>) {
		if (busy) return;
		busy = true;
		error = '';
		notice = '';
		try {
			await action();
		} catch (value) {
			console.error('Library sync action failed', value);
			error = value instanceof Error ? value.message : 'Recovery could not complete. Try again.';
		} finally {
			busy = false;
		}
	}
	async function openDiscard(item: PendingMutation) {
		await perform(async () => {
			discardPlan = await prepareRecovery(item.mutationId, 'discard');
			await tick();
			discardDialog?.showModal();
		});
	}
	async function openCorrection(item: PendingMutation) {
		await perform(async () => {
			const plan = await prepareRecovery(item.mutationId, 'correct');
			const record = correctionRecord(plan);
			if (!record)
				throw new Error('The original record is unavailable. Discard the edit or resolve its missing record.');
			if ('artist_id' in record && !$database?.artists.some((item) => item.id === record.artist_id))
				throw new Error(
					'Restore the required artist before correcting this album. The album change remains recoverable in the queue.',
				);
			correctionPlan = plan;
			await tick();
			editDialog?.showModal();
		});
	}
	async function saveCorrection(payload: UpdateArtistRequest | UpdateAlbumMutationPayload) {
		if (!correctionPlan) throw new Error('Open the correction again.');
		const result = await applyRecovery(correctionPlan, payload);
		notice = 'Correction saved on this device. Later edits were preserved.';
		if (!result.corrected) throw new Error('The corrected record is unavailable. Review the remaining queue.');
		return result.corrected;
	}
</script>

<div>
	<SettingRow title="Library sync" {description} icon={CheckIcon} tone="accent">
		{#snippet descriptionContent()}
			<span class:text-error={Boolean(syncMessage)} role={syncMessage ? 'alert' : undefined}
			>
				{statusDescription}
			</span>
			· Last synced: {$syncStatus.lastSyncedAt ? formatter.format($syncStatus.lastSyncedAt) : 'Not yet'}
		{/snippet}

		<button
			type="button"
			class="btn hover-lift btn-accent btn-md shrink-0 rounded-full shadow-sm"
			disabled={sendingDisabled}
			onclick={() => perform(() => refreshDatabaseData())}
		>
			{#if $syncStatus.state === 'connecting'}
				<span class="loading loading-sm" aria-hidden="true">
				</span>
			{:else}
				<SyncIcon class="size-4" />
			{/if}
			{$syncStatus.state === 'connecting' ? 'Syncing…' : 'Sync now'}
		</button>
	</SettingRow>

	{#if needsAttention || notice}
		<div class="space-y-3 px-4 pb-5 sm:px-5">
			{#if notice}
				<p role="status" class="text-success text-sm">
					{notice}
				</p>
			{/if}

			<SyncQueueList
				mutations={$pendingMutations}
				status={$syncStatus}
				{recoveryDisabled}
				{sendingDisabled}
				onretry={(item) => perform(() => retryFailedMutation(item.mutationId))}
				onreconcile={(item) => perform(() => reconcileUnknownMutation(item.mutationId))}
				oncorrect={openCorrection}
				ondiscard={openDiscard}
			/>
		</div>
	{/if}
</div>

<DiscardMutationModal
	bind:dialog={discardDialog}
	plan={discardPlan}
	{busy}
	{error}
	onconfirm={() =>
		perform(async () => {
			if (discardPlan) await applyRecovery(discardPlan);
			discardDialog?.close();
			notice = 'Local change discarded. The reviewed consequences were applied.';
			discardPlan = null;
		})}
/>

{#if correctionPlan && editedRecord}
	{#if 'name' in editedRecord}
		<EditArtistModal
			artist={editedRecord as Artist}
			bind:dialog={editDialog}
			mode="correct"
			recoveryMessages={correctionPlan.messages}
			onsave={async (payload) => (await saveCorrection(payload)) as Artist}
			onupdated={() => (correctionPlan = null)}
		/>
	{:else if artistForAlbum}

			<EditAlbumModal
			album={editedRecord as Album}
			artist={artistForAlbum}
			bind:dialog={editDialog}
			mode="correct"
			recoveryMessages={correctionPlan.messages}
			onsave={async (payload) => (await saveCorrection(payload)) as Album}
			onupdated={() => (correctionPlan = null)}
			/>
		{/if}
{/if}
