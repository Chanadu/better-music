<script lang="ts">
	import SearchIcon from '../icons/SearchIcon.svelte';

	let { placeholder = 'Search', value = $bindable('') }: { placeholder?: string; value?: string } = $props();
</script>

<div class="relative w-full pt-3">
	<span class="bg-base-100 text-base-content/70 absolute top-1 left-3 z-10 px-1 text-xs leading-4 font-medium">
		Search
	</span>

	<label class="input input-primary text-primary w-full">
		<SearchIcon class="h-4 w-4 shrink-0" />
		<input type="search" class="grow" {placeholder} bind:value />
	</label>
</div>
