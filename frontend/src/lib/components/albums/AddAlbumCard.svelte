<script lang="ts">
	import PlusIcon from '../icons/PlusIcon.svelte';

	interface Props {
		subtitle: string;
		onclick?: () => void;
	}

	let { subtitle, onclick }: Props = $props();
</script>

<button class="group min-w-0 text-left" type="button" aria-label="Add album" {onclick}>
	<span
		class="btn btn-ghost border-primary/35 hover:border-primary hover:bg-primary/10 rounded-box aspect-square h-auto w-full border-2 border-dashed"
	>
		<PlusIcon class="text-primary size-10 transition-transform group-hover:scale-110" />
	</span>
	<span class="text-primary mt-2 block font-semibold">Add album</span>
	<span class="text-base-content/50 mt-0.5 block text-sm">{subtitle}</span>
</button>
