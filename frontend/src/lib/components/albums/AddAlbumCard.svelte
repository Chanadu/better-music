<script lang="ts">
	import AddButton from '$lib/components/common/AddButton.svelte';

	interface Props {
		onclick?: () => void;
	}

	let { onclick }: Props = $props();
</script>

<AddButton label="Add album" class="aspect-square h-auto" {onclick} />
