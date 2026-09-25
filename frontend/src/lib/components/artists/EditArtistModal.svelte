<script lang="ts">
	import ManualArtistForm from '$lib/components/create/ManualArtistForm.svelte';
	import FormModalShell from '$lib/components/create/FormModalShell.svelte';
	import SpotifySearch from '$lib/components/create/SpotifySearch.svelte';
	import { artistsApi, spotifyApi } from '$lib/scripts/api';
	import { refreshDatabaseData } from '$lib/scripts/database';
	import type { Artist, SpotifyRow as Row } from '$lib/scripts/types';

	let {
		artist,
		dialog = $bindable(),
		onupdated,
	}: {
		artist: Artist;
		dialog?: HTMLDialogElement;
		onupdated?: (artist: Artist) => void;
	} = $props();

	let name = $state('');
	let coverUrl = $state('');
	let spotifyId = $state('');
	let error = $state('');
	let refreshMessage = $state('');
	let saving = $state(false);
	let refreshing = $state(false);
	let tab = $state<'details' | 'spotify'>('details');
	let selected = $state<Row | undefined>();
	let spotify = $state<SpotifySearch>();

	let changed = $derived(
		name.trim() !== artist.name ||
			coverUrl.trim() !== (artist.cover_url ?? '') ||
			spotifyId.trim() !== (artist.spotify_id ?? ''),
	);
	let canSave = $derived(
		!saving && !refreshing && (tab === 'spotify' ? Boolean(selected) : Boolean(name.trim()) && changed),
	);

	$effect(() => {
		const current = artist;
		name = current.name;
		coverUrl = current.cover_url ?? '';
		spotifyId = current.spotify_id ?? '';
		error = '';
		refreshMessage = '';
		tab = 'details';
		selected = undefined;
	});

	function formatError(value: unknown, fallback: string) {
		const text = value instanceof Error ? value.message : fallback;
		return text.charAt(0).toUpperCase() + text.slice(1);
	}

	function reset() {
		name = artist.name;
		coverUrl = artist.cover_url ?? '';
		spotifyId = artist.spotify_id ?? '';
		error = '';
		refreshMessage = '';
		tab = 'details';
		selected = undefined;
		spotify?.reset();
	}

	async function refreshFromSpotify() {
		if (!spotifyId.trim() || refreshing) return;

		refreshing = true;
		error = '';
		refreshMessage = '';
		try {
			const spotifyArtist = await spotifyApi.getArtist(spotifyId.trim());
			name = spotifyArtist.name;
			coverUrl = spotifyArtist.images[0]?.url ?? '';
			refreshMessage = 'Latest artist data loaded from Spotify. Save to apply it.';
		} catch (e) {
			error = formatError(e, 'Failed to refresh artist from Spotify');
		} finally {
			refreshing = false;
		}
	}

	async function save() {
		if (!canSave) return;

		saving = true;
		error = '';
		try {
			const selectedArtist = tab === 'spotify' ? selected : undefined;

			await artistsApi.update(artist.id, {
				name: selectedArtist?.name ?? name.trim(),
				cover_url: selectedArtist?.imageUrl ?? coverUrl.trim(),
				spotify_id: selectedArtist?.id ?? spotifyId.trim(),
			});

			const updated = await artistsApi.get(artist.id);

			onupdated?.(updated);

			await refreshDatabaseData();
			dialog?.close();
		} catch (e) {
			error = formatError(e, 'Failed to save artist');
		} finally {
			saving = false;
		}
	}
</script>

<FormModalShell
	bind:dialog
	title="Artist"
	headingPrefix="Edit"
	saveLabel="Save changes"
	{error}
	{saving}
	{canSave}
	onsave={save}
	onclose={reset}
>
	<div role="tablist" class="tabs tabs-border flex">
		<input
			type="radio"
			name="edit_artist_modal_tabs"
			role="tab"
			class="tab flex-1"
			aria-label="Details"
			value="details"
			bind:group={tab}
		/>
		<div role="tabpanel" class="tab-content px-2 pt-4">
			<ManualArtistForm
				bind:name
				{coverUrl}
				{spotifyId}
				showSpotifyRefresh={Boolean(artist.spotify_id)}
				{refreshing}
				onrefresh={refreshFromSpotify}
			/>

			{#if refreshMessage}
				<div class="alert alert-success alert-soft mt-4 justify-center text-center" role="status">
					<span>{refreshMessage}</span>
				</div>
			{/if}
		</div>

		<input
			type="radio"
			name="edit_artist_modal_tabs"
			role="tab"
			class="tab flex-1"
			aria-label={artist.spotify_id ? 'Change Spotify' : 'Link Spotify'}
			value="spotify"
			bind:group={tab}
		/>

		<SpotifySearch bind:this={spotify} type="artist" bind:selected />
	</div>
</FormModalShell>
