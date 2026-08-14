<script lang="ts">
	import FloatingField from '../common/FloatingField.svelte';
	import MediaThumbnail from '../common/MediaThumbnail.svelte';

	let { name = $bindable('') }: { name?: string } = $props();
</script>

<div class="flex flex-col gap-3">
	<FloatingField label="Artist Name">
		<input type="text" class="input w-full" placeholder="e.g. J. Cole" bind:value={name} />
	</FloatingField>

	<div class="flex justify-center">
		<MediaThumbnail
			variant="artist-preview"
			label={name}
			alt={name ? `${name} artist preview` : ''}
			emptyFallback=""
		/>
	</div>
</div>
