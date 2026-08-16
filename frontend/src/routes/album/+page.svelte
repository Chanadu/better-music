<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { onMount } from 'svelte';
	import AlbumArtistCard from '$lib/components/albums/AlbumArtistCard.svelte';
	import AlbumNotes from '$lib/components/albums/AlbumNotes.svelte';
	import AlbumStats from '$lib/components/albums/AlbumStats.svelte';
	import MediaHero from '$lib/components/common/MediaHero.svelte';
	import DeleteConfirmationDialog from '$lib/components/common/DeleteConfirmationDialog.svelte';
	import SadFaceIcon from '$lib/components/icons/SadFaceIcon.svelte';
	import { albumsApi, ApiError, artistsApi } from '$lib/scripts/api';
	import { getReturnHref } from '$lib/scripts/navigation';
	import { refreshDatabaseData } from '$lib/scripts/database';
	import type { Album, Artist } from '$lib/scripts/types';

	let album = $state<Album>();
	let artist = $state<Artist>();
	let status = $state('Loading album...');
	let deleteDialog = $state<HTMLDialogElement>();
	let backHref = $derived(getReturnHref(page.url, '/albums'));

	async function deleteAlbum() {
		if (!album) return;

		await albumsApi.delete(album.id, album.artist_id);
		void refreshDatabaseData().catch((error) => console.error('Failed to refresh albums after deletion', error));
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

	onMount(async () => {
		const rawId = page.url.searchParams.get('id');
		const rawArtistId = page.url.searchParams.get('artist_id');
		const id = Number(rawId);
		const artistId = Number(rawArtistId);

		if (!rawId || !rawArtistId || !Number.isInteger(id) || !Number.isInteger(artistId) || id < 1 || artistId < 1) {
			status = 'Album not found.';
			return;
		}

		try {
			[album, artist] = await Promise.all([albumsApi.get(id, artistId), artistsApi.get(artistId)]);
			status = '';
		} catch (error) {
			status = error instanceof ApiError && error.status === 404 ? 'Album not found.' : 'Could not load album.';
		}
	});
</script>

<svelte:head>
	<title>{album && artist ? `${album.title} by ${artist.name} · Better Music` : 'Album · Better Music'}</title>
</svelte:head>

{#if status}
	<div class="flex min-h-72 items-center justify-center">
		{#if status === 'Loading album...'}
			<span class="loading loading-spinner loading-lg text-primary" aria-label={status}></span>
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
		{backHref}
		ondelete={() => deleteDialog?.showModal()}
	/>

	<DeleteConfirmationDialog
		bind:dialog={deleteDialog}
		title={`Delete album (${album.title})?`}
		description={`“${album.title}” will be permanently removed from your library.`}
		confirmLabel="Delete album"
		onconfirm={deleteAlbum}
	/>

	<div class="mx-auto max-w-5xl">
		<div class="mt-7 grid gap-4 md:grid-cols-[minmax(0,1.4fr)_minmax(18rem,0.6fr)]">
			<AlbumStats {album} {addedDate} {listenedDate} />
			<AlbumArtistCard {artist} />
		</div>

		<AlbumNotes comment={album.comment} />
	</div>
{/if}
