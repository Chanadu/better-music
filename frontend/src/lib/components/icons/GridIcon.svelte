<script lang="ts">
	let { class: className = '' }: { class?: string } = $props();
</script>

<svg
	class={className}
	viewBox="0 0 24 24"
	fill="none"
	stroke="currentColor"
	stroke-width="1.8"
	stroke-linecap="round"
	stroke-linejoin="round"
	aria-hidden="true"
>
	<rect x="3.5" y="3.5" width="6.5" height="6.5" rx="1"></rect>
	<rect x="14" y="3.5" width="6.5" height="6.5" rx="1"></rect>
	<rect x="3.5" y="14" width="6.5" height="6.5" rx="1"></rect>
	<rect x="14" y="14" width="6.5" height="6.5" rx="1"></rect>
</svg>
