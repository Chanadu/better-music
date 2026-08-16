<script lang="ts">
	import ManualArtistForm from '$lib/components/create/ManualArtistForm.svelte';
	import ModalShell from '$lib/components/create/ModalShell.svelte';
	import { artistsApi, spotifyApi } from '$lib/scripts/api';
	import { refreshDatabaseData } from '$lib/scripts/database';
	import type { Artist } from '$lib/scripts/types';

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

	let changed = $derived(
		name.trim() !== artist.name ||
			coverUrl.trim() !== (artist.cover_url ?? '') ||
			spotifyId.trim() !== (artist.spotify_id ?? ''),
	);
	let canSave = $derived(!saving && !refreshing && Boolean(name.trim()) && changed);

	$effect(() => {
		const current = artist;
		name = current.name;
		coverUrl = current.cover_url ?? '';
		spotifyId = current.spotify_id ?? '';
		error = '';
		refreshMessage = '';
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
			await artistsApi.update(artist.id, {
				name: name.trim(),
				cover_url: coverUrl.trim(),
				spotify_id: spotifyId.trim(),
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

<ModalShell
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
	<div class="px-2 pt-2">
		<ManualArtistForm
			bind:name
			{coverUrl}
			{spotifyId}
			showSpotifyRefresh
			{refreshing}
			onrefresh={refreshFromSpotify}
		/>

		{#if refreshMessage}
			<div class="alert alert-success alert-soft mt-4 justify-center text-center" role="status">
				<span>{refreshMessage}</span>
			</div>
		{/if}
	</div>
</ModalShell>
