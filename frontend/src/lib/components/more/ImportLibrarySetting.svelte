<script lang="ts">
	import DownloadIcon from '$lib/components/icons/DownloadIcon.svelte';
	import SettingRow from './SettingRow.svelte';
	import { importLibraryJson } from '$lib/scripts/library-import';

	let fileInput: HTMLInputElement;
	let importing = $state(false);
	let error = $state(false);
	let feedback = $state('');

	async function importFile(event: Event) {
		const input = event.currentTarget as HTMLInputElement;
		const file = input.files?.[0];
		if (!file || importing) {
			return;
		}

		importing = true;
		error = false;
		feedback = 'Importing your library...';

		try {
			if (file.size > 20 * 1024 * 1024) {
				throw new Error('Choose a JSON file smaller than 20 MB.');
			}

			const result = await importLibraryJson(await file.text());
			feedback = `Added ${result.artists} artists and ${result.albums} albums; skipped ${result.skipped} existing records. Changes are saved on this device and queued for sync.`;
		} catch (value) {
			error = true;
			feedback = 'Could not import your library. Please try again.';

			if (value instanceof Error) {
				feedback = value.message;
			}
		} finally {
			importing = false;
			input.value = '';
		}
	}
</script>

<SettingRow
	title="Import library"
	description="Add artists, albums, ratings, notes, and listening dates from JSON."
	icon={DownloadIcon}
	tone="accent"
>
	<input
		bind:this={fileInput}
		type="file"
		accept=".json,application/json"
		class="hidden"
		aria-label="Choose library JSON file"
		onchange={importFile}
		disabled={importing}
	/>
	<button
		type="button"
		class="btn hover-lift btn-accent btn-md shrink-0 rounded-full shadow-sm"
		disabled={importing}
		onclick={() => fileInput.click()}
	>
		{#if importing}<span class="loading loading-sm" aria-hidden="true"></span>{/if}
		{#if importing}Importing…{:else}Import JSON{/if}
	</button>
</SettingRow>
{#if feedback}
	<p class="text-base-content/55 px-5 pb-4 text-sm" class:text-error={error} role="status" aria-live="polite">
		{feedback}
	</p>
{/if}
