<script lang="ts">
	import type { Artist } from '$lib/scripts/types';

	let { artist, onedit }: { artist: Artist; onedit?: () => void } = $props();
	let imageFailed = $state(false);

	$effect(() => {
		artist.cover_url;
		imageFailed = false;
	});
</script>

<section class="relative -mx-4 -mt-2 h-[60dvh] overflow-hidden">
	{#if artist.cover_url && !imageFailed}
		<img
			class="absolute inset-0 size-full object-cover"
			src={artist.cover_url}
			alt={`${artist.name} artist portrait`}
			onerror={() => (imageFailed = true)}
		/>
	{:else}
		<div
			class="from-primary/45 via-accent/20 to-base-300 absolute inset-0 flex items-center justify-center bg-linear-to-br"
		>
			<span class="text-base-content/20 text-[10rem] font-black sm:text-[15rem]" aria-hidden="true">
				{artist.name.charAt(0).toUpperCase()}
			</span>
		</div>
	{/if}

	<div class="from-base-100 absolute inset-0 bg-linear-to-t via-black/15 to-black/5"></div>
	<div class="from-base-100 absolute inset-x-0 bottom-0 h-24 bg-linear-to-t to-transparent"></div>

	<div class="absolute inset-x-0 bottom-5 px-4 sm:bottom-7 sm:px-8">
		<h1
			class="pr-16 text-4xl leading-none font-black tracking-tight wrap-break-word text-white sm:text-6xl lg:text-7xl"
		>
			{artist.name}
		</h1>
	</div>

	<button
		class="btn btn-square btn-outline btn-secondary absolute right-4 bottom-5 z-10 shadow-xl sm:right-8 sm:bottom-7"
		type="button"
		aria-label="Edit artist"
		onclick={onedit}
	>
		<svg class="size-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
			<path d="M12 20h9"></path>
			<path d="M16.5 3.5a2.1 2.1 0 0 1 3 3L8 18l-4 1 1-4Z"></path>
		</svg>
	</button>
</section>
