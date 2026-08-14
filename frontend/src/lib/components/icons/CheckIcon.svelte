<script lang="ts">
	let { class: className = '', label }: { class?: string; label?: string } = $props();
</script>

<svg
	class={className}
	viewBox="0 0 24 24"
	fill="none"
	stroke="currentColor"
	stroke-width="4"
	stroke-linecap="round"
	stroke-linejoin="round"
	aria-label={label}
	aria-hidden={label ? undefined : 'true'}
>
	<path d="M20 6 9 17l-5-5"></path>
</svg>
