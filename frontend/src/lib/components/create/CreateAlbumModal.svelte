<script lang="ts">
	import ListenedFields from './ListenedFields.svelte';
	import ManualAlbumForm from './ManualAlbumForm.svelte';
	import ModalShell from './ModalShell.svelte';
	import SpotifySearch from './SpotifySearch.svelte';
	import { albumsApi, artistsApi, spotifyApi } from '$lib/scripts/api';
	import { fetchDatabaseData, refreshDatabaseData } from '$lib/scripts/database';
	import type { Artist, SpotifyRow as Row } from '$lib/scripts/types';

	let { dialog = $bindable(), onclose }: { dialog?: HTMLDialogElement; onclose?: () => void } = $props();

	let tab = $state<'manual' | 'spotify'>('manual');
	let name = $state('');
	let selected = $state<Row | undefined>();
	let error = $state('');
	let saving = $state(false);
	let spotify: SpotifySearch;
	let artists = $state<Artist[]>([]);
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
		!saving && (tab === 'spotify' ? Boolean(selected) : Boolean(name.trim()) && Boolean(artistId) && yearValid),
	);

	$effect(() => {
		fetchDatabaseData()
			.then((data) => (artists = data.artists))
			.catch((e) => (error = formatError(e, 'Failed to load artists')));
	});

	function formatError(value: unknown, fallback: string) {
		const text = value instanceof Error ? value.message : fallback;
		return text.charAt(0).toUpperCase() + text.slice(1);
	}

	function reset() {
		tab = 'manual';
		name = '';
		selected = undefined;
		error = '';
		artistId = '';
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

	async function findOrCreateArtist(row: Row) {
		const existing = artists.find(
			(artist) =>
				(row.artistId && artist.spotify_id === row.artistId) ||
				artist.name.trim().toLowerCase() === (row.artistName ?? '').trim().toLowerCase(),
		);
		if (existing) return existing;

		let cover_url: string | undefined;
		if (row.artistId && row.artistName) {
			try {
				cover_url = (await spotifyApi.searchArtists(row.artistName, 10)).find(
					(artist) => artist.id === row.artistId,
				)?.images[0]?.url;
			} catch {}
		}

		const created = await artistsApi.create({
			name: row.artistName ?? '',
			spotify_id: row.artistId,
			cover_url,
		});
		artists = [...artists, created];
		return created;
	}

	async function save() {
		if (!canSave) return;

		saving = true;
		error = '';

		try {
			const chosenArtist = tab === 'spotify' ? await findOrCreateArtist(selected!) : undefined;
			const id = chosenArtist?.id ?? Number(artistId);
			const album = await albumsApi.create({
				artist_id: id,
				title: tab === 'spotify' ? selected!.name : name.trim(),
				spotify_id: tab === 'spotify' ? selected!.id : undefined,
			});

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

		<SpotifySearch bind:this={spotify} type="album" bind:selected />
	</div>

	<ListenedFields bind:listened bind:listenedAt bind:rating />
</ModalShell>
