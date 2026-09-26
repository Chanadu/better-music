<script lang="ts">
	import StatCard from '$lib/components/common/StatCard.svelte';
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

	let timeFilter = $state<TimeFilter>('month');

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
			maxDistribution: Math.max(...distribution.map(({ count }) => count), 1),
		};
	});
</script>

<section aria-labelledby="library-stats-heading">
	<div class="mb-5 flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
		<div>
			<p class="text-secondary text-xs font-bold tracking-[0.18em] uppercase">Your library</p>
			<h1 id="library-stats-heading" class="mt-1 text-2xl font-black sm:text-3xl">Stats</h1>
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

		<div class="bg-base-200 mt-3 rounded-xl px-4 pt-5 pb-4 shadow-sm sm:px-6 sm:pt-6">
			<h2 class="text-lg font-black">Rating distribution</h2>
			<p class="text-base-content/50 mt-0.5 text-sm">Albums listened to in the selected period</p>

			<div class="mt-6 grid h-52 grid-cols-10 items-end gap-1 sm:gap-3" aria-label="Rating distribution chart">
				{#each stats.distribution as item}
					<div class="flex h-full min-w-0 flex-col items-center justify-end gap-2">
						<span class="text-base-content/60 text-xs font-bold" aria-hidden={item.count === 0}
							>{item.count}</span
						>
						<div class="flex h-36 w-full items-end justify-center">
							<div
								class="w-full max-w-14 rounded-t-sm transition-[height] duration-300"
								class:min-h-1={item.count > 0}
								style:height={`${(item.count / stats.maxDistribution) * 100}%`}
								style:background-color={ratingColor(item.rating)}
								title={`${item.count} ${item.count === 1 ? 'album' : 'albums'} rated ${item.rating}`}
							></div>
						</div>
						<span class="text-base-content/60 text-xs font-bold">{item.rating}</span>
					</div>
				{/each}
			</div>
		</div>
	{/if}
</section>
