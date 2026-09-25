<script lang="ts">
	import { ratingColor } from '$lib/scripts/rating-colors';
	import { page } from '$app/state';
	import { withReturnTo } from '$lib/scripts/navigation';
	import { newlyAdded } from '$lib/scripts/newly-added';
	import type { Album } from '$lib/scripts/types';
	import MediaThumbnail from '../common/MediaThumbnail.svelte';
	import StarIcon from '../icons/StarIcon.svelte';

	interface Props {
		album: Album;
		subtitle: string;
		showRating?: boolean;
		compact?: boolean;
	}

	let { album, subtitle, showRating = false, compact = false }: Props = $props();
</script>

<a
	class="rounded-box focus-visible:outline-primary hover:bg-primary/10 block min-w-0 pb-1 transition duration-200 ease-out hover:-translate-y-1 hover:scale-[1.02] hover:shadow-xl focus-visible:outline-2 focus-visible:outline-offset-4 active:translate-y-0 active:scale-[0.98] active:shadow-sm active:duration-75 motion-reduce:transform-none"
	class:self-start={compact}
	href={withReturnTo(`/album?id=${album.id}&artist_id=${album.artist_id}`, page.url)}
	aria-label={`View ${album.title}${showRating && typeof album.rating === 'number' ? `, rated ${album.rating} out of 10` : ''}`}
>
	<div class="indicator relative mb-3 block w-full">
		<MediaThumbnail
			variant="card"
			imageUrl={album.cover_url ?? ''}
			label={album.title}
			alt={`${album.title} album cover`}
		/>

		{#if showRating && typeof album.rating === 'number'}
			<div
				class="absolute right-2 bottom-2 flex items-center gap-1 rounded-lg border border-white/15 bg-black/80 px-2 py-1 text-white shadow-sm backdrop-blur-md"
				aria-hidden="true"
			>
				<span style:color={ratingColor(album.rating)}>
					<StarIcon class="size-3.5" filled />
				</span>
				<span class="text-sm leading-5 font-bold tabular-nums">{album.rating}</span>
			</div>
		{/if}
		{#if $newlyAdded.albumIds.has(album.id)}
			<span
				class="status status-success status-lg indicator-item indicator-start"
				aria-label="Newly added"
				title="Newly added"
			></span>
		{/if}
	</div>

	<h3
		class={compact ? 'mt-2 text-xs font-semibold text-wrap wrap-break-word' : 'mt-2 truncate font-semibold'}
		title={album.title}
	>
		{album.title}
	</h3>

	<p
		class={compact ?
			'text-base-content/60 text-xs font-semibold text-wrap wrap-break-word'
		:	'text-base-content/50 mt-0.5 text-sm'}
	>
		{subtitle}
	</p>
</a>
