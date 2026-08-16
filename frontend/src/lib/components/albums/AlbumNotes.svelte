<script lang="ts">
	let { comment }: { comment?: string } = $props();
</script>

<section class="mt-9">
	<h2 class="text-secondary mb-4 text-sm font-black uppercase">Your notes</h2>
	{#if comment?.trim()}
		<div
			class="border-secondary bg-base-200/50 relative overflow-hidden rounded-r-2xl border-l-4 px-6 py-7 sm:px-9 sm:py-8"
		>
			<p class="text-base-content/80 relative max-w-3xl text-lg leading-8 whitespace-pre-wrap sm:text-xl">
				{comment}
			</p>
		</div>
	{:else}
		<div
			class="border-base-content/15 bg-base-200/25 flex min-h-22 items-center rounded-xl border border-dashed px-6 sm:min-h-24 sm:px-9"
		>
			<p class="text-base-content/45 text-sm italic">No comments yet.</p>
		</div>
	{/if}
</section>
