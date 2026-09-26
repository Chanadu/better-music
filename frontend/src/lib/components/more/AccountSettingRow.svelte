<script lang="ts">
	import type { Component } from 'svelte';

	type Tone = 'primary' | 'accent' | 'error';

	let {
		title,
		description,
		actionLabel,
		icon: Icon,
		tone = 'primary',
		disabled = false,
		onclick,
	}: {
		title: string;
		description: string;
		actionLabel: string;
		icon: Component<{ class?: string }>;
		tone?: Tone;
		disabled?: boolean;
		onclick: () => void;
	} = $props();

	const iconClasses: Record<Tone, string> = {
		primary: 'bg-primary/10 text-primary',
		accent: 'bg-accent/10 text-accent',
		error: 'bg-error/10 text-error',
	};
	const buttonClasses: Record<Tone, string> = {
		primary: 'btn-primary',
		accent: 'btn-accent',
		error: 'btn-error',
	};
</script>

<div class="flex items-center gap-4 p-4 sm:p-5">
	<div class={`${iconClasses[tone]} flex size-11 shrink-0 items-center justify-center rounded-full`}>
		<Icon class="size-5" />
	</div>
	<div class="min-w-0 flex-1">
		<h3 class="font-bold">{title}</h3>
		<p class="text-base-content/55 truncate text-sm">{description}</p>
	</div>
	<button type="button" class={`btn btn-soft btn-sm ${buttonClasses[tone]}`} {disabled} {onclick}>
		{actionLabel}
	</button>
</div>
