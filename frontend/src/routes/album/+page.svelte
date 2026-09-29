<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import AlbumArtistCard from '$lib/components/albums/AlbumArtistCard.svelte';
	import EditAlbumModal from '$lib/components/albums/EditAlbumModal.svelte';
	import AlbumNotes from '$lib/components/albums/AlbumNotes.svelte';
	import AlbumStats from '$lib/components/albums/AlbumStats.svelte';
	import MediaHero from '$lib/components/common/MediaHero.svelte';
	import DeleteConfirmationDialog from '$lib/components/common/DeleteConfirmationDialog.svelte';
	import SadFaceIcon from '$lib/components/icons/SadFaceIcon.svelte';
	import { albumsApi } from '$lib/scripts/api';
	import { getReturnHref } from '$lib/scripts/navigation';
	import { database, refreshDatabaseDataAfterMutation } from '$lib/scripts/database';

	let deleteDialog = $state<HTMLDialogElement>();
	let editDialog = $state<HTMLDialogElement>();
	let editModal = $state<EditAlbumModal>();
	let backHref = $derived(getReturnHref(page.url, '/albums'));

	let id = $derived(Number(page.url.searchParams.get('id')));
	let artistId = $derived(Number(page.url.searchParams.get('artist_id')));
	let validId = $derived(Number.isInteger(id) && Number.isInteger(artistId) && id > 0 && artistId > 0);
	let album = $derived(
		validId ? $database?.albums.find((item) => item.id === id && item.artist_id === artistId) : undefined,
	);
	let artist = $derived(validId ? $database?.artists.find((item) => item.id === artistId) : undefined);
	let status = $derived(
		!validId || ($database && (!album || !artist)) ? 'Album not found.'
		: album && artist ? ''
		: 'Loading album...',
	);

	async function deleteAlbum() {
		if (!album) return;

		await albumsApi.delete(album.id, album.artist_id);
		void refreshDatabaseDataAfterMutation().catch((error) =>
			console.error('Failed to refresh albums after deletion', error),
		);
		await goto(backHref, { replaceState: true });
	}

	let listenedDate = $derived.by(() => {
		if (!album?.listened_at) return null;
		const date = new Date(album.listened_at);
		if (Number.isNaN(date.getTime())) return album.listened_at;
		return new Intl.DateTimeFormat(undefined, {
			month: 'numeric',
			day: 'numeric',
			year: 'numeric',
		}).format(date);
	});

	let addedDate = $derived.by(() => {
		if (!album?.created_at) return 'Not recorded';
		const date = new Date(album.created_at);
		if (Number.isNaN(date.getTime())) return album.created_at;
		return new Intl.DateTimeFormat(undefined, {
			month: 'numeric',
			day: 'numeric',
			year: 'numeric',
		}).format(date);
	});
</script>

<svelte:head>
	<title>{album && artist ? `${album.title} by ${artist.name} · Better Music` : 'Album · Better Music'}</title>
</svelte:head>

{#if status}
	<div class="flex min-h-72 items-center justify-center">
		{#if status === 'Loading album...'}
			<span class="loading loading-lg text-base-content" aria-label={status}></span>
		{:else}
			<div class="text-center">
				<div class="bg-base-200 mx-auto mb-4 flex size-14 items-center justify-center rounded-full">
					<SadFaceIcon class="text-base-content/50 size-6" />
				</div>
				<p class="text-base-content/60">{status}</p>
				<a class="btn btn-ghost btn-sm mt-3" href="/albums">Back to albums</a>
			</div>
		{/if}
	</div>
{:else if album && artist}
	<MediaHero
		title={album.title}
		subtitle={album.year}
		imageUrl={album.cover_url}
		imageAlt={`${album.title} album cover`}
		editLabel="Edit album"
		deleteLabel="Delete album"
		spotifyHref={album.spotify_id ?
			`https://open.spotify.com/album/${encodeURIComponent(album.spotify_id)}`
		:	undefined}
		{backHref}
		ondelete={() => deleteDialog?.showModal()}
		onedit={() => editDialog?.showModal()}
	/>

	<EditAlbumModal bind:this={editModal} bind:dialog={editDialog} {album} {artist} />

	<DeleteConfirmationDialog
		bind:dialog={deleteDialog}
		title={`Delete album (${album.title})?`}
		description={`“${album.title}” will be permanently removed from your library.`}
		confirmLabel="Delete album"
		onconfirm={deleteAlbum}
	/>

	<div class="mx-auto max-w-5xl">
		<div class="mt-7 grid gap-4 md:grid-cols-[minmax(0,1.4fr)_minmax(18rem,0.6fr)]">
			<AlbumStats {album} {addedDate} {listenedDate} onaddrating={() => editModal?.markListened()} />
			<AlbumArtistCard {artist} />
		</div>

		<AlbumNotes comment={album.comment} />
	</div>
{/if}
