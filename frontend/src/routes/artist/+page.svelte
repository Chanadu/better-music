<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { onMount } from 'svelte';
	import DiscographySection from '$lib/components/artists/DiscographySection.svelte';
	import EditArtistModal from '$lib/components/artists/EditArtistModal.svelte';
	import MediaHero from '$lib/components/common/MediaHero.svelte';
	import DeleteConfirmationDialog from '$lib/components/common/DeleteConfirmationDialog.svelte';
	import StatCard from '$lib/components/common/StatCard.svelte';
	import CreateAlbumModal from '$lib/components/create/CreateAlbumModal.svelte';
	import AlbumIcon from '$lib/components/icons/AlbumIcon.svelte';
	import CalendarIcon from '$lib/components/icons/CalendarIcon.svelte';
	import StarIcon from '$lib/components/icons/StarIcon.svelte';
	import SadFaceIcon from '$lib/components/icons/SadFaceIcon.svelte';
	import { ApiError, artistsApi } from '$lib/scripts/api';
	import { database, refreshDatabaseData } from '$lib/scripts/database';
	import { getReturnHref } from '$lib/scripts/navigation';
	import type { Artist } from '$lib/scripts/types';

	let artist = $state<Artist>();
	let status = $state('Loading artist...');
	let albumDialog = $state<HTMLDialogElement>();
	let deleteDialog = $state<HTMLDialogElement>();
	let editDialog = $state<HTMLDialogElement>();
	let backHref = $derived(getReturnHref(page.url, '/artists'));

	let albums = $derived(
		$database?.albums
			.filter((album) => album.artist_id === artist?.id)
			.sort((a, b) => (b.year ?? 0) - (a.year ?? 0)) ?? [],
	);
	let ratedAlbums = $derived(albums.filter((album) => typeof album.rating === 'number'));
	let averageRating = $derived(
		ratedAlbums.length ?
			ratedAlbums.reduce((total, album) => total + (album.rating ?? 0), 0) / ratedAlbums.length
		:	null,
	);
	let addedDate = $derived(
		artist ?
			new Intl.DateTimeFormat(undefined, {
				month: '2-digit',
				day: '2-digit',
				year: '2-digit',
			}).format(new Date(artist.created_at))
		:	'—',
	);
	let deleteBlockedReason = $derived(
		albums.length > 0 ?
			`Remove ${albums.length === 1 ? 'the album' : `all ${albums.length} albums`} from this artist first.`
		:	undefined,
	);

	async function deleteArtist() {
		if (!artist) return;

		await artistsApi.delete(artist.id);
		void refreshDatabaseData().catch((error) => console.error('Failed to refresh artists after deletion', error));
		await goto(backHref, { replaceState: true });
	}

	onMount(async () => {
		const raw = page.url.searchParams.get('id');
		const id = Number(raw);

		if (!raw || !Number.isInteger(id) || id < 1) {
			status = 'Artist not found.';
			return;
		}

		try {
			artist = await artistsApi.get(id);
			status = '';
		} catch (error) {
			status = error instanceof ApiError && error.status === 404 ? 'Artist not found.' : 'Could not load artist.';
		}
	});
</script>

<svelte:head>
	<title>{artist ? `${artist.name} · Better Music` : 'Artist · Better Music'}</title>
</svelte:head>

{#if status}
	<div class="flex min-h-72 items-center justify-center">
		{#if status === 'Loading artist...'}
			<span class="loading loading-spinner loading-lg text-primary" aria-label={status}></span>
		{:else}
			<div class="text-center">
				<div class="bg-base-200 mx-auto mb-4 flex size-14 items-center justify-center rounded-full">
					<SadFaceIcon class="text-base-content/50 size-6" />
				</div>

				<p class="text-base-content/60">{status}</p>
				<a class="btn btn-ghost btn-sm mt-3" href="/artists">Back to artists</a>
			</div>
		{/if}
	</div>
{:else if artist}
	<MediaHero
		title={artist.name}
		imageUrl={artist.cover_url}
		imageAlt={`${artist.name} artist portrait`}
		editLabel="Edit artist"
		deleteLabel="Delete artist"
		{backHref}
		ondelete={() => deleteDialog?.showModal()}
		onedit={() => editDialog?.showModal()}
	/>

	<EditArtistModal bind:dialog={editDialog} {artist} onupdated={(updated) => (artist = updated)} />

	<DeleteConfirmationDialog
		bind:dialog={deleteDialog}
		title={`Delete artist (${artist.name})?`}
		description={`${artist.name} will be permanently removed from your library.`}
		confirmLabel="Delete artist"
		disabledReason={deleteBlockedReason}
		onconfirm={deleteArtist}
	/>

	<section class="mt-7 grid grid-cols-2 grid-rows-2 gap-3 sm:grid-cols-3 sm:grid-rows-1" aria-label="Artist stats">
		<StatCard
			value={albums.length}
			label="Albums"
			icon={AlbumIcon}
			class="col-start-1 row-start-1 h-full sm:col-auto sm:row-auto"
		/>
		<StatCard
			value={addedDate}
			label="Added"
			icon={CalendarIcon}
			tone="secondary"
			valueClass="text-xl"
			class="col-start-1 row-start-2 h-full sm:col-auto sm:row-auto"
		/>
		<StatCard
			value={averageRating === null ? '—' : averageRating.toFixed(1)}
			label="Avg. rating"
			icon={StarIcon}
			tone="accent"
			class="col-start-2 row-span-2 row-start-1 h-full flex-col justify-center text-center sm:col-auto sm:row-span-1 sm:row-start-auto sm:flex-row sm:justify-start sm:text-left"
		/>
	</section>

	<DiscographySection {albums} onadd={() => albumDialog?.showModal()} />
	<CreateAlbumModal bind:dialog={albumDialog} initialArtistId={artist.id} />
{/if}
