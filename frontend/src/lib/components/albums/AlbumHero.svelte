<script lang="ts">
	import EditIcon from '$lib/components/icons/EditIcon.svelte';
	import type { Album } from '$lib/scripts/types';

	let { album, onedit }: { album: Album; onedit?: () => void } = $props();
	let imageFailed = $state(false);

	$effect(() => {
		album.cover_url;
		imageFailed = false;
	});
</script>

<section class="relative -mx-4 -mt-2 h-[70dvh] overflow-hidden">
	{#if album.cover_url && !imageFailed}
		<img
			class="absolute inset-0 size-full scale-110 object-cover opacity-35 blur-3xl"
			src={album.cover_url}
			alt=""
			aria-hidden="true"
		/>
		<img
			class="absolute inset-0 size-full object-contain object-top"
			src={album.cover_url}
			alt={`${album.title} album cover`}
			onerror={() => (imageFailed = true)}
		/>
	{:else}
		<div
			class="from-primary/35 via-accent/25 to-secondary/25 absolute inset-0 flex items-center justify-center bg-linear-to-br"
		>
			<span class="text-base-content/20 text-[10rem] font-black" aria-hidden="true">
				{album.title.charAt(0).toUpperCase()}
			</span>
		</div>
	{/if}

	<div class="from-base-100 via-base-100/80 absolute inset-x-0 bottom-0 h-64 bg-linear-to-t to-transparent"></div>

	<div class="absolute inset-x-0 bottom-6 mx-auto max-w-5xl px-5 sm:bottom-8 sm:px-8">
		<h1
			class="pr-16 text-4xl leading-[0.95] font-black tracking-tight wrap-break-word text-white sm:text-6xl lg:text-7xl"
		>
			{album.title}
		</h1>
		{#if album.year}
			<p class="text-base-content/70 mt-3 pr-16 text-lg font-medium sm:text-xl">{album.year}</p>
		{/if}

		<button
			class="btn btn-square btn-outline btn-secondary absolute right-5 bottom-0 shadow-xl sm:right-8"
			type="button"
			aria-label="Edit album"
			onclick={onedit}
		>
			<EditIcon class="size-5" />
		</button>
	</div>
</section>
