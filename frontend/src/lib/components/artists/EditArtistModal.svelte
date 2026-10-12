<script lang="ts">
	import { getContext } from 'svelte';
	import { previewContext, type PreviewContext } from '$lib/components/preview/context';
	const preview = getContext<PreviewContext | undefined>(previewContext);
	const instanceId = $props.id();
	import { online } from 'svelte/reactivity/window';
	import ManualArtistForm from '$lib/components/create/ManualArtistForm.svelte';
	import FormModalShell from '$lib/components/create/FormModalShell.svelte';
	import SpotifySearch from '$lib/components/create/SpotifySearch.svelte';
	import { spotifyApi } from '$lib/scripts/api';
	import type { Artist, SpotifyRow as Row, UpdateArtistRequest } from '$lib/scripts/types';

	let {
		artist,
		dialog = $bindable(),
		onupdated,
		onsave,
		mode = 'edit',
		recoveryMessages = [],
	}: {
		artist: Artist;
		dialog?: HTMLDialogElement;
		onupdated?: (artist: Artist) => void;
		onsave: (payload: UpdateArtistRequest) => Promise<Artist>;
		mode?: 'edit' | 'correct';
		recoveryMessages?: string[];
	} = $props();

	let name = $state('');
	let coverUrl = $state('');
	let spotifyId = $state('');
	let error = $state('');
	let refreshed = $state(false);
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
	$effect(() => {
		if ((preview?.online ?? online.current) === false && tab === 'spotify') tab = 'details';
	});

	let canSave = $derived(
		!saving && !refreshing && (tab === 'spotify' ? Boolean(selected) : Boolean(name.trim()) && changed),
	);

	$effect(() => {
		const current = artist;
		if (!current) return;
		name = current.name;
		coverUrl = current.cover_url ?? '';
		spotifyId = current.spotify_id ?? '';
		error = '';
		refreshed = false;
		tab = 'details';
		selected = undefined;
	});

	function formatError(value: unknown, fallback: string) {
		const text = value instanceof Error ? value.message : fallback;
		return text.charAt(0).toUpperCase() + text.slice(1);
	}

	function reset() {
		if (artist) {
			name = artist.name;
			coverUrl = artist.cover_url ?? '';
			spotifyId = artist.spotify_id ?? '';
		}
		error = '';
		refreshed = false;
		tab = 'details';
		selected = undefined;
		spotify?.reset();
	}

	async function refreshFromSpotify() {
		if (!navigator.onLine || !spotifyId.trim() || refreshing) return;

		refreshing = true;
		error = '';
		refreshed = false;
		try {
			const spotifyArtist = await spotifyApi.getArtist(spotifyId.trim());
			name = spotifyArtist.name;
			coverUrl = spotifyArtist.images[0]?.url ?? '';
			refreshed = true;
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

			const payload: UpdateArtistRequest = {
				name: selectedArtist?.name ?? name.trim(),
				cover_url: selectedArtist?.imageUrl ?? (coverUrl.trim() || null),
				spotify_id: selectedArtist?.id ?? (spotifyId.trim() || null),
			};
			const updated = await onsave(payload);

			dialog?.close();
			onupdated?.(updated);
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
	headingPrefix={mode === 'correct' ? 'Correct' : 'Edit'}
	saveLabel={mode === 'correct' ? 'Save correction' : 'Save changes'}
	{error}
	{saving}
	{canSave}
	onsave={save}
	onclose={reset}
>
	{#if recoveryMessages.length}
		<div class="alert alert-warning alert-soft my-4">
			<ul class="list-disc pl-4">
				{#each recoveryMessages as message}
					<li>
						{message}
					</li>
				{/each}
			</ul>
		</div>
	{/if}

	<div role="tablist" class="tabs tabs-border">
		<input
			type="radio"
			name={preview ? `edit_artist_modal_tabs-${instanceId}` : 'edit_artist_modal_tabs'}
			role="tab"
			class="tab hover-lift flex-1"
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
				{refreshed}
				onrefresh={refreshFromSpotify}
			/>
		</div>

		<input
			type="radio"
			name={preview ? `edit_artist_modal_tabs-${instanceId}` : 'edit_artist_modal_tabs'}
			role="tab"
			class="tab hover-lift flex-1"
			aria-label={artist.spotify_id ? 'Change Spotify' : 'Link Spotify'}
			disabled={(preview?.online ?? online.current) === false}
			value="spotify"
			bind:group={tab}
		/>

		<SpotifySearch bind:this={spotify} type="artist" bind:selected />
	</div>
</FormModalShell>
