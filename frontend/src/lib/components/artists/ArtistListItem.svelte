<script lang="ts">
	import { page } from '$app/state';
	import MediaThumbnail from '$lib/components/common/MediaThumbnail.svelte';
	import { withReturnTo } from '$lib/scripts/navigation';
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

<li class="p-0">
	<a
		href={withReturnTo(`/artist?id=${artist.id}`, page.url)}
		class="list-row hover:bg-base-300 rounded-box transition-colors"
		aria-label={`Open ${artist.name}`}
	>
		<MediaThumbnail
			variant="list"
			imageUrl={artist.cover_url ?? ''}
			label={artist.name}
			alt={`${artist.name} artist image`}
		/>

		<div>
			<div>{artist.name}</div>

			<div class="text-base-content/60 text-xs font-semibold uppercase">
				{albumCount}

				{albumCount === 1 ? 'Album' : 'Albums'}

				{#if averageRating !== null}
					• Avg {averageRating.toFixed(1)}
				{/if}

				• Added {dateFormatter.format(new Date(artist.created_at))}
			</div>
		</div>
	</a>
</li>
