<script lang="ts">
	import { online } from 'svelte/reactivity/window';
	import ListenedFields from './ListenedFields.svelte';
	import ManualAlbumForm from './ManualAlbumForm.svelte';
	import FormModalShell from './FormModalShell.svelte';
	import SpotifySearch from './SpotifySearch.svelte';
	import { spotifyApi } from '$lib/scripts/api';
	import { albumsLibrary, artistsLibrary, database, getDatabaseData, parseEntityId } from '$lib/scripts/database';
	import { markAlbumAsNew, markArtistAsNew } from '$lib/scripts/newly-added';
	import type { Album, Artist, EntityId, SpotifyArtistCredit, SpotifyRow as Row } from '$lib/scripts/types';

	let {
		dialog = $bindable(),
		onclose,
		initialArtistId,
	}: {
		dialog?: HTMLDialogElement;
		onclose?: () => void;
		initialArtistId?: EntityId;
	} = $props();

	let tab = $state<'manual' | 'spotify'>('manual');
	let name = $state('');
	let selected = $state<Row | undefined>();
	let selectedSpotifyArtistId = $state('');
	let error = $state('');
	let saving = $state(false);
	let spotify: SpotifySearch;
	let artists = $state<Artist[]>([]);
	let albums = $state<Album[]>([]);
	let artistId = $state('');
	let year = $state('');
	let comment = $state('');
	let listened = $state(false);
	let listenedAt = $state('');
	let rating = $state(5);

	let yearValid = $state(true);

	$effect(() => {
		if (online.current === false && tab === 'spotify') tab = 'manual';
	});

	let canSave = $derived(
		!saving &&
			(tab === 'spotify' ?
				Boolean(selected && selectedSpotifyArtistId)
			:	Boolean(name.trim()) && Boolean(artistId) && yearValid),
	);

	$effect(() => {
		if (initialArtistId && !artistId) artistId = initialArtistId.toString();
	});

	$effect(() => {
		if ($database) {
			artists = $database.artists;
			albums = $database.albums;
		} else void loadDatabaseData();
	});

	async function loadDatabaseData() {
		try {
			const data = (await getDatabaseData()) ?? { artists: [], albums: [], loadedAt: 0 };
			artists = data.artists;
			albums = data.albums;
		} catch (e) {
			error = formatError(e, 'Failed to load library');
		}
	}

	function formatError(value: unknown, fallback: string) {
		const text = value instanceof Error ? value.message : fallback;
		return text.charAt(0).toUpperCase() + text.slice(1);
	}

	function reset() {
		tab = 'manual';
		name = '';
		selected = undefined;
		selectedSpotifyArtistId = '';
		error = '';
		artistId = initialArtistId?.toString() ?? '';
		year = '';
		comment = '';
		listened = false;
		listenedAt = '';
		rating = 5;
		spotify?.reset();
	}

	function getAlbumYear() {
		if (tab === 'spotify' && selected?.releaseYear) return Number(selected.releaseYear);
		if (year) return Number(year);
		return undefined;
	}

	async function findOrCreateArtist(credit: SpotifyArtistCredit) {
		const existing = artists.find(
			(artist) =>
				artist.spotify_id === credit.id ||
				artist.name.trim().toLowerCase() === credit.name.trim().toLowerCase(),
		);
		if (existing) return existing;

		let cover_url: string | undefined;
		try {
			if (navigator.onLine)
				cover_url = (await spotifyApi.searchArtists(credit.name, 10)).find((artist) => artist.id === credit.id)
					?.images[0]?.url;
		} catch {}

		const created = await artistsLibrary.create({
			name: credit.name,
			spotify_id: credit.id,
			cover_url,
		});
		markArtistAsNew(created.id);
		artists = [...artists, created];
		return created;
	}

	async function save() {
		if (!canSave) return;

		saving = true;
		error = '';

		try {
			if (tab === 'spotify') {
				albums = (await getDatabaseData())?.albums ?? [];
				if (albums.some((album) => album.spotify_id === selected!.id)) {
					throw new Error('Album has already been added');
				}
			}

			const selectedCredit = selected?.artists?.find((artist) => artist.id === selectedSpotifyArtistId);
			if (tab === 'spotify' && !selectedCredit) throw new Error('Select an artist for this album');

			const chosenArtist = selectedCredit ? await findOrCreateArtist(selectedCredit) : undefined;
			const id = chosenArtist?.id ?? parseEntityId(artistId);
			if (id === null) throw new Error('Select an artist');
			const album = await albumsLibrary.create({
				artist_id: id,
				title: tab === 'spotify' ? selected!.name : name.trim(),
				spotify_id: tab === 'spotify' ? selected!.id : undefined,
				cover_url: tab === 'spotify' ? selected!.imageUrl : undefined,
				year: getAlbumYear(),
				listened,
				rating: listened ? rating : undefined,
				comment: comment.trim() || undefined,
				listened_at: listened ? listenedAt || undefined : undefined,
			});
			markAlbumAsNew(album.id);

			dialog?.close();
			reset();
		} catch (e) {
			console.error('Failed to save album', e);
			error = formatError(e, 'Failed to save album');
		} finally {
			saving = false;
		}
	}
</script>

<FormModalShell bind:dialog title="Album" {error} {saving} {canSave} onsave={save} {onclose}>
	<div role="tablist" class="tabs tabs-border">
		<input
			type="radio"
			name="album_modal_tabs"
			role="tab"
			class="tab flex-1"
			aria-label="Manual"
			value="manual"
			bind:group={tab}
		/>
		<div role="tabpanel" class="tab-content px-2 pt-4">
			<ManualAlbumForm {artists} bind:artistId bind:name bind:year bind:yearValid bind:comment />
		</div>

		<input
			type="radio"
			name="album_modal_tabs"
			role="tab"
			class="tab flex-1"
			aria-label="Spotify"
			disabled={online.current === false}
			value="spotify"
			bind:group={tab}
		/>

		<SpotifySearch bind:this={spotify} type="album" bind:selected bind:selectedArtistId={selectedSpotifyArtistId} />
	</div>

	<ListenedFields bind:listened bind:listenedAt bind:rating />
</FormModalShell>
