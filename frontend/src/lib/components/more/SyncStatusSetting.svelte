<script lang="ts">
	import CheckIcon from '$lib/components/icons/CheckIcon.svelte';
	import SyncIcon from '$lib/components/icons/SyncIcon.svelte';
	import {
		syncStatus,
		syncProblems,
		resolveSyncProblem,
		storageProblem,
		pendingRecords,
		refreshDatabaseData,
	} from '$lib/scripts/database';
	import SettingRow from './SettingRow.svelte';

	let busy = $state(false);
	let error = $state('');
	const formatter = new Intl.DateTimeFormat(undefined, { dateStyle: 'medium', timeStyle: 'short' });
	let message = $derived(error || $storageProblem || $syncStatus.error || '');
	let description = $derived(
		message ? `Sync paused · ${message}`
		: $syncStatus.state === 'offline' ? 'Offline · changes stay saved on this device'
		: $syncStatus.state === 'connecting' ? 'Syncing your library...'
		: $pendingRecords.length ? `${$pendingRecords.length} records waiting to sync`
		: 'Your library is up to date',
	);
	async function perform(action: () => Promise<unknown>) {
		busy = true;
		error = '';
		try {
			await action();
		} catch (value) {
			error = value instanceof Error ? value.message : String(value);
		} finally {
			busy = false;
		}
	}
</script>

<div>
	<SettingRow title="Library sync" {description} icon={CheckIcon} tone="accent">
		{#snippet descriptionContent()}
			<span class:text-error={Boolean(message)} role={message ? 'alert' : undefined}>{description}</span>
			· Last synced: {$syncStatus.lastSyncedAt ? formatter.format($syncStatus.lastSyncedAt) : 'Not yet'}
		{/snippet}
		<button
			type="button"
			class="btn hover-lift btn-accent btn-md shrink-0 rounded-full shadow-sm"
			disabled={busy || $syncStatus.state === 'offline'}
			onclick={() => perform(refreshDatabaseData)}
		>
			<SyncIcon class="size-4" />{busy ? 'Syncing...' : 'Sync now'}
		</button>
	</SettingRow>
	{#if $syncProblems.length}
		<ul class="space-y-3 px-4 pb-5 sm:px-5">
			{#each $syncProblems as problem (`${problem.kind}:${problem.id}`)}
				<li class="text-sm">
					<p><strong>{problem.label}</strong>: {problem.message}</p>
					{#if problem.deleted}
						<button
							class="btn btn-sm btn-outline"
							disabled={busy}
							onclick={() => perform(() => resolveSyncProblem(problem, true))}>Undo deletion</button
						>
					{:else}
						<a class="btn btn-sm btn-outline" href={problem.href}>Open record</a>
						<button
							class="btn btn-sm btn-outline"
							disabled={busy}
							onclick={() => perform(() => resolveSyncProblem(problem, false))}>Remove record</button
						>
					{/if}
				</li>
			{/each}
		</ul>
	{/if}
	{#if message}
		<p class="px-4 pb-5 text-sm sm:px-5">
			Your edits remain saved locally. Correct the affected record or sign in to the same account, then retry.
			Export your library before clearing browser data.
		</p>
		{#if /authenticat|sign in/i.test(message)}
			<a class="btn btn-outline mx-4 mb-4" href="/login">Sign in again</a>
		{/if}
	{/if}
</div>
