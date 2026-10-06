<script lang="ts">
	import RecordSyncBadge from '$lib/components/common/RecordSyncBadge.svelte';
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
	class="rounded-box focus-visible:outline-base-content hover:bg-base-200 hover-lift block min-w-0 pb-1 focus-visible:outline-2 focus-visible:outline-offset-4"
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
				class="bg-base-200/95 absolute right-2 bottom-2 flex items-center gap-1 rounded-lg border border-white/15 px-2 py-1 text-white shadow-sm backdrop-blur-md"
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
		class={compact ?
			'mt-2 flex items-center gap-2 text-xs font-semibold'
		:	'mt-2 flex items-center gap-2 font-semibold'}
		title={album.title}
	>
		<span class={compact ? 'min-w-0 text-wrap wrap-break-word' : 'min-w-0 truncate'}>{album.title}</span>
		<RecordSyncBadge entity="album" id={album.id} variant="dot" />
	</h3>

	<p
		class={compact ?
			'text-base-content/60 text-xs font-semibold text-wrap wrap-break-word'
		:	'text-base-content/50 mt-0.5 text-sm'}
	>
		{subtitle}
	</p>
</a>
