<script lang="ts">
	import { ratingColor } from '$lib/scripts/rating-colors';
	let {
		value = $bindable(5),
		disabled = false,
	}: {
		value?: number;
		disabled?: boolean;
	} = $props();

	const values = Array.from({ length: 10 }, (_, index) => index + 1);
</script>

<fieldset class="fieldset">
	<legend class="fieldset-legend">Rating</legend>

	<div class="w-full">
		<input
			type="range"
			min="1"
			max="10"
			step="1"
			class="range range-lg w-full"
			style:color={ratingColor(disabled ? undefined : value)}
			aria-label="Rating"
			{disabled}
			bind:value
		/>

		<div class="px-4">
			<div class="relative mt-1 h-4 text-center text-xs">
				{#each values as option}
					<span
						class="absolute -translate-x-1/2"
						style:color={ratingColor(disabled ? undefined : option)}
						style={`left: ${((option - 1) / 9) * 100}%`}
					>
						{option === value ? '↓' : '|'}
					</span>
				{/each}
			</div>
		</div>

		<div class="px-4">
			<div class="relative h-4 text-center text-xs">
				{#each values as option}
					<span
						class="absolute -translate-x-1/2"
						style:color={ratingColor(disabled ? undefined : option)}
						class:text-xl={option === value}
						style={`left: ${((option - 1) / 9) * 100}%`}
					>
						{option}
					</span>
				{/each}
			</div>
		</div>
	</div>
</fieldset>
