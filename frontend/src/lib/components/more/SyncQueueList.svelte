<script lang="ts">
	import { onMount } from 'svelte';
	import type { SyncStatus } from '$lib/scripts/database';
	import type { PendingMutation } from '$lib/scripts/types';
	import { getBlockingMutations, hasUnknownOutcome, getChangesNeedingAttention } from '$lib/scripts/queue-state';
	import { mutationLabel } from '$lib/scripts/queue-recovery';

	let {
		mutations,
		status,
		recoveryDisabled,
		sendingDisabled,
		onretry,
		onreconcile,
		oncorrect,
		onresolve = () => {},
		ondiscard,
	}: {
		mutations: PendingMutation[];
		status: SyncStatus;
		recoveryDisabled: boolean;
		sendingDisabled: boolean;
		onretry: (item: PendingMutation) => void;
		onreconcile: (item: PendingMutation) => void;
		oncorrect: (item: PendingMutation) => void;
		onresolve?: (item: PendingMutation) => void;
		ondiscard: (item: PendingMutation) => void;
	} = $props();

	let attention = $derived(getChangesNeedingAttention(mutations));
	const formatter = new Intl.DateTimeFormat(undefined, { dateStyle: 'medium', timeStyle: 'short' });
	const date = (value: number) => formatter.format(value);
	let now = $state(Date.now());
	onMount(() => {
		const timer = setInterval(() => (now = Date.now()), 1000);
		return () => clearInterval(timer);
	});

	function details(item: PendingMutation) {
		if (hasUnknownOutcome(item))
			return 'We need to check whether this change was saved online before you can edit or remove it.';
		const blockers = getBlockingMutations(item, mutations);
		if (blockers.length) return `Waiting for ${blockers.map(mutationLabel).join(', ')} to sync.`;
		if (item.blockedReason) return item.blockedReason;
		if (item.nextAttemptAt && item.nextAttemptAt > now) return `Will retry after ${date(item.nextAttemptAt)}.`;
		if (item.status === 'conflict')
			return 'This item was changed elsewhere. Compare the versions and choose how to resolve it.';
		if (item.failure) return item.failure.message;
		if (status.state === 'offline') return 'Saved on this device. Will sync when you reconnect.';
		return 'Saved on this device. Waiting to sync.';
	}
</script>

{#if attention.length}
	<details
		class="group/review border-primary/20 bg-primary/5 rounded-box open:border-primary/30 overflow-hidden border transition-colors"
	>
		<summary
			class="text-primary hover:bg-primary/10 group-open/review:bg-primary/5 focus-visible:outline-primary flex cursor-pointer list-none items-center justify-between gap-3 px-4 py-3 text-sm font-semibold transition-colors focus-visible:outline-2 focus-visible:-outline-offset-2 [&::-webkit-details-marker]:hidden"
		>
			<span class="flex items-center gap-2.5">
				Review changes

				<span class="badge badge-sm badge-secondary badge-soft font-semibold">
					{attention.length}
				</span>
			</span>

			<svg
				class="text-primary bg-primary/10 size-7 shrink-0 rounded-full p-1.5 transition-transform group-open/review:rotate-180"
				viewBox="0 0 24 24"
				fill="none"
				stroke="currentColor"
				stroke-width="2"
				stroke-linecap="round"
				stroke-linejoin="round"
				aria-hidden="true"
			>
				<path d="m6 9 6 6 6-6" />
			</svg>
		</summary>

		<ul class="border-base-300 divide-base-300 divide-y border-t">
			{#each attention as item (item.mutationId)}
				<li class="grid gap-x-4 gap-y-2 px-4 py-3 sm:grid-cols-[minmax(0,1fr)_auto]">
					<div class="flex items-center gap-2 sm:col-start-1 sm:row-start-1">
						<p class="min-w-0 flex-1 text-sm font-semibold break-words">
							{mutationLabel(item)}

							<span class="text-base-content/50 ml-1 text-xs font-normal">
								· {item.operation === 'create' ? 'Added'
								: item.operation === 'delete' ? 'Removed'
								: 'Edited'}
							</span>
						</p>
					</div>

					<p class="text-base-content/70 text-xs break-words sm:col-start-1 sm:row-start-2">
						{details(item)}
					</p>

					<div
						class="flex flex-wrap items-center gap-1.5 pt-1 sm:col-start-2 sm:row-span-2 sm:row-start-1 sm:self-center sm:pt-0 [&>button]:grow sm:[&>button]:grow-0"
					>
						{#if hasUnknownOutcome(item)}
							<button
								class="btn btn-sm btn-warning hover-lift"
								disabled={sendingDisabled}
								onclick={() => onreconcile(item)}
							>
								Check status
							</button>
						{:else}
							{#if item.status === 'conflict' || item.failure?.status === 404}
								<button
									class="btn btn-sm btn-soft btn-primary hover-lift"
									disabled={recoveryDisabled}
									onclick={() => onresolve(item)}
								>
									Resolve conflict
								</button>
							{/if}

							{#if item.status === 'failed'}
								<button
									class="btn btn-sm btn-soft hover-lift"
									disabled={recoveryDisabled}
									onclick={() => onretry(item)}
								>
									Try again
								</button>

								{#if item.operation !== 'delete'}
									<button
										class="btn btn-sm btn-soft hover-lift"
										disabled={recoveryDisabled}
										onclick={() => oncorrect(item)}
									>
										Edit change
									</button>
								{/if}
							{/if}

							<button
								class="btn btn-sm btn-soft btn-error hover-lift"
								aria-label={`Discard local change to ${mutationLabel(item)}`}
								disabled={recoveryDisabled}
								onclick={() => ondiscard(item)}
							>
								Discard
							</button>
						{/if}
					</div>
				</li>
			{/each}
		</ul>
	</details>
{/if}
