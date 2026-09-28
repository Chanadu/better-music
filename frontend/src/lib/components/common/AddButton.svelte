<script lang="ts">
	import PlusIcon from '$lib/components/icons/PlusIcon.svelte';

	let {
		label,
		onclick,
		class: className = '',
	}: {
		label: string;
		onclick?: () => void;
		class?: string;
	} = $props();
</script>

<button
	class={`btn btn-ghost border-base-content/25 hover:bg-base-300 w-full flex-col gap-2 rounded-xl border-dashed transition hover:-translate-y-1 hover:scale-[1.02] hover:shadow-xl active:translate-y-0 active:scale-[0.98] active:shadow-sm active:duration-75 motion-reduce:transform-none ${className}`}
	type="button"
	aria-label={label}
	{onclick}
>
	<PlusIcon class="text-base-content size-7" />
	<span class="text-base-content font-semibold">{label}</span>
</button>
