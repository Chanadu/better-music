<script lang="ts">
	import FloatingField from './FloatingField.svelte';

	let { value = $bindable(''), valid = $bindable(true) }: { value?: string; valid?: boolean } = $props();

	const fieldId = $props.id();
	let yearValid = $derived(
		value === '' ||
			(value.length === 4 &&
				[...value].every(isDigit) &&
				Number(value) >= 1000 &&
				Number(value) <= new Date().getFullYear()),
	);

	$effect(() => {
		valid = yearValid;
	});

	function isDigit(character: string) {
		return character >= '0' && character <= '9';
	}

	function cleanYear(event: Event) {
		const input = event.currentTarget as HTMLInputElement;
		value = [...input.value].filter(isDigit).join('').slice(0, 4);
		input.value = value;
	}
</script>

<FloatingField label="Year">
	<input
		type="text"
		aria-label="Year"
		inputmode="numeric"
		maxlength="4"
		class="input validator w-full"
		aria-invalid={!yearValid}
		aria-describedby={!yearValid ? `${fieldId}-error` : undefined}
		placeholder="2014"
		{value}
		oninput={cleanYear}
	/>
	<div id={`${fieldId}-error`} class="validator-hint hidden">
		Enter a year between 1000 and {new Date().getFullYear()}
	</div>
</FloatingField>
