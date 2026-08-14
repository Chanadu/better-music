<script lang="ts">
	import type { Album } from '$lib/scripts/types';
	import MediaThumbnail from '../common/MediaThumbnail.svelte';
	import StarIcon from '../icons/StarIcon.svelte';

	interface Props {
		album?: Album;
		subtitle: string;
		showRating?: boolean;
		compact?: boolean;
		add?: boolean;
		onclick?: () => void;
	}

	let { album, subtitle, showRating = false, compact = false, add = false, onclick }: Props = $props();
</script>

{#if add}
	<button class="group min-w-0 text-left" type="button" aria-label="Add album" {onclick}>
		<span
			class="btn btn-ghost border-primary/35 hover:border-primary hover:bg-primary/10 rounded-box aspect-square h-auto w-full border-2 border-dashed"
		>
			<svg
				class="text-primary size-10 transition-transform group-hover:scale-110"
				viewBox="0 0 24 24"
				fill="none"
				stroke="currentColor"
				stroke-width="1.5"
				aria-hidden="true"
			>
				<path d="M12 5v14M5 12h14"></path>
			</svg>
		</span>
		<span class="text-primary mt-2 block font-semibold">Add album</span>
		<span class="text-base-content/50 mt-0.5 block text-sm">{subtitle}</span>
	</button>
{:else if album}
	<article class="group min-w-0">
		<div class="indicator mb-3 block w-full">
			<MediaThumbnail
				variant="card"
				imageUrl={album.cover_url ?? ''}
				label={album.title}
				alt={`${album.title} album cover`}
			/>

			{#if showRating}
				{#if typeof album.rating === 'number'}
					<div
						class="indicator-item indicator-bottom indicator-center badge badge-primary mr-3 mb-1 gap-1 border-0 font-bold shadow-lg"
					>
						<StarIcon class="size-3" filled />
						{album.rating}/10
					</div>
				{/if}
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
{/if}
