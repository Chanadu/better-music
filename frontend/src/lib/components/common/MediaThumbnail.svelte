<script lang="ts">
	let {
		imageUrl = '',
		alt = '',
		label = '',
		fallbackText,
		emptyFallback = '?',
		variant = 'list',
	}: {
		imageUrl?: string;
		alt?: string;
		label?: string;
		fallbackText?: string;
		emptyFallback?: string;
		variant?: 'card' | 'list' | 'result' | 'artist-preview' | 'album-preview';
	} = $props();

	let failed = $state(false);
	let shownUrl = $derived(failed ? '' : imageUrl);
	let fallback = $derived(fallbackText ?? label.trim().charAt(0).toUpperCase() ?? emptyFallback);

	const classes = {
		card: 'aspect-square w-full text-3xl',
		list: 'size-10 shrink-0 text-sm',
		result: 'size-10 shrink-0 text-sm',
		'artist-preview': 'size-44 text-5xl sm:size-56 sm:text-6xl',
		'album-preview': 'aspect-square w-full text-4xl',
	};

	$effect(() => {
		imageUrl;
		failed = false;
	});
</script>

<div class={`block overflow-hidden ${classes[variant]}`}>
	{#if shownUrl}
		<img
			class="rounded-box bg-base-300 size-full object-cover"
			src={shownUrl}
			{alt}
			onerror={() => (failed = true)}
		/>
	{:else}
		<div
			class="rounded-box bg-base-300 flex size-full items-center justify-center font-semibold"
			role={fallback ? 'img' : undefined}
			aria-label={fallback ? alt : undefined}
			aria-hidden={!fallback}
		>
			{fallback || emptyFallback}
		</div>
	{/if}
</div>
