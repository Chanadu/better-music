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
	class="focus-visible:outline-base-content relative cursor-pointer rounded-xl text-left hover-lift focus-visible:outline-2 focus-visible:outline-offset-2"
	{onclick}
	aria-pressed={selected}
>
	<ThemePalette {theme} {label} {variant} {selected} />
</button>
