<script lang="ts">
	import type { Component, Snippet } from 'svelte';

	let {
		title,
		description,
		icon: Icon,
		tone = 'primary',
		children,
	}: {
		title: string;
		description: string;
		icon: Component<{ class?: string }>;
		tone?: 'primary' | 'accent';
		children?: Snippet;
	} = $props();

	const tones = {
		primary: 'bg-primary/10 text-primary',
		accent: 'bg-accent/10 text-accent',
	};
</script>

<div class="flex items-center gap-4 p-4 sm:p-5">
	<div class={`${tones[tone]} flex size-11 shrink-0 items-center justify-center rounded-full`}>
		<Icon class="size-5" />
	</div>
	<div class="min-w-0 flex-1">
		<h3 class="font-bold">{title}</h3>
		<p class="text-base-content/55 text-sm">{description}</p>
	</div>
	{#if children}
		{@render children()}
	{:else}
		<span class="badge badge-ghost shrink-0">Coming soon</span>
	{/if}
</div>
