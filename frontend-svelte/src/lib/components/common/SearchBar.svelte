<script lang="ts">
	let { placeholder = 'Search', value = $bindable('') }: { placeholder?: string; value?: string } = $props();
</script>

<div class="relative w-full pt-3">
	<span class="bg-base-100 text-base-content/70 absolute top-1 left-3 z-10 px-1 text-xs leading-4 font-medium">
		Search
	</span>

	<label class="input input-md input-primary text-primary flex w-full items-center gap-2">
		<svg
			viewBox="0 0 24 24"
			fill="none"
			stroke="currentColor"
			stroke-width="2"
			class="h-4 w-4 shrink-0"
			aria-hidden="true"
		>
			<circle cx="11" cy="11" r="7"></circle>
			<path d="m21 21-4.3-4.3"></path>
		</svg>
		<input type="search" class="grow" {placeholder} bind:value />
	</label>
</div>
