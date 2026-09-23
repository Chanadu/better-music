<script lang="ts">
	import { database } from '$lib/scripts/database';
	import AlbumCard from './AlbumCard.svelte';
	import BackIcon from '../icons/BackIcon.svelte';

	let { mode }: { mode: 'listened' | 'added' } = $props();
	let title = $derived(mode === 'listened' ? 'Recently listened' : 'Recently added');
	let headingId = $derived(`recently-${mode}-heading`);
	let dateField: 'listened_at' | 'created_at' = $derived(mode === 'listened' ? 'listened_at' : 'created_at');

	function timestamp(value?: string) {
		const parsed = value ? Date.parse(value) : NaN;
		return Number.isFinite(parsed) ? parsed : 0;
	}

	let recentAlbums = $derived(
		($database?.albums ?? [])
			.filter((album) => (mode === 'listened' ? album.listened : album.rating == null))
			.sort((a, b) => timestamp(b[dateField]) - timestamp(a[dateField]) || b.id - a.id)
			.slice(0, 6),
	);
	let artistNames = $derived(new Map($database?.artists.map((artist) => [artist.id, artist.name]) ?? []));
</script>

{#if $database}
	<section aria-labelledby={headingId}>
		<div class="mb-3 flex items-center justify-between gap-3 sm:mb-4">
			<h2 id={headingId} class="text-xl font-semibold">{title}</h2>
			<a
				href={mode === 'listened' ? '/albums' : '/listen'}
				class="text-base-content/60 inline-flex shrink-0 items-center gap-1 border-b border-transparent text-sm hover:border-current focus-visible:border-current"
			>
				View all
				<BackIcon class="size-4 rotate-180" />
			</a>
		</div>
		{#if recentAlbums.length}
			<div class="grid grid-cols-4 gap-3 sm:gap-4 lg:grid-cols-6 lg:gap-5">
				{#each recentAlbums as album, index (album.id)}
					<div class={index >= 4 ? 'hidden min-w-0 lg:block' : 'min-w-0'}>
						<AlbumCard
							{album}
							subtitle={artistNames.get(album.artist_id) ?? 'Unknown artist'}
							showRating={mode === 'listened'}
							compact
						/>
					</div>
				{/each}
			</div>
		{:else}
			<p class="text-base-content/60 py-4 text-sm">
				{mode === 'listened' ?
					'Your recently listened albums will appear here once you mark an album as listened.'
				:	'Your recently added unrated albums will appear here.'}
			</p>
		{/if}
	</section>
{/if}
