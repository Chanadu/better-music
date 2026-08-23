<script lang="ts">
	import PlusIcon from '../icons/PlusIcon.svelte';

	interface Props {
		subtitle: string;
		onclick?: () => void;
	}

	let { subtitle, onclick }: Props = $props();
</script>

<button
	class="rounded-box hover:bg-primary/10 min-w-0 text-left transition duration-200 ease-out hover:-translate-y-1 hover:scale-[1.02] hover:shadow-xl active:translate-y-0 active:scale-[0.98] active:shadow-sm active:duration-75 motion-reduce:transform-none"
	type="button"
	aria-label="Add album"
	{onclick}
>
	<span class="btn btn-ghost border-primary/35 rounded-box aspect-square h-auto w-full border-2 border-dashed">
		<PlusIcon class="text-primary size-10" />
	</span>
	<span class="text-primary mt-2 block font-semibold">Add album</span>
	<span class="text-base-content/50 mt-0.5 block text-sm">{subtitle}</span>
</button>
