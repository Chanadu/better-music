<script lang="ts">
	import ThemePalette from './ThemePalette.svelte';

	let {
		theme,
		label,
		variant,
		selected = false,
		onclick,
	}: {
		theme: string;
		label: string;
		variant: 'quick' | 'catalog';
		selected?: boolean;
		onclick: () => void;
	} = $props();
</script>

<button
	type="button"
	class="focus-visible:outline-base-content relative cursor-pointer rounded-xl text-left transition duration-200 hover:-translate-y-0.5 hover:shadow-lg focus-visible:outline-2 focus-visible:outline-offset-2 active:translate-y-0 active:scale-[0.98] motion-reduce:transform-none"
	{onclick}
	aria-pressed={selected}
>
	<ThemePalette {theme} {label} {variant} {selected} />
</button>
