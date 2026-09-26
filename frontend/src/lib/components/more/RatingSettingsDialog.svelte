<script lang="ts">
	import ModalShell from '$lib/components/common/ModalShell.svelte';
	import { appSettings, defaultRatingColors, defaultRatingLabels } from '$lib/scripts/app-settings.svelte';

	let dialog = $state<HTMLDialogElement>();
	let labels = $state<string[]>([]);
	let colors = $state<string[]>([]);

	export function open() {
		labels = [...appSettings.values.ratingLabels];
		colors = [...appSettings.values.ratingColors];
		dialog?.showModal();
	}

	function save() {
		appSettings.set(
			'ratingLabels',
			labels.map((label) => label.trim()),
		);
		appSettings.set('ratingColors', [...colors]);
		dialog?.close();
	}

	function reset() {
		labels = [...defaultRatingLabels];
		colors = [...defaultRatingColors];
	}
</script>

<ModalShell bind:dialog>
	<h2 class="text-xl font-bold">Rating labels and colors</h2>
	<p class="text-base-content/60 mt-1 text-sm">Customize how each score appears on this device.</p>

	<form
		class="mt-5"
		onsubmit={(event) => {
			event.preventDefault();
			save();
		}}
	>
		<div class="grid grid-cols-[auto_1fr_auto] items-center gap-x-3 gap-y-2">
			<span class="text-base-content/45 text-xs font-bold tracking-wider uppercase">Score</span>
			<span class="text-base-content/45 text-xs font-bold tracking-wider uppercase">Label</span>
			<span class="text-base-content/45 text-xs font-bold tracking-wider uppercase">Color</span>

			{#each labels as _, index}
				<label class="text-center font-bold tabular-nums" for={`rating-label-${index + 1}`}>{index + 1}</label>
				<input
					id={`rating-label-${index + 1}`}
					type="text"
					class="input input-sm w-full"
					maxlength="40"
					aria-label={`Label for rating ${index + 1}`}
					bind:value={labels[index]}
				/>
				<label
					class="border-base-300 has-focus-visible:outline-primary relative size-9 cursor-pointer overflow-hidden rounded-lg border-2 has-focus-visible:outline-2 has-focus-visible:outline-offset-2"
					style:background-color={colors[index]}
					title={`Choose color for rating ${index + 1}`}
				>
					<input
						type="color"
						class="absolute inset-0 size-full cursor-pointer opacity-0"
						aria-label={`Color for rating ${index + 1}`}
						bind:value={colors[index]}
					/>
				</label>
			{/each}
		</div>

		<div class="modal-action flex-wrap items-center">
			<button type="button" class="btn btn-ghost w-full sm:mr-auto sm:w-auto" onclick={reset}
				>Reset defaults</button
			>
			<button type="button" class="btn btn-ghost flex-1 sm:flex-none" onclick={() => dialog?.close()}
				>Cancel</button
			>
			<button type="submit" class="btn btn-primary flex-1 sm:flex-none">Save</button>
		</div>
	</form>
</ModalShell>
