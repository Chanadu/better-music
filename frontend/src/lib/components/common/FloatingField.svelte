<script lang="ts">
	let { label, children }: { label: string; children: import('svelte').Snippet } = $props();
</script>

<div class="relative w-full pt-3">
	<span class="bg-base-100 text-base-content/70 absolute top-1 left-3 z-10 px-1 text-xs leading-4 font-medium">
		{label}
	</span>
	{@render children()}
</div>
