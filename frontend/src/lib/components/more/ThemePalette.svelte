<script lang="ts">
	import CheckIcon from '$lib/components/icons/CheckIcon.svelte';

	let {
		theme,
		label,
		variant,
		selected = false,
	}: {
		theme: string;
		label: string;
		variant: 'current' | 'quick' | 'catalog';
		selected?: boolean;
	} = $props();

	const surfaces = {
		current: 'border-primary/60 bg-base-100 rounded-xl border-2 p-3 shadow-md',
		quick: 'bg-base-100 rounded-xl border-2 p-3 shadow-sm',
		catalog: 'bg-base-200 rounded-xl p-4 shadow-sm',
	};
</script>

<div data-theme={theme} class={`rounded-xl text-left ${variant === 'current' ? '' : 'h-full'}`}>
	<div
		class={`${surfaces[variant]} relative ${variant === 'current' ? '' : 'h-full'} ${
			variant === 'quick' ?
				selected ? 'border-primary/60'
				:	'border-transparent'
			:	''
		}`}
	>
		<span class="mb-3 flex gap-1" aria-hidden="true">
			<span class="bg-primary size-4 rounded-full"></span>
			<span class="bg-secondary size-4 rounded-full"></span>
			<span class="bg-accent size-4 rounded-full"></span>
		</span>
		<span class="text-base-content block text-sm font-bold">{label}</span>

		{#if selected}
			<span
				class="bg-primary text-primary-content absolute top-2 right-2 flex size-5 items-center justify-center rounded-full shadow-sm"
				aria-label="Current theme"
			>
				<CheckIcon class="size-3" />
			</span>
		{/if}
	</div>
</div>
