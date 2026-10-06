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
	class={`btn btn-ghost border-base-content/25 hover:bg-base-300 hover-lift w-full flex-col gap-2 rounded-xl border-dashed ${className}`}
	type="button"
	aria-label={label}
	{onclick}
>
	<PlusIcon class="text-base-content size-7" />
	<span class="text-base-content font-semibold">{label}</span>
</button>
