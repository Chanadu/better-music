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
	<path d="M12 3v12"></path>
	<path d="m7 10 5 5 5-5"></path>
	<path d="M5 21h14a2 2 0 0 0 2-2v-3"></path>
	<path d="M3 16v3a2 2 0 0 0 2 2"></path>
</svg>
