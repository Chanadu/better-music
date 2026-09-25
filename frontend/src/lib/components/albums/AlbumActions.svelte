<script lang="ts">
	import HeadphonesIcon from '../icons/HeadphonesIcon.svelte';
	import ShuffleIcon from '../icons/ShuffleIcon.svelte';
	import SpotifyIcon from '../icons/SpotifyIcon.svelte';
	import AlbumActionButton from './AlbumActionButton.svelte';

	let {
		canShuffle,
		canMarkListened,
		spotifyId,
		onshuffle,
		onlistened,
	}: {
		canShuffle: boolean;
		canMarkListened: boolean;
		spotifyId?: string;
		onshuffle: () => void;
		onlistened: () => void;
	} = $props();

	let spotifyUrl = $derived(
		spotifyId ? `https://open.spotify.com/album/${encodeURIComponent(spotifyId)}` : undefined,
	);
</script>

<div class="@container mt-4 sm:mt-5">
	<div class="grid grid-cols-[1fr_1.5fr_1fr] gap-2 sm:grid-cols-1 sm:gap-2.5">
		<AlbumActionButton color="secondary" disabled={!canShuffle} onclick={onshuffle}>
			<ShuffleIcon />
			Shuffle
		</AlbumActionButton>
		<AlbumActionButton color="success" disabled={!spotifyUrl} href={spotifyUrl}>
			<SpotifyIcon branded={false} />
			Open in Spotify
		</AlbumActionButton>
		<AlbumActionButton color="primary" disabled={!canMarkListened} onclick={onlistened}>
			<HeadphonesIcon />
			Listened
		</AlbumActionButton>
	</div>
</div>
