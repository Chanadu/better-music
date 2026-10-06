<script lang="ts">
	let { class: className = '' }: { class?: string } = $props();
</script>

<svg
	class={className}
	viewBox="0 0 24 24"
	fill="none"
	stroke="currentColor"
	stroke-width="2"
	stroke-linecap="round"
	stroke-linejoin="round"
	aria-hidden="true"
>
	<path d="M20 4v6h-6M4 20v-6h6" />
	<path d="M20 10a8 8 0 0 0-14-4M4 14a8 8 0 0 0 14 4" />
</svg>
