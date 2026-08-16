<script lang="ts">
	import type { Artist } from '$lib/scripts/types';

	let { artist }: { artist: Artist } = $props();
	let imageFailed = $state(false);

	$effect(() => {
		artist.cover_url;
		imageFailed = false;
	});
</script>

<section class="order-1 h-full md:col-start-1 md:row-start-1">
	<a
		class="bg-base-200 focus-visible:outline-primary group relative flex h-full min-h-72 items-center justify-center gap-4 overflow-hidden rounded-2xl p-5 shadow-lg transition duration-200 ease-out hover:-translate-y-1 hover:shadow-2xl focus-visible:outline-2 focus-visible:outline-offset-4 active:translate-y-0 active:scale-[0.99] sm:gap-8 sm:p-8"
		href={`/artist?id=${artist.id}`}
		aria-label={`View ${artist.name}`}
	>
		<div class="relative w-1/2 shrink-0">
			{#if artist.cover_url && !imageFailed}
				<img
					class="bg-base-300 aspect-square w-full rounded-full object-cover shadow-2xl ring-4 ring-white/10 transition-transform duration-300 group-hover:scale-105"
					src={artist.cover_url}
					alt={`${artist.name} artist portrait`}
					onerror={() => (imageFailed = true)}
				/>
			{:else}
				<div
					class="from-primary/45 via-accent/30 to-secondary/35 text-base-content/60 bg-base-100 flex aspect-square w-full items-center justify-center rounded-full text-[clamp(3rem,10vw,6rem)] font-black shadow-2xl ring-4 ring-white/10"
					role="img"
					aria-label={`${artist.name} artist portrait`}
				>
					{artist.name.charAt(0).toUpperCase()}
				</div>
			{/if}
		</div>

		<div class="relative min-w-0 flex-1 text-left">
			<p class="text-primary text-xs font-black tracking-[0.2em] uppercase">Artist</p>
			<h2
				class="group-hover:text-primary mt-1 text-2xl font-black tracking-tight text-white transition-colors sm:text-3xl"
			>
				{artist.name}
			</h2>
		</div>
	</a>
</section>
