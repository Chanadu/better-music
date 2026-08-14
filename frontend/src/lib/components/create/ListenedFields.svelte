<script lang="ts">
	import FloatingField from '../common/FloatingField.svelte';
	import CheckIcon from '../icons/CheckIcon.svelte';
	import XIcon from '../icons/XIcon.svelte';
	import RatingInput from './RatingInput.svelte';

	let {
		listened = $bindable(false),
		listenedAt = $bindable(''),
		rating = $bindable(5),
	}: { listened?: boolean; listenedAt?: string; rating?: number } = $props();

	function today() {
		const date = new Date();
		return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`;
	}

	function toggleListened() {
		if (listened && !listenedAt) listenedAt = today();
	}
</script>

<div class="px-2 pt-4">
	<label class="group divider mb-1 cursor-pointer">
		<span class="flex items-center gap-3">
			<span class="toggle has-checked:bg-primary has-checked:text-primary-content">
				<input type="checkbox" bind:checked={listened} onchange={toggleListened} />

				<XIcon label="disabled" />
				<CheckIcon class="text-primary" label="enabled" />
			</span>

			<span class="text-neutral-content group-has-checked:text-primary"> Listened </span>
		</span>
	</label>

	<div class="flex flex-col gap-2" class:opacity-40={!listened}>
		<RatingInput disabled={!listened} bind:value={rating} />

		<FloatingField label="Listened At">
			<input type="date" class="input w-full" disabled={!listened} bind:value={listenedAt} />
		</FloatingField>
	</div>
</div>
