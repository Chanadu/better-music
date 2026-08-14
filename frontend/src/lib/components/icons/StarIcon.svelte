<script lang="ts">
	let {
		class: className = '',
		filled = false,
		slashed = false,
	}: { class?: string; filled?: boolean; slashed?: boolean } = $props();
</script>

<svg
	class={className}
	viewBox="0 0 24 24"
	fill={filled ? 'currentColor' : 'none'}
	stroke="currentColor"
	stroke-width={filled ? undefined : 1.8}
	stroke-linecap="round"
	stroke-linejoin="round"
	aria-hidden="true"
>
	<path d="m12 2.5 2.9 5.9 6.5.9-4.7 4.6 1.1 6.5-5.8-3-5.8 3 1.1-6.5-4.7-4.6 6.5-.9Z"></path>
	{#if slashed}
		<path d="M3 3l18 18" fill="none" stroke-width="2.2"></path>
	{/if}
</svg>
