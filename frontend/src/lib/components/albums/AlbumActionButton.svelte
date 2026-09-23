<script lang="ts">
	import type { Snippet } from 'svelte';

	let {
		color,
		disabled = false,
		onclick,
		children,
	}: {
		color: 'primary' | 'secondary' | 'success';
		disabled?: boolean;
		onclick?: () => void;
		children: Snippet;
	} = $props();
</script>

<button
	type="button"
	class="btn btn-soft h-14 w-full min-w-0 flex-col gap-1.5 rounded-xl p-2 text-xs font-semibold whitespace-nowrap shadow-none transition-[background-color,border-color,box-shadow] duration-150 lg:h-11 lg:flex-row lg:justify-center lg:gap-2 lg:px-4 lg:text-sm @min-[360px]:h-11 @min-[360px]:flex-row @min-[360px]:gap-2 @min-[360px]:text-[0.8125rem] [&>svg]:size-4.5 [&>svg]:shrink-0"
	class:btn-primary={color === 'primary'}
	class:btn-secondary={color === 'secondary'}
	class:btn-success={color === 'success'}
	{disabled}
	{onclick}
>
	{@render children()}
</button>
