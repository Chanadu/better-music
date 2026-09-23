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
	<path d="M3 14v-3a9 9 0 0 1 18 0v3" />
	<rect x="3" y="12" width="4" height="9" rx="2" />
	<rect x="17" y="12" width="4" height="9" rx="2" />
</svg>
