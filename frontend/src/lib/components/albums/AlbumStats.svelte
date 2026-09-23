<script lang="ts">
	import { ratingColor } from '$lib/scripts/rating-colors';
	import StatCard from '$lib/components/common/StatCard.svelte';
	import CalendarIcon from '$lib/components/icons/CalendarIcon.svelte';
	import PlusIcon from '$lib/components/icons/PlusIcon.svelte';
	import StarIcon from '$lib/components/icons/StarIcon.svelte';
	import type { Album } from '$lib/scripts/types';

	let {
		album,
		addedDate,
		listenedDate,
		onaddrating,
	}: {
		album: Album;
		addedDate: string;
		listenedDate: string | null;
		onaddrating?: () => void;
	} = $props();
</script>

<section
	class="order-2 grid grid-cols-2 grid-rows-2 gap-3 md:col-start-2 md:row-start-1 md:grid-cols-1 md:grid-rows-3"
	aria-label="Album details"
>
	<div
		class:row-span-2={album.listened}
		class={(album.listened ? 'col-start-2' : 'col-start-1') +
			'row-start-1 md:col-start-auto md:row-span-1 md:row-start-auto'}
	>
		<StatCard
			value={album.listened && typeof album.rating === 'number' ? `${album.rating} / 10` : 'N/A'}
			label="Your rating"
			valueColor={ratingColor(album.listened ? album.rating : undefined)}
			icon={StarIcon}
			class={album.listened ?
				'h-full flex-col justify-center text-center md:flex-row md:justify-start md:text-left'
			:	'h-full'}
		/>
	</div>

	<div
		class={album.listened ?
			'col-start-1 row-start-1 md:col-start-auto md:row-start-auto'
		:	'col-start-1 row-start-2 md:col-start-auto md:row-start-auto'}
	>
		<StatCard
			value={addedDate}
			label="Added"
			icon={CalendarIcon}
			tone="accent"
			valueClass="text-lg"
			class="h-full"
		/>
	</div>

	<div
		class={album.listened ?
			'col-start-1 row-start-2 md:col-start-auto md:row-start-auto'
		:	'col-start-2 row-span-2 row-start-1 md:col-start-auto md:row-span-1 md:row-start-auto'}
	>
		{#if album.listened}
			<StatCard
				value={listenedDate ?? 'Not recorded'}
				label="Listened on"
				icon={CalendarIcon}
				tone="secondary"
				valueClass="text-lg"
				class="h-full"
			/>
		{:else}
			<button
				class="btn btn-ghost border-primary/35 hover:bg-primary/10 h-full min-h-28 w-full flex-col gap-2 rounded-xl border-2 border-dashed transition duration-200 ease-out hover:-translate-y-1 hover:scale-[1.02] hover:shadow-xl active:translate-y-0 active:scale-[0.98] active:shadow-sm active:duration-75 motion-reduce:transform-none md:col-span-1"
				type="button"
				aria-label="Add rating"
				onclick={onaddrating}
			>
				<PlusIcon class="text-primary size-7" />
				<span class="text-primary font-semibold">Add rating</span>
			</button>
		{/if}
	</div>
</section>
