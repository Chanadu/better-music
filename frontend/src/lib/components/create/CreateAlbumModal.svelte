<script lang="ts">
	import ListenedFields from './ListenedFields.svelte';
	import ManualAlbumForm from './ManualAlbumForm.svelte';
	import ModalShell from './ModalShell.svelte';
	import SpotifySearch from './SpotifySearch.svelte';
	import { albumsApi, artistsApi, spotifyApi } from '$lib/scripts/api';
	import { fetchDatabaseData, refreshDatabaseData } from '$lib/scripts/database';
	import { markAlbumAsNew, markArtistAsNew } from '$lib/scripts/newly-added';
	import type { Album, Artist, SpotifyArtistCredit, SpotifyRow as Row } from '$lib/scripts/types';

	let {
		dialog = $bindable(),
		onclose,
		initialArtistId,
	}: {
		dialog?: HTMLDialogElement;
		onclose?: () => void;
		initialArtistId?: number;
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
	let currentYear = new Date().getFullYear();

	function isDigit(character: string) {
		return character >= '0' && character <= '9';
	}

	let yearValid = $derived(
		!year || (year.length === 4 && [...year].every(isDigit) && Number(year) >= 1000 && Number(year) <= currentYear),
	);
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
		fetchDatabaseData()
			.then((data) => {
				artists = data.artists;
				albums = data.albums;
			})
			.catch((e) => (error = formatError(e, 'Failed to load library')));
	});

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
			cover_url = (await spotifyApi.searchArtists(credit.name, 10)).find((artist) => artist.id === credit.id)
				?.images[0]?.url;
		} catch {}

		const created = await artistsApi.create({
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
				albums = (await fetchDatabaseData({ force: true })).albums;
				if (albums.some((album) => album.spotify_id === selected!.id)) {
					throw new Error('Album has already been added');
				}
			}

			const selectedCredit = selected?.artists?.find((artist) => artist.id === selectedSpotifyArtistId);
			if (tab === 'spotify' && !selectedCredit) throw new Error('Select an artist for this album');

			const chosenArtist = selectedCredit ? await findOrCreateArtist(selectedCredit) : undefined;
			const id = chosenArtist?.id ?? Number(artistId);
			const album = await albumsApi.create({
				artist_id: id,
				title: tab === 'spotify' ? selected!.name : name.trim(),
				spotify_id: tab === 'spotify' ? selected!.id : undefined,
			});
			markAlbumAsNew(album.id);

			const metadata = {
				artist_id: id,
				cover_url: tab === 'spotify' ? selected!.imageUrl : undefined,
				year: getAlbumYear(),
				listened: listened || undefined,
				rating: listened ? rating : undefined,
				comment: comment.trim() || undefined,
				listened_at: listened ? listenedAt || undefined : undefined,
			};

			if (Object.values(metadata).some((value, index) => index > 0 && value !== undefined)) {
				await albumsApi.update(album.id, metadata);
			}

			await refreshDatabaseData();
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

<ModalShell bind:dialog title="Album" {error} {saving} {canSave} onsave={save} {onclose}>
	<div role="tablist" class="tabs tabs-border flex">
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
			<ManualAlbumForm {artists} bind:artistId bind:name bind:year bind:comment />
		</div>

		<input
			type="radio"
			name="album_modal_tabs"
			role="tab"
			class="tab flex-1"
			aria-label="Spotify"
			value="spotify"
			bind:group={tab}
		/>

		<SpotifySearch bind:this={spotify} type="album" bind:selected bind:selectedArtistId={selectedSpotifyArtistId} />
	</div>

	<ListenedFields bind:listened bind:listenedAt bind:rating />
</ModalShell>
