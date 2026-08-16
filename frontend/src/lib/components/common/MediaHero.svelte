<script lang="ts">
	import BackIcon from '$lib/components/icons/BackIcon.svelte';
	import EditIcon from '$lib/components/icons/EditIcon.svelte';

	let {
		title,
		subtitle,
		imageUrl,
		imageAlt,
		editLabel,
		backHref,
		onedit,
	}: {
		title: string;
		subtitle?: string | number | null;
		imageUrl?: string | null;
		imageAlt: string;
		editLabel: string;
		backHref: string;
		onedit?: () => void;
	} = $props();

	let imageFailed = $state(false);

	$effect(() => {
		imageUrl;
		imageFailed = false;
	});
</script>

<section class="relative -mx-4 -mt-2 h-[60dvh] overflow-hidden">
	<a
		class="btn btn-square btn-soft btn-accent absolute top-4 left-4 z-10 shadow-xl sm:top-6 sm:left-8"
		href={backHref}
		aria-label="Go back"
	>
		<BackIcon class="size-6" />
	</a>

	{#if imageUrl && !imageFailed}
		<img
			class="absolute inset-0 size-full object-cover"
			src={imageUrl}
			alt={imageAlt}
			onerror={() => (imageFailed = true)}
		/>
	{:else}
		<div
			class="from-primary/45 via-accent/20 to-base-300 absolute inset-0 flex items-center justify-center bg-linear-to-br"
		>
			<span class="text-base-content/20 text-[10rem] font-black sm:text-[15rem]" aria-hidden="true">
				{title.charAt(0).toUpperCase()}
			</span>
		</div>
	{/if}

	<div class="from-base-100 absolute inset-0 bg-linear-to-t via-black/15 to-black/5"></div>
	<div class="from-base-100 absolute inset-x-0 bottom-0 h-24 bg-linear-to-t to-transparent"></div>

	<div class="absolute inset-x-0 bottom-5 px-4 sm:bottom-7 sm:px-8">
		<h1
			class="pr-16 text-4xl leading-none font-black tracking-tight wrap-break-word text-white sm:text-6xl lg:text-7xl"
		>
			{title}
		</h1>
		{#if subtitle !== undefined && subtitle !== null && subtitle !== ''}
			<p class="text-base-content/70 mt-3 pr-16 text-lg font-medium sm:text-xl">{subtitle}</p>
		{/if}
	</div>

	<button
		class="btn btn-square btn-outline btn-secondary absolute right-4 bottom-5 z-10 shadow-xl sm:right-8 sm:bottom-7"
		type="button"
		aria-label={editLabel}
		onclick={onedit}
	>
		<EditIcon class="size-5" />
	</button>
</section>
