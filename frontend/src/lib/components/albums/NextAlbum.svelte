<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import EditAlbumModal from '$lib/components/albums/EditAlbumModal.svelte';
	import MediaThumbnail from '$lib/components/common/MediaThumbnail.svelte';
	import AlbumActions from '$lib/components/albums/AlbumActions.svelte';
	import { database, fetchDatabaseData } from '$lib/scripts/database';
	import { withReturnTo } from '$lib/scripts/navigation';
	import { getCurrentUserId } from '$lib/scripts/auth';

	let selectedId = $state<number>();
	let selectionLoaded = $state(false);
	let storageKey = '';
	let editModal = $state<EditAlbumModal>();
	let error = $state('');
	let unlistened = $derived($database?.albums.filter((album) => !album.listened) ?? []);
	let album = $derived(unlistened.find((item) => item.id === selectedId));
	let artist = $derived($database?.artists.find((item) => item.id === album?.artist_id));

	function shuffle() {
		const alternatives = unlistened.filter((item) => item.id !== selectedId);
		selectedId = alternatives[Math.floor(Math.random() * alternatives.length)]?.id;
	}

	$effect(() => {
		if (!selectionLoaded || !$database) return;
		if (unlistened.length && !album) shuffle();
		else if (!unlistened.length) selectedId = undefined;
	});

	$effect(() => {
		if (!selectionLoaded || !$database || !storageKey) return;
		try {
			if (selectedId !== undefined) localStorage.setItem(storageKey, String(selectedId));
			else localStorage.removeItem(storageKey);
		} catch {}
	});

	async function load() {
		error = '';
		try {
			await fetchDatabaseData();
		} catch {
			error = 'Could not load your albums. Please try again.';
		}
	}

	onMount(() => {
		const userId = getCurrentUserId();
		if (userId !== null) {
			storageKey = `bettermusic:next-album:${userId}`;
			try {
				const savedId = Number(localStorage.getItem(storageKey));
				if (Number.isSafeInteger(savedId) && savedId > 0) selectedId = savedId;
			} catch {}
		}
		selectionLoaded = true;
		void load();
	});
</script>

<section aria-label="Recommended album">
	{#if album}
		<div
			class="grid items-center gap-4 sm:grid-cols-[minmax(0,0.75fr)_minmax(0,1fr)] sm:gap-6 lg:grid-cols-[minmax(0,11fr)_minmax(0,9fr)] lg:gap-10"
		>
			<a
				href={withReturnTo(`/album?id=${album.id}&artist_id=${album.artist_id}`, page.url)}
				aria-label={`View ${album.title}`}
				class="rounded-box block"
			>
				<MediaThumbnail
					variant="card"
					imageUrl={album.cover_url ?? ''}
					label={album.title}
					alt={`${album.title} album cover`}
				/>
			</a>
			<div class="min-w-0">
				<div aria-live="polite" aria-atomic="true">
					<h1 class="text-xl leading-tight font-bold wrap-break-word sm:text-2xl">{album.title}</h1>
					<p class="text-base-content/75 mt-1.5 text-base">
						{artist?.name ?? 'Unknown artist'}
						<span class="text-base-content/50"> · {album.year ?? 'Year unknown'}</span>
					</p>
				</div>
				<AlbumActions
					canShuffle={unlistened.length > 1}
					canMarkListened={Boolean(artist)}
					onshuffle={shuffle}
					onlistened={() => editModal?.markListened()}
				/>
			</div>
		</div>
	{:else if error}
		<div role="alert" class="py-10 text-center">
			<p>{error}</p>
			<button class="btn btn-soft mt-4" onclick={load}>Try again</button>
		</div>
	{:else if !$database}
		<div class="flex min-h-64 items-center justify-center" role="status">
			<span class="loading loading-spinner text-primary" aria-label="Loading your next album"></span>
		</div>
	{:else if !unlistened.length}
		<div class="py-12 text-center">
			<h3 class="text-xl font-semibold">Your queue is all caught up</h3>
			<p class="text-base-content/60 mt-2">Add an album to find your next listen.</p>
			<a href="/listen" class="btn btn-primary mt-5">Add an album</a>
		</div>
	{/if}
</section>

{#if album && artist}<EditAlbumModal bind:this={editModal} {album} {artist} />{/if}
