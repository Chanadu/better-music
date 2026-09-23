<script lang="ts">
	let { class: className = '' }: { class?: string } = $props();
</script>

<svg
	xmlns="http://www.w3.org/2000/svg"
	class={className}
	viewBox="0 0 24 24"
	fill="none"
	stroke="currentColor"
	stroke-width="2"
	stroke-linecap="round"
	stroke-linejoin="round"
	aria-hidden="true"
>
	<path
		d="m18 14 4 4-4 4M18 2l4 4-4 4M2 18h2.5c6 0 9-12 15-12H22M2 6h2.5c2 0 3.5 1.3 5 3.2M14.5 14.8C16 16.7 17.5 18 19.5 18H22"
	/>
</svg>
