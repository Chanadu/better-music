<script lang="ts">
	import ModalShell from '$lib/components/common/ModalShell.svelte';
	import { appSettings, daisyThemes, type ThemePreference } from '$lib/scripts/app-settings.svelte';
	import ThemeOption from './ThemeOption.svelte';

	const moreThemes: readonly ThemePreference[] = ['bettermusic', ...daisyThemes.slice(2)];

	let dialog = $state<HTMLDialogElement>();

	function themeName(theme: string): string {
		if (theme === 'bettermusic') return 'Better Music';
		return theme[0].toUpperCase() + theme.slice(1);
	}

	export function open() {
		dialog?.showModal();
	}

	function choose(theme: ThemePreference) {
		appSettings.set('theme', theme);
		dialog?.close();
	}
</script>

<ModalShell bind:dialog class="bg-base-100 text-base-content max-w-2xl">
	<h2 class="text-xl font-bold">Choose a theme</h2>
	<p class="text-base-content/60 mt-1 text-sm">Select any DaisyUI palette for Better Music.</p>

	<div class="mt-5 grid grid-cols-2 gap-2 sm:grid-cols-3">
		{#each moreThemes as theme}
			<ThemeOption
				{theme}
				label={themeName(theme)}
				variant="catalog"
				selected={appSettings.values.theme === theme}
				onclick={() => choose(theme)}
			/>
		{/each}
	</div>

	<div class="modal-action">
		<button type="button" class="btn hover-lift btn-error cursor-pointer" onclick={() => dialog?.close()}>Cancel</button>
	</div>
</ModalShell>
