<script lang="ts">
	import type { RecordSyncStatus } from '$lib/scripts/types';

	let { status }: { status: RecordSyncStatus } = $props();

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
		pending: 'text-warning',
		syncing: 'text-info',
		paused: 'text-base-content/60',
		failed: 'text-error',
		conflict: 'text-warning',
		blocked: 'text-error',
	};
</script>

{#if status !== 'synced'}
	<span
		class={`${colors[status]} tooltip tooltip-top inline-grid size-6 shrink-0 place-items-center self-center align-middle leading-none`}
		role="status"
		aria-label={labels[status]}
		data-tip={labels[status]}
	>
		<svg
			class="block size-5"
			viewBox="0 0 24 24"
			fill="none"
			stroke="currentColor"
			stroke-width="2"
			stroke-linecap="round"
			stroke-linejoin="round"
			aria-hidden="true"
		>
			{#if status === 'pending'}
				<circle cx="12" cy="12" r="9" />

				<path d="M12 7v5l3 2" />
			{:else if status === 'syncing'}
				<path d="M20 7v5h-5M4 17v-5h5M6 7a7 7 0 0 1 12-1l2 3M4 15l2 3a7 7 0 0 0 12-1" />
			{:else if status === 'paused'}
				<path d="M9 5v14M15 5v14" />
			{:else if status === 'failed'}
				<path d="m12 3 10 18H2L12 3ZM12 9v4M12 17h.01" />
			{:else if status === 'conflict'}
				<path d="M12 21v-7M12 14 5 7M12 14l7-7M5 11V5h6M13 5h6v6" />
			{:else if status === 'blocked'}
				<rect x="5" y="10" width="14" height="11" rx="2" />

				<path d="M8 10V7a4 4 0 0 1 8 0v3" />
			{/if}
		</svg>
	</span>
{/if}
