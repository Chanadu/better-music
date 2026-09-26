<script lang="ts">
	import SortControls from '$lib/components/common/SortControls.svelte';

	let {
		label,
		options,
		preference,
	}: {
		label: string;
		options: readonly { label: string; value: string }[];
		preference: { sort: string; reversed: boolean };
	} = $props();
</script>

<div class="min-w-0">
	<span class="text-base-content/50 mb-1.5 block px-1 text-xs font-bold tracking-[0.08em] uppercase">{label}</span>
	<SortControls
		{options}
		bind:sort={preference.sort}
		bind:reversed={preference.reversed}
		name={`settings-${label.toLowerCase().replaceAll(' ', '-')}`}
		fullWidth
	/>
</div>
