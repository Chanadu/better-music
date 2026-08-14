<script lang="ts">
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

<article class="group min-w-0">
	<div class="indicator mb-3 block w-full">
		<MediaThumbnail
			variant="card"
			imageUrl={album.cover_url ?? ''}
			label={album.title}
			alt={`${album.title} album cover`}
		/>

		{#if showRating && typeof album.rating === 'number'}
			<div
				class="indicator-item indicator-bottom indicator-center badge badge-primary mr-3 mb-1 gap-1 border-0 font-bold shadow-lg"
			>
				<StarIcon class="size-3" filled />
				{album.rating}/10
			</div>
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
</article>
