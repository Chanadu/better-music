<script lang="ts">
	import ModalShell from '$lib/components/common/ModalShell.svelte';
	import CheckIcon from '$lib/components/icons/CheckIcon.svelte';
	import { correctionRecord, mutationLabel, type RecoveryPlan } from '$lib/scripts/queue-recovery';
	import type { UpdateArtistRequest, UpdateAlbumMutationPayload } from '$lib/scripts/types';

	let {
		dialog = $bindable(),
		plan,
		busy,
		error,
		onresolve,
	}: {
		dialog?: HTMLDialogElement;
		plan: RecoveryPlan | null;
		busy: boolean;
		error: string;
		onresolve: (
			choice: 'server' | 'local' | 'merge',
			payload?: UpdateArtistRequest | UpdateAlbumMutationPayload,
		) => void;
	} = $props();

	type Source = 'server' | 'local';
	const sources: Source[] = ['server', 'local'];

	let choice = $state<'server' | 'local' | 'merge'>('local');
	let selected = $state<Record<string, Source>>({});
	let local = $derived(plan ? (correctionRecord(plan) ?? plan.mutation.before) : null);
	let definitions = $derived(
		plan?.mutation.entity === 'artist' ?
			[['name', 'Name']]
		:	[
				['title', 'Title'],
				['year', 'Year'],
				['listened', 'Listened'],
				['rating', 'Rating'],
				['listened_at', 'Listening date'],
				['comment', 'Comment'],
			],
	);
	let deletesRecord = $derived(choice === 'local' && plan?.mutation.operation === 'delete');
	let unlistened = $derived(choice === 'merge' && read(selected.listened ?? 'local', 'listened') === false);

	$effect(() => {
		// Reset selections only when opening a different recovery plan.
		const current = plan;
		selected = Object.fromEntries(
			(current?.mutation.entity === 'artist' ?
				['name']
			:	['title', 'year', 'listened', 'rating', 'listened_at', 'comment']
			).map((key) => [key, 'local' as Source]),
		);
		choice = 'local';
	});

	function read(source: Source, key: string): unknown {
		const record = source === 'server' ? plan?.baseline : local;
		return (record as Record<string, unknown> | null | undefined)?.[key] ?? null;
	}

	function identical(key: string) {
		if (!plan?.baseline || !local) return false;
		if (key === 'listened_at') return display('server', key) === display('local', key);
		return Object.is(read('server', key), read('local', key));
	}

	function display(source: Source, key: string) {
		const value = read(source, key);
		if (key === 'listened') return value === true ? 'Listened' : 'Not listened';
		if (key === 'listened_at') return String(value ?? '').slice(0, 10) || '—';
		return String(value ?? '') || '—';
	}

	function chooseMode(mode: Source) {
		selected = Object.fromEntries(definitions.map(([key]) => [key, mode]));
		choice = mode;
	}

	function chooseField(key: string, source: Source) {
		if (choice !== 'merge' && choice === source) return;
		if (choice !== 'merge') selected = Object.fromEntries(definitions.map(([field]) => [field, choice as Source]));
		selected = { ...selected, [key]: source };
		choice = 'merge';
	}

	function payload() {
		const result: Record<string, unknown> = { cover_url: local?.cover_url, spotify_id: local?.spotify_id };
		for (const [key] of definitions)
			result[key] = read(choice === 'merge' ? (selected[key] ?? 'local') : 'local', key);

		if (plan?.mutation.entity === 'album') {
			result.artist_id = plan.mutation.payload.artist_id;
			if (result.listened === false) {
				result.rating = null;
				result.listened_at = null;
			}
		}

		return result as UpdateArtistRequest | UpdateAlbumMutationPayload;
	}
</script>

<ModalShell
	bind:dialog
	class={`resolution-dialog ${plan?.mutation.entity === 'album' ? 'max-w-3xl' : 'max-w-xl'}`}
	backdropDisabled={busy}
	oncancel={(event) => busy && event.preventDefault()}
>
	<div class="mb-2">
		<p class="text-secondary mb-1 text-[11px] font-semibold tracking-wider uppercase">Review conflict</p>

		<h2 class="truncate text-lg font-semibold" title={plan ? mutationLabel(plan.mutation) : undefined}>
			{plan ? mutationLabel(plan.mutation) : 'Resolve conflict'}
		</h2>
	</div>

	<div>
		<p class="text-base-content/60 mt-2 text-xs">
			{choice === 'merge' ? 'Choose a value for each field.' : 'Select a whole column or individual values.'}
		</p>

		{#if !plan?.baseline}
			<p class="text-warning mt-2 text-xs" role="status">
				Deleted on server. {choice === 'server' || deletesRecord ? 'Keep it deleted.' : 'Restore this record.'}
			</p>
		{/if}

		{#if plan?.mutation.operation === 'delete' && choice !== 'server'}
			<p class="text-base-content/70 mt-2 text-xs">
				{deletesRecord ?
					'Your local change deletes this record.'
				:	'Choosing fields keeps a record instead of deleting it.'}
			</p>
		{/if}

		<div class="border-base-300 mt-2 overflow-hidden rounded-xl border">
			<div
				class="comparison-heading comparison-grid bg-base-200 border-base-300 border-b p-3 text-xs font-semibold"
			>
				<span class="text-base-content/50 self-center">Field</span>

				{#each sources as source}
					<div class="min-w-0 text-center">
						<button
							type="button"
							class="source-select btn btn-sm btn-secondary hover-lift w-full"
							class:btn-soft={choice !== source}
							aria-label={`Select all ${source} values`}
							aria-pressed={choice === source}
							disabled={busy || !plan || (source === 'local' && !local)}
							onclick={() => chooseMode(source)}
						>
							{#if choice === source}
								<CheckIcon class="size-3.5 shrink-0" />
							{/if}

							{source === 'server' ? 'Server' : 'Local'}
						</button>
					</div>
				{/each}
			</div>

			<dl class="comparison-fields p-3">
				{#each definitions as [key, label]}
					{@const same = identical(key)}
					<div class="comparison-row comparison-grid" class:comment-row={key === 'comment' && !same}>
						<dt
							class="field-label text-base-content/60 self-center text-xs"
							class:col-span-full={key === 'comment' && !same}
						>
							{#if key === 'comment' && !same}
								<span class="grid grid-cols-2 gap-2">
									<span>Server comment</span>

									<span>Local comment</span>
								</span>
							{:else}
								{label}
							{/if}
						</dt>

						{#if same}
							<dd class="col-span-2 min-w-0">
								<div
									class="shared-value field-option text-base-content/75 rounded-lg px-2 py-1.5 text-center text-sm"
									class:comment-option={key === 'comment'}
									class:empty-comment={key === 'comment' && !read('server', key)}
									title={String(read('server', key) ?? '')}
								>
									<span
										class:truncate={key !== 'comment'}
										class:comment-value={key === 'comment'}
										class="block"
									>
										{display('server', key)}
									</span>
								</div>
							</dd>
						{:else}
							{#each sources as source}
								{@const available = source === 'server' ? Boolean(plan?.baseline) : Boolean(local)}
								{@const active =
									(choice === 'merge' ? selected[key] : choice) === source &&
									available &&
									!(unlistened && (key === 'rating' || key === 'listened_at'))}

								<dd class="min-w-0">
									<button
										type="button"
										class={`field-option btn btn-sm hover-lift h-auto w-full justify-start rounded-lg border px-2 py-1.5 text-left text-sm font-normal whitespace-normal ${active ? 'border-primary bg-primary/10 hover:bg-primary/20' : 'border-base-300 hover:bg-base-200'}`}
										class:comment-option={key === 'comment'}
										aria-label={`Use ${source} ${label.toLowerCase()}`}
										aria-pressed={active}
										disabled={busy ||
											!available ||
											(unlistened && (key === 'rating' || key === 'listened_at'))}
										title={available ? String(read(source, key) ?? '') : 'Deleted on the server'}
										onclick={() => chooseField(key, source)}
									>
										<span class="flex w-full items-start gap-1">
											<span
												class="min-w-0 flex-1"
												class:truncate={key !== 'comment'}
												class:comment-value={key === 'comment'}
											>
												{available ? display(source, key) : '—'}
											</span>

											{#if active}
												<CheckIcon class="text-primary mt-0.5 size-3 shrink-0" />
											{/if}
										</span>
									</button>
								</dd>
							{/each}
						{/if}
					</div>
				{/each}
			</dl>
		</div>

		{#if error}
			<div class="alert alert-error alert-soft mt-2 p-2 text-xs" role="alert">
				<span>{error}</span>
			</div>
		{/if}

		<div class="modal-action gap-4 px-2">
			<form method="dialog" class="flex-1">
				<button class="btn hover-lift btn-error w-full" disabled={busy}>Cancel</button>
			</form>

			<button
				type="button"
				class="btn hover-lift btn-primary flex-1"
				disabled={busy || !plan || (choice === 'merge' && !local)}
				onclick={() => onresolve(choice, choice === 'server' || deletesRecord ? undefined : payload())}
			>
				{#if busy}
					<span class="loading loading-spinner loading-xs" aria-hidden="true"></span>
				{/if}

				{busy ? 'Saving…'
				: choice === 'server' ? 'Keep server'
				: choice === 'local' ? 'Use local'
				: 'Save selection'}
			</button>
		</div>
	</div>
</ModalShell>

<style>
	:global(.resolution-dialog) {
		padding: 1.5rem;
	}

	.comparison-grid {
		display: grid;
		grid-template-columns: minmax(0, 0.55fr) repeat(2, minmax(0, 1fr));
		gap: 0.75rem;
	}

	.comparison-row + .comparison-row {
		margin-top: 0.5rem;
		padding-top: 0.5rem;
		border-top: 1px solid color-mix(in oklch, var(--color-base-content) 7%, transparent);
	}
	.field-option {
		min-height: 2.5rem;
	}

	.shared-value {
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.shared-value.empty-comment {
		min-height: 2.5rem;
	}

	.field-option:disabled {
		opacity: 0.5;
	}

	.comment-row {
		grid-template-columns: repeat(2, minmax(0, 1fr));
		gap: 0.375rem 0.75rem;
		padding-top: 0.5rem;
	}

	.comment-option {
		min-height: 4.5rem;
	}
	.comment-value {
		display: -webkit-box;
		-webkit-box-orient: vertical;
		-webkit-line-clamp: 3;
		line-clamp: 3;
		overflow: hidden;
		white-space: pre-wrap;
		overflow-wrap: anywhere;
	}

	@media (max-height: 650px) {
		:global(.resolution-dialog) {
			padding: 0.625rem;
		}
		.comparison-heading {
			padding: 0 0.5rem;
		}
		.comment-value {
			line-height: 1rem;
		}
		.comparison-fields {
			padding: 0.5rem;
		}
		.comparison-grid {
			gap: 0.5rem;
		}
		.comparison-row + .comparison-row {
			margin-top: 0.125rem;
			padding-top: 0.125rem;
		}
		.field-option {
			min-height: 1.5rem;
			padding: 0.125rem 0.375rem;
			font-size: 0.75rem;
		}
		.field-label {
			font-size: 0.6875rem;
		}
		.comment-row {
			gap: 0.25rem 0.5rem;
			padding-top: 0.25rem;
		}
		.comment-option {
			min-height: 3rem;
		}
	}
</style>
