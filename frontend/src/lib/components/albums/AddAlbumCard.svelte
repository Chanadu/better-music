<script lang="ts">
	import PlusIcon from '../icons/PlusIcon.svelte';

	interface Props {
		onclick?: () => void;
	}

	let { onclick }: Props = $props();
</script>

<button
	class="btn btn-ghost border-primary/35 hover:bg-primary/10 aspect-square h-auto w-full flex-col gap-2 rounded-xl border-2 border-dashed transition duration-200 ease-out hover:-translate-y-1 hover:scale-[1.02] hover:shadow-xl active:translate-y-0 active:scale-[0.98] active:shadow-sm active:duration-75 motion-reduce:transform-none"
	type="button"
	aria-label="Add album"
	{onclick}
>
	<PlusIcon class="text-primary size-7" />
	<span class="text-primary font-semibold">Add album</span>
</button>
