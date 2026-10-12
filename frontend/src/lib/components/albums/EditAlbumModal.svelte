<script lang="ts">
	import { getContext } from 'svelte';
	import { previewContext, type PreviewContext } from '$lib/components/preview/context';
	const preview = getContext<PreviewContext | undefined>(previewContext);
	const instanceId = $props.id();
	import { online } from 'svelte/reactivity/window';
	import SpotifyIcon from '$lib/components/icons/SpotifyIcon.svelte';
	import ListenedFields from '$lib/components/create/ListenedFields.svelte';
	import ManualAlbumForm from '$lib/components/create/ManualAlbumForm.svelte';
	import FormModalShell from '$lib/components/create/FormModalShell.svelte';
	import SpotifySearch from '$lib/components/create/SpotifySearch.svelte';
	import { spotifyApi } from '$lib/scripts/api';
	import type { Album, Artist, SpotifyRow as Row, UpdateAlbumMutationPayload } from '$lib/scripts/types';

	let {
		album,
		artist,
		dialog = $bindable(),
		onupdated,
		onsave,
		mode = 'edit',
		recoveryMessages = [],
	}: {
		album: Album;
		artist: Artist;
		dialog?: HTMLDialogElement;
		onupdated?: (album: Album) => void;
		onsave: (payload: UpdateAlbumMutationPayload) => Promise<Album>;
		mode?: 'edit' | 'correct';
		recoveryMessages?: string[];
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
	let refreshed = $state(false);
	let saving = $state(false);
	let refreshing = $state(false);
	let tab = $state<'details' | 'spotify'>('details');
	let selected = $state<Row | undefined>();
	let spotify = $state<SpotifySearch>();

	function dateInputValue(value?: string | null) {
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
	$effect(() => {
		if ((preview?.online ?? online.current) === false && tab === 'spotify') tab = 'details';
	});

	let canSave = $derived(
		!saving &&
			!refreshing &&
			(tab === 'spotify' ? Boolean(selected) : Boolean(title.trim()) && yearValid && changed),
	);

	function loadAlbum(current: Album | null | undefined) {
		if (!current) return;
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
		refreshed = false;
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
		refreshed = false;
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
		if (!navigator.onLine || !spotifyId.trim() || refreshing) return;

		refreshing = true;
		error = '';
		refreshed = false;
		try {
			const spotifyAlbum = await spotifyApi.getAlbum(spotifyId.trim());
			title = spotifyAlbum.name;
			coverUrl = spotifyAlbum.images[0]?.url ?? '';
			year = spotifyAlbum.release_date.split('-')[0];
			refreshed = true;
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

			const payload: UpdateAlbumMutationPayload = {
				artist_id: album.artist_id,
				title: selectedAlbum?.name ?? title.trim(),
				cover_url: selectedAlbum?.imageUrl ?? (coverUrl.trim() || null),
				spotify_id: selectedAlbum?.id ?? (spotifyId.trim() || null),
				year:
					selectedYear ? Number(selectedYear)
					: year ? Number(year)
					: null,
				comment: comment.trim() || null,
				listened,
				rating: listened ? rating : undefined,
				listened_at: listened ? listenedAt || null : undefined,
			};

			const updated = await onsave(payload);

			dialog?.close();
			onupdated?.(updated);
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
			name={preview ? `edit_album_modal_tabs-${instanceId}` : 'edit_album_modal_tabs'}
			role="tab"
			class="tab hover-lift flex-1"
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
					class="btn btn-soft hover-lift mt-4 w-full"
					disabled={refreshing || (preview?.online ?? online.current) === false}
					onclick={refreshFromSpotify}
				>
					<SpotifyIcon class="size-5" />

					<span role={refreshed ? 'status' : undefined}
					>
						{refreshing ? 'Refreshing from Spotify...'
						: refreshed ? 'Updated · Save to apply'
						: 'Refresh from Spotify'}
					</span>
				</button>
			{/if}
		</div>

		<input
			type="radio"
			name={preview ? `edit_album_modal_tabs-${instanceId}` : 'edit_album_modal_tabs'}
			role="tab"
			class="tab hover-lift flex-1"
			aria-label={album.spotify_id ? 'Change Spotify' : 'Link Spotify'}
			disabled={(preview?.online ?? online.current) === false}
			value="spotify"
			bind:group={tab}
		/>

		<SpotifySearch bind:this={spotify} type="album" bind:selected showArtistPicker={false} />
	</div>

	<ListenedFields bind:listened bind:listenedAt bind:rating />
</FormModalShell>
