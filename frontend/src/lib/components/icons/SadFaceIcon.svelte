<script lang="ts">
	let { class: className = '' }: { class?: string } = $props();
</script>

<svg class={className} viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
	<circle cx="12" cy="12" r="9"></circle>
	<path d="M9 9h.01M15 9h.01M8 16c1-1.3 2.3-2 4-2s3 .7 4 2"></path>
</svg>
