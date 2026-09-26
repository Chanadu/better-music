<script lang="ts">
	import { ratingColor } from '$lib/scripts/rating-colors';

	type RatingCount = { rating: number; count: number };

	let { distribution }: { distribution: RatingCount[] } = $props();
	let maximum = $derived(Math.max(...distribution.map(({ count }) => count), 1));
</script>

<div class="bg-base-200 mt-3 rounded-xl px-4 pt-5 pb-4 shadow-sm sm:px-6 sm:pt-6">
	<h2 class="text-lg font-black">Rating distribution</h2>
	<p class="text-base-content/50 mt-0.5 text-sm">Albums listened to in the selected period</p>

	<div class="mt-6 grid h-52 grid-cols-10 items-end gap-1 sm:gap-3" aria-label="Rating distribution chart">
		{#each distribution as item}
			<div class="flex h-full min-w-0 flex-col items-center justify-end gap-2">
				<span class="text-base-content/60 text-xs font-bold" aria-hidden={item.count === 0}>{item.count}</span>
				<div class="flex h-36 w-full items-end justify-center">
					<div
						class="w-full max-w-14 rounded-t-sm transition-[height] duration-300"
						class:min-h-1={item.count > 0}
						style:height={`${(item.count / maximum) * 100}%`}
						style:background-color={ratingColor(item.rating)}
						title={`${item.count} ${item.count === 1 ? 'album' : 'albums'} rated ${item.rating}`}
					></div>
				</div>
				<span class="text-base-content/60 text-xs font-bold">{item.rating}</span>
			</div>
		{/each}
	</div>
</div>
