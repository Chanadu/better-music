<script lang="ts">
	import { page } from '$app/state';
	import { onMount } from 'svelte';
	import ArtistHero from '$lib/components/artists/ArtistHero.svelte';
	import ArtistStat from '$lib/components/artists/ArtistStat.svelte';
	import DiscographySection from '$lib/components/artists/DiscographySection.svelte';
	import AlbumIcon from '$lib/components/icons/AlbumIcon.svelte';
	import CalendarIcon from '$lib/components/icons/CalendarIcon.svelte';
	import StarIcon from '$lib/components/icons/StarIcon.svelte';
	import SadFaceIcon from '$lib/components/icons/SadFaceIcon.svelte';
	import { ApiError, artistsApi } from '$lib/scripts/api';
	import { database } from '$lib/scripts/database';
	import type { Artist } from '$lib/scripts/types';

	let artist = $state<Artist>();
	let status = $state('Loading artist...');

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
	<ArtistHero {artist} />

	<section class="stats bg-base-200 mt-4 grid grid-cols-3 shadow-sm" aria-label="Artist stats">
		<ArtistStat value={albums.length} label="Albums" icon={AlbumIcon} />
		<ArtistStat value={addedDate} label="Added" icon={CalendarIcon} />
		<ArtistStat
			value={averageRating === null ? '—' : averageRating.toFixed(1)}
			label="Avg. rating"
			icon={StarIcon}
		/>
	</section>

	<DiscographySection {albums} />
{/if}
