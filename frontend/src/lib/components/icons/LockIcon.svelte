<script lang="ts">
	let { class: className = '' }: { class?: string } = $props();
</script>

<svg
	class={className}
	viewBox="0 0 24 24"
	fill="none"
	stroke="currentColor"
	stroke-width="2.5"
	stroke-linecap="round"
	stroke-linejoin="round"
	aria-hidden="true"
>
	<rect width="18" height="11" x="3" y="11" rx="2"></rect>
	<path d="M7 11V7a5 5 0 0 1 10 0v4"></path>
</svg>
