<script lang="ts">
	import SpotifyIcon from '$lib/components/icons/SpotifyIcon.svelte';
	import ListenedFields from '$lib/components/create/ListenedFields.svelte';
	import ManualAlbumForm from '$lib/components/create/ManualAlbumForm.svelte';
	import FormModalShell from '$lib/components/create/FormModalShell.svelte';
	import SpotifySearch from '$lib/components/create/SpotifySearch.svelte';
	import { albumsApi, spotifyApi } from '$lib/scripts/api';
	import { refreshDatabaseData } from '$lib/scripts/database';
	import type { Album, Artist, SpotifyRow as Row } from '$lib/scripts/types';

	let {
		album,
		artist,
		dialog = $bindable(),
		onupdated,
	}: {
		album: Album;
		artist: Artist;
		dialog?: HTMLDialogElement;
		onupdated?: (album: Album) => void;
	} = $props();

	let title = $state('');
	let coverUrl = $state('');
	let spotifyId = $state('');
	let year = $state('');
	let comment = $state('');
	let listened = $state(false);
	let listenedAt = $state('');
	let rating = $state(5);
	let error = $state('');
	let refreshMessage = $state('');
	let saving = $state(false);
	let refreshing = $state(false);
	let tab = $state<'details' | 'spotify'>('details');
	let selected = $state<Row | undefined>();
	let spotify = $state<SpotifySearch>();

	function dateInputValue(value?: string) {
		if (!value) return '';
		const match = value.match(/^\d{4}-\d{2}-\d{2}/);
		return match?.[0] ?? '';
	}

	let yearValid = $state(true);

	let changed = $derived(
		title.trim() !== album.title ||
			coverUrl.trim() !== (album.cover_url ?? '') ||
			spotifyId.trim() !== (album.spotify_id ?? '') ||
			year !== (album.year?.toString() ?? '') ||
			comment.trim() !== (album.comment ?? '') ||
			listened !== album.listened ||
			(listened && rating !== (album.rating ?? 5)) ||
			(listened && listenedAt !== dateInputValue(album.listened_at)),
	);
	let canSave = $derived(
		!saving &&
			!refreshing &&
			(tab === 'spotify' ? Boolean(selected) : Boolean(title.trim()) && yearValid && changed),
	);

	function loadAlbum(current: Album) {
		title = current.title;
		coverUrl = current.cover_url ?? '';
		spotifyId = current.spotify_id ?? '';
		year = current.year?.toString() ?? '';
		comment = current.comment ?? '';
		listened = current.listened;
		listenedAt = dateInputValue(current.listened_at);
		rating = current.rating ?? 5;
	}

	$effect(() => {
		loadAlbum(album);
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
		loadAlbum(album);
		error = '';
		refreshMessage = '';
		tab = 'details';
		selected = undefined;
		spotify?.reset();
	}

	export function markListened() {
		reset();
		listened = true;
		const date = new Date();
		listenedAt = `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`;
		dialog?.showModal();
	}

	async function refreshFromSpotify() {
		if (!spotifyId.trim() || refreshing) return;

		refreshing = true;
		error = '';
		refreshMessage = '';
		try {
			const spotifyAlbum = await spotifyApi.getAlbum(spotifyId.trim());
			title = spotifyAlbum.name;
			coverUrl = spotifyAlbum.images[0]?.url ?? '';
			year = spotifyAlbum.release_date.split('-')[0];
			refreshMessage = 'Latest album data loaded from Spotify. Save to apply it.';
		} catch (e) {
			error = formatError(e, 'Failed to refresh album from Spotify');
		} finally {
			refreshing = false;
		}
	}

	async function save() {
		if (!canSave) return;

		saving = true;
		error = '';
		try {
			const selectedAlbum = tab === 'spotify' ? selected : undefined;
			const selectedYear = selectedAlbum?.releaseYear;

			await albumsApi.update(album.id, {
				artist_id: album.artist_id,
				title: selectedAlbum?.name ?? title.trim(),
				cover_url: selectedAlbum?.imageUrl ?? coverUrl.trim(),
				spotify_id: selectedAlbum?.id ?? spotifyId.trim(),
				year:
					selectedYear ? Number(selectedYear)
					: year ? Number(year)
					: undefined,
				comment: comment.trim(),
				listened,
				rating: listened ? rating : undefined,
				listened_at: listened ? listenedAt || undefined : undefined,
			});

			const updated = await albumsApi.get(album.id, album.artist_id);
			onupdated?.(updated);
			await refreshDatabaseData();
			dialog?.close();
		} catch (e) {
			error = formatError(e, 'Failed to save album');
		} finally {
			saving = false;
		}
	}
</script>

<FormModalShell
	bind:dialog
	title="Album"
	headingPrefix="Edit"
	saveLabel="Save changes"
	{error}
	{saving}
	{canSave}
	onsave={save}
	onclose={reset}
>
	<div role="tablist" class="tabs tabs-border">
		<input
			type="radio"
			name="edit_album_modal_tabs"
			role="tab"
			class="tab flex-1"
			aria-label="Details"
			value="details"
			bind:group={tab}
		/>
		<div role="tabpanel" class="tab-content px-2 pt-4">
			<ManualAlbumForm
				artists={[artist]}
				artistId={album.artist_id.toString()}
				bind:name={title}
				bind:year
				bind:yearValid
				bind:comment
				{coverUrl}
				lockArtist
			/>

			{#if album.spotify_id}
				<button
					type="button"
					class="btn btn-soft mt-4 w-full"
					disabled={refreshing}
					onclick={refreshFromSpotify}
				>
					<SpotifyIcon class="size-5" />
					{refreshing ? 'Refreshing from Spotify...' : 'Refresh from Spotify'}
				</button>
			{/if}

			{#if refreshMessage}
				<div class="alert alert-success alert-soft mt-4 justify-center text-center" role="status">
					<span>{refreshMessage}</span>
				</div>
			{/if}
		</div>

		<input
			type="radio"
			name="edit_album_modal_tabs"
			role="tab"
			class="tab flex-1"
			aria-label={album.spotify_id ? 'Change Spotify' : 'Link Spotify'}
			value="spotify"
			bind:group={tab}
		/>

		<SpotifySearch bind:this={spotify} type="album" bind:selected showArtistPicker={false} />
	</div>

	<ListenedFields bind:listened bind:listenedAt bind:rating />
</FormModalShell>
