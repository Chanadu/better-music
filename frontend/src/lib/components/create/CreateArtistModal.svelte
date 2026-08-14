<script lang="ts">
	import ManualArtistForm from './ManualArtistForm.svelte';
	import ModalShell from './ModalShell.svelte';
	import SpotifySearch from './SpotifySearch.svelte';
	import { artistsApi } from '$lib/scripts/api';
	import { refreshDatabaseData } from '$lib/scripts/database';
	import type { SpotifyRow as Row } from '$lib/scripts/types';

	let { dialog = $bindable(), onclose }: { dialog?: HTMLDialogElement; onclose?: () => void } = $props();

	let tab = $state<'manual' | 'spotify'>('manual');
	let name = $state('');
	let selected = $state<Row | undefined>();
	let error = $state('');
	let saving = $state(false);
	let spotify: SpotifySearch;
	let canSave = $derived(!saving && (tab === 'spotify' ? Boolean(selected) : Boolean(name.trim())));

	function formatError(value: unknown) {
		const text = value instanceof Error ? value.message : 'Failed to save artist';
		return text.charAt(0).toUpperCase() + text.slice(1);
	}

	function reset() {
		tab = 'manual';
		name = '';
		selected = undefined;
		error = '';
		spotify?.reset();
	}

	async function save() {
		if (!canSave) return;

		saving = true;
		error = '';

		try {
			await artistsApi.create(
				tab === 'manual' ?
					{ name: name.trim() }
				:	{ name: selected!.name, cover_url: selected!.imageUrl, spotify_id: selected!.id },
			);
			await refreshDatabaseData();
			dialog?.close();
			reset();
		} catch (e) {
			console.error('Failed to save artist', e);
			error = formatError(e);
		} finally {
			saving = false;
		}
	}
</script>

<ModalShell bind:dialog title="Artist" {error} {saving} {canSave} onsave={save} {onclose}>
	<div role="tablist" class="tabs tabs-border flex">
		<input
			type="radio"
			name="artist_modal_tabs"
			role="tab"
			class="tab flex-1"
			aria-label="Manual"
			value="manual"
			bind:group={tab}
		/>
		<div role="tabpanel" class="tab-content px-2 pt-4">
			<ManualArtistForm bind:name />
		</div>

		<input
			type="radio"
			name="artist_modal_tabs"
			role="tab"
			class="tab flex-1"
			aria-label="Spotify"
			value="spotify"
			bind:group={tab}
		/>

		<SpotifySearch bind:this={spotify} type="artist" bind:selected />
	</div>
</ModalShell>
