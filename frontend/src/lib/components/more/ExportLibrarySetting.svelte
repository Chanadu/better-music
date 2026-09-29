<script lang="ts">
	import DownloadIcon from '$lib/components/icons/DownloadIcon.svelte';
	import { ApiError } from '$lib/scripts/api';
	import { getCurrentUserId, invalidateSession } from '$lib/scripts/auth';
	import { downloadLibraryExport } from '$lib/scripts/library-export';

	type ExportState = 'idle' | 'exporting' | 'success' | 'error';

	let exportState = $state<ExportState>('idle');
	let feedback = $state('Download your library, ratings, notes, and listening dates.');

	async function exportLibrary() {
		if (exportState === 'exporting') return;

		exportState = 'exporting';
		feedback = 'Preparing a fresh copy of your library…';

		try {
			const filename = await downloadLibraryExport();
			exportState = 'success';
			feedback = `Downloaded ${filename}`;
		} catch (value) {
			if ((value instanceof ApiError && value.status === 401) || getCurrentUserId() === null) {
				invalidateSession();
				location.assign('/login');
				return;
			}

			console.error('Failed to export library', value);
			exportState = 'error';
			feedback = value instanceof ApiError ? value.message : 'Could not export your library. Please try again.';
		}
	}
</script>

<div class="flex items-center gap-4 p-4 sm:p-5">
	<div class="bg-accent text-accent-content flex size-11 shrink-0 items-center justify-center rounded-full">
		<DownloadIcon class="size-5" />
	</div>
	<div class="min-w-0 flex-1">
		<h3 class="font-bold">Export library</h3>
		<p
			class="text-base-content/55 text-sm"
			class:text-error={exportState === 'error'}
			role={exportState === 'error' ? 'alert' : undefined}
			aria-live="polite"
		>
			{feedback}
		</p>
	</div>
	<button
		type="button"
		class="btn btn-accent btn-md shrink-0 gap-2 rounded-full shadow-sm"
		disabled={exportState === 'exporting'}
		onclick={exportLibrary}
	>
		{#if exportState === 'exporting'}
			<span class="loading loading-sm" aria-hidden="true"></span>
		{:else}
			<DownloadIcon class="size-4" />
		{/if}
		{exportState === 'exporting' ? 'Preparing…' : 'Export JSON'}
	</button>
</div>
