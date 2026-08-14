<script lang="ts">
	import type { Component } from 'svelte';

	let {
		value,
		label,
		icon: Icon,
	}: { value: string | number; label: string; icon?: Component<{ class?: string }> } = $props();
</script>

<div class="stat min-w-0 px-3 py-4 text-center">
	<div class="flex min-w-0 items-center justify-center gap-1.5">
		{#if Icon}
			<div class="text-primary shrink-0"><Icon class="size-5" /></div>
		{/if}
		<div class="stat-value min-w-0 text-xl sm:text-3xl">{value}</div>
	</div>
	<div class="stat-desc text-base-content/55">{label}</div>
</div>
