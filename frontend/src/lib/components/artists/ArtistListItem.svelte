<script lang="ts">
	import { ratingColor } from '$lib/scripts/rating-colors';
	import { page } from '$app/state';
	import MediaThumbnail from '$lib/components/common/MediaThumbnail.svelte';
	import { withReturnTo } from '$lib/scripts/navigation';
	import { newlyAdded } from '$lib/scripts/newly-added';
	import type { Artist } from '$lib/scripts/types';

	interface Props {
		artist: Artist;
		albumCount: number;
		averageRating: number | null;
	}

	let { artist, albumCount, averageRating }: Props = $props();

	const dateFormatter = new Intl.DateTimeFormat(undefined, {
		month: '2-digit',
		day: '2-digit',
		year: '2-digit',
	});
</script>

<li class="group/artist-list-item indicator block w-full p-0">
	{#if $newlyAdded.artistIds.has(artist.id)}
		<span
			class="status status-success indicator-item indicator-start indicator-top top-1"
			aria-label="Newly added"
			title="Newly added"
		></span>
	{/if}

	<a
		href={withReturnTo(`/artist?id=${artist.id}`, page.url)}
		class="list-row group-hover/artist-list-item:bg-primary/10 rounded-box w-full transition duration-200 ease-out group-hover/artist-list-item:-translate-y-1 group-hover/artist-list-item:shadow-xl active:translate-y-0 active:scale-[0.98] active:shadow-sm active:duration-75 motion-reduce:transform-none"
		aria-label={`Open ${artist.name}`}
	>
		<MediaThumbnail
			variant="list"
			imageUrl={artist.cover_url ?? ''}
			label={artist.name}
			alt={`${artist.name} artist image`}
		/>

		<div class="self-center">
			<div class="flex items-center gap-2">
				{artist.name}
			</div>

			<div class="text-base-content/60 text-xs font-semibold uppercase">
				{albumCount}

				{albumCount === 1 ? 'Album' : 'Albums'}

				{#if averageRating !== null}
					• <span style:color={ratingColor(averageRating)}>Avg {averageRating.toFixed(1)}</span>
				{/if}

				• Added {dateFormatter.format(new Date(artist.created_at))}
			</div>
		</div>
	</a>
</li>
