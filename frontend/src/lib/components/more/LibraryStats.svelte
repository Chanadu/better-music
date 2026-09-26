<script lang="ts">
	import StatCard from '$lib/components/common/StatCard.svelte';
	import RatingDistributionChart from './RatingDistributionChart.svelte';
	import AlbumIcon from '$lib/components/icons/AlbumIcon.svelte';
	import ArtistIcon from '$lib/components/icons/ArtistIcon.svelte';
	import HeadphonesIcon from '$lib/components/icons/HeadphonesIcon.svelte';
	import StarIcon from '$lib/components/icons/StarIcon.svelte';
	import { database } from '$lib/scripts/database';
	import { ratingColor } from '$lib/scripts/rating-colors';
	import type { Album } from '$lib/scripts/types';

	type TimeFilter = 'month' | 'year' | 'all';

	const filters: { value: TimeFilter; label: string }[] = [
		{ value: 'month', label: 'This month' },
		{ value: 'year', label: 'This year' },
		{ value: 'all', label: 'All time' },
	];

	let timeFilter = $state<TimeFilter>('all');

	function isInPeriod(album: Album, filter: TimeFilter) {
		if (!album.listened) return false;
		if (filter === 'all') return true;
		if (!album.listened_at) return false;

		const listenedAt = new Date(album.listened_at);
		if (Number.isNaN(listenedAt.getTime())) return false;

		const now = new Date();
		if (listenedAt.getFullYear() !== now.getFullYear()) return false;

		return filter === 'year' || listenedAt.getMonth() === now.getMonth();
	}

	let stats = $derived.by(() => {
		const albums = $database?.albums ?? [];
		const listenedAlbums = albums.filter((album) => isInPeriod(album, timeFilter));
		const ratings = listenedAlbums.flatMap((album) => (typeof album.rating === 'number' ? [album.rating] : []));
		const distribution = Array.from({ length: 10 }, (_, index) => ({
			rating: index + 1,
			count: ratings.filter((rating) => Math.round(rating) === index + 1).length,
		}));

		return {
			listened: listenedAlbums.length,
			artists: new Set(listenedAlbums.map((album) => album.artist_id)).size,
			queued: albums.filter((album) => !album.listened).length,
			averageRating:
				ratings.length > 0 ? ratings.reduce((total, rating) => total + rating, 0) / ratings.length : null,
			distribution,
		};
	});
</script>

<section aria-labelledby="library-stats-heading">
	<div class="mb-5 flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
		<div>
			<h1 id="library-stats-heading" class="text-secondary text-lg font-bold tracking-[0.14em] uppercase sm:text-xl">
				Your library
			</h1>
		</div>

		<div class="join bg-base-200 w-fit p-1" aria-label="Stats time period">
			{#each filters as filter}
				<button
					type="button"
					class="join-item btn btn-sm border-0"
					class:btn-secondary={timeFilter === filter.value}
					class:btn-ghost={timeFilter !== filter.value}
					aria-pressed={timeFilter === filter.value}
					onclick={() => (timeFilter = filter.value)}
				>
					{filter.label}
				</button>
			{/each}
		</div>
	</div>

	{#if !$database}
		<div class="grid grid-cols-2 gap-3 lg:grid-cols-4" aria-label="Loading library stats">
			{#each Array(4) as _}
				<div class="bg-base-200 flex min-h-28 items-center gap-4 rounded-xl px-5 py-4 shadow-sm">
					<div class="skeleton size-11 shrink-0 rounded-full"></div>
					<div class="min-w-0 flex-1">
						<div class="skeleton h-3 w-16"></div>
						<div class="skeleton mt-2 h-7 w-12"></div>
					</div>
				</div>
			{/each}
		</div>
	{:else}
		<div class="grid grid-cols-2 gap-3 lg:grid-cols-4">
			<StatCard value={stats.listened} label="Albums listened" icon={HeadphonesIcon} class="h-full" />
			<StatCard
				value={stats.artists}
				label="Artists explored"
				icon={ArtistIcon}
				tone="secondary"
				class="h-full"
			/>
			<StatCard value={stats.queued} label="Albums queued" icon={AlbumIcon} tone="accent" class="h-full" />
			<StatCard
				value={stats.averageRating === null ? '—' : stats.averageRating.toFixed(1)}
				label="Avg. rating"
				icon={StarIcon}
				color={ratingColor(stats.averageRating)}
				class="h-full"
			/>
		</div>

		<RatingDistributionChart distribution={stats.distribution} />
	{/if}
</section>
