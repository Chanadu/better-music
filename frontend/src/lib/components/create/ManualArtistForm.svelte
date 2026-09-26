<script lang="ts">
	import FloatingField from '../common/FloatingField.svelte';
	import MediaThumbnail from '../common/MediaThumbnail.svelte';
	import SpotifyIcon from '../icons/SpotifyIcon.svelte';

	let {
		name = $bindable(''),
		coverUrl = '',
		spotifyId = '',
		showSpotifyRefresh = false,
		refreshing = false,
		onrefresh,
	}: {
		name?: string;
		coverUrl?: string;
		spotifyId?: string;
		showSpotifyRefresh?: boolean;
		refreshing?: boolean;
		onrefresh?: () => void;
	} = $props();
</script>

<div class="flex flex-col gap-3">
	<FloatingField label="Artist Name">
		<input type="text" class="input w-full" placeholder="e.g. J. Cole" bind:value={name} />
	</FloatingField>

	<div class={showSpotifyRefresh ? 'grid grid-cols-2 gap-4' : 'flex justify-center'}>
		<div class={showSpotifyRefresh ? '' : 'w-1/2'}>
			<MediaThumbnail
				variant="artist-preview"
				imageUrl={coverUrl}
				label={name}
				alt={name ? `${name} artist preview` : ''}
				emptyFallback=""
			/>
		</div>

		{#if showSpotifyRefresh}
			<button
				type="button"
				class="btn btn-soft aspect-square h-auto w-full flex-col gap-3 whitespace-normal"
				disabled={refreshing || !spotifyId}
				aria-label={refreshing ? 'Refreshing artist data from Spotify' : 'Refresh artist data from Spotify'}
				title={spotifyId ? 'Refresh artist data from Spotify' : 'This artist is not linked to Spotify'}
				onclick={onrefresh}
			>
				{#if refreshing}
					<span class="loading loading-xs" aria-hidden="true"></span>
				{:else}
					<SpotifyIcon class="size-5 text-[#1ed760]" />
				{/if}
				{refreshing ? 'Refreshing from Spotify...' : 'Refresh from Spotify'}
			</button>
		{/if}
	</div>
</div>
