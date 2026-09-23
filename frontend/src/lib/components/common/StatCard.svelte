<script lang="ts">
	import type { Component } from 'svelte';

	type Tone = 'primary' | 'secondary' | 'accent';

	let {
		value,
		label,
		icon: Icon,
		tone = 'primary',
		valueClass = 'text-2xl',
		color,
		valueColor = color,
		class: className = '',
	}: {
		value: string | number;
		label: string;
		icon: Component<{ class?: string }>;
		tone?: Tone;
		valueClass?: string;
		color?: string;
		valueColor?: string;
		class?: string;
	} = $props();

	const toneClasses: Record<Tone, string> = {
		primary: 'bg-primary/10 text-primary',
		secondary: 'bg-secondary/10 text-secondary',
		accent: 'bg-accent/10 text-accent',
	};
</script>

<div class={`bg-base-200 flex min-h-28 min-w-0 items-center gap-4 rounded-xl px-5 py-4 shadow-sm ${className}`}>
	<div
		class={`${toneClasses[tone]} flex size-11 shrink-0 items-center justify-center rounded-full`}
		style:color
		style:background-color={color ? `color-mix(in srgb, ${color} 10%, transparent)` : undefined}
	>
		<Icon class="size-5" />
	</div>
	<div class="min-w-0">
		<p class="text-base-content/50 text-xs font-bold tracking-wider uppercase">{label}</p>
		<p class={`mt-0.5 truncate font-black ${valueClass}`} style:color={valueColor} title={String(value)}>{value}</p>
	</div>
</div>
