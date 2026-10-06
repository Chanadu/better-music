<script lang="ts">
	import { online } from 'svelte/reactivity/window';
	import ManualArtistForm from './ManualArtistForm.svelte';
	import FormModalShell from './FormModalShell.svelte';
	import SpotifySearch from './SpotifySearch.svelte';
	import { artistsLibrary } from '$lib/scripts/database';
	import { markArtistAsNew } from '$lib/scripts/newly-added';
	import type { SpotifyRow as Row } from '$lib/scripts/types';

	let { dialog = $bindable(), onclose }: { dialog?: HTMLDialogElement; onclose?: () => void } = $props();

	let tab = $state<'manual' | 'spotify'>('manual');
	let name = $state('');
	let selected = $state<Row | undefined>();
	let error = $state('');
	let saving = $state(false);
	let spotify: SpotifySearch;
	$effect(() => {
		if (online.current === false && tab === 'spotify') tab = 'manual';
	});

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
			const artist = await artistsLibrary.create(
				tab === 'manual' ?
					{ name: name.trim() }
				:	{ name: selected!.name, cover_url: selected!.imageUrl, spotify_id: selected!.id },
			);
			markArtistAsNew(artist.id);
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

<FormModalShell bind:dialog title="Artist" {error} {saving} {canSave} onsave={save} {onclose}>
	<div role="tablist" class="tabs tabs-border">
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
			disabled={online.current === false}
			value="spotify"
			bind:group={tab}
		/>

		<SpotifySearch bind:this={spotify} type="artist" bind:selected />
	</div>
</FormModalShell>
