<script lang="ts">
	import RecordSyncBadge from '$lib/components/common/RecordSyncBadge.svelte';
	import { getContext, onMount } from 'svelte';
	import { previewContext, type PreviewContext } from '$lib/components/preview/context';
	const preview = getContext<PreviewContext | undefined>(previewContext);
	import { page } from '$app/state';
	import EditAlbumModal from '$lib/components/albums/EditAlbumModal.svelte';
	import MediaThumbnail from '$lib/components/common/MediaThumbnail.svelte';
	import AlbumActions from '$lib/components/albums/AlbumActions.svelte';
	import {
		database,
		albumsLibrary,
		getDatabaseData,
		refreshDatabaseData,
		parseEntityId,
	} from '$lib/scripts/database';
	import { withReturnTo } from '$lib/scripts/navigation';
	import { getCurrentUserId } from '$lib/scripts/auth';
	import type { EntityId } from '$lib/scripts/types';
	import { persistentStorage } from '$lib/scripts/storage';

	let library = $derived(preview?.library ?? $database);
	let selectedId = $state<EntityId>();
	let selectionLoaded = $state(false);
	let storageKey = '';
	let editModal = $state<EditAlbumModal>();
	let error = $state('');
	let unlistened = $derived(library?.albums.filter((album) => !album.listened) ?? []);
	let album = $derived(unlistened.find((item) => item.id === selectedId));
	let artist = $derived(library?.artists.find((item) => item.id === album?.artist_id));

	function shuffle() {
		const alternatives = unlistened.filter((item) => item.id !== selectedId);
		selectedId = alternatives[Math.floor(Math.random() * alternatives.length)]?.id;
	}

	$effect(() => {
		if (!selectionLoaded || !library) return;
		if (unlistened.length && !album) shuffle();
		else if (!unlistened.length) selectedId = undefined;
	});

	$effect(() => {
		if (preview || !selectionLoaded || !library || !storageKey) return;
		if (selectedId !== undefined) persistentStorage.set(storageKey, String(selectedId));
		else persistentStorage.remove(storageKey);
	});

	async function load() {
		error = '';
		try {
			if (!(await getDatabaseData())) await refreshDatabaseData();
		} catch {
			error = 'Could not load your albums. Please try again.';
		}
	}

	onMount(() => {
		if (preview) {
			selectedId = unlistened[0]?.id;
			selectionLoaded = true;
			return;
		}
		const userId = getCurrentUserId();
		if (userId !== null) {
			storageKey = `bettermusic:next-album:${userId}`;
			selectedId = parseEntityId(persistentStorage.get(storageKey)) ?? undefined;
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
				class="rounded-box hover-lift block"
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
					<h1 class="text-xl leading-tight font-bold wrap-break-word sm:text-2xl">
						{album.title}
						<RecordSyncBadge entity="album" id={album.id} />
					</h1>

					<p class="text-base-content/75 mt-1.5 text-base">
						{artist?.name ?? 'Unknown artist'}
						<span class="text-base-content/50">
							· {album.year ?? 'Year unknown'}
						</span>
					</p>
				</div>

				<AlbumActions
					canShuffle={unlistened.length > 1}
					canMarkListened={Boolean(artist)}
					spotifyId={album.spotify_id ?? undefined}
					onshuffle={shuffle}
					onlistened={() => editModal?.markListened()}
				/>
			</div>
		</div>
	{:else if error}

			<div role="alert" class="py-10 text-center">
				<p>
					{error}
				</p>

				<button class="btn btn-soft hover-lift mt-4" onclick={load}>
					Try again
				</button>
			</div>
	{:else if !library}

				<div class="flex min-h-64 items-center justify-center" role="status">
					<span class="loading text-base-content" aria-label="Loading your next album">
					</span>
				</div>
	{:else if !unlistened.length}

					<div class="py-12 text-center">
						<h3 class="text-xl font-semibold">
							Your queue is all caught up
						</h3>

						<p class="text-base-content/60 mt-2">
							Add an album to find your next listen.
						</p>

						<a href="/listen" class="btn btn-primary hover-lift mt-5">
							Add an album
						</a>
					</div>
				{/if}
</section>

{#if !preview && album && artist}
	<EditAlbumModal
		bind:this={editModal}
		{album}
		{artist}
		onsave={(payload) => albumsLibrary.update(album.id, payload)}
	/>
{/if}
