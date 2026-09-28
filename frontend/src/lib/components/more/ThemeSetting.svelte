<script lang="ts">
	import EyeIcon from '$lib/components/icons/EyeIcon.svelte';
	import GridIcon from '$lib/components/icons/GridIcon.svelte';
	import { appSettings, type ThemePreference } from '$lib/scripts/app-settings.svelte';
	import ThemeOption from './ThemeOption.svelte';
	import ThemePalette from './ThemePalette.svelte';
	import ThemePickerDialog from './ThemePickerDialog.svelte';

	const quickThemes = ['system', 'light', 'dark'] as const satisfies readonly ThemePreference[];

	let themePickerDialog: ThemePickerDialog;

	function themeName(theme: string): string {
		if (theme === 'bettermusic') return 'Better Music';
		return theme[0].toUpperCase() + theme.slice(1);
	}

	function themeLabel(theme: ThemePreference): string {
		return theme === 'system' ? `Device (${themeName(appSettings.deviceTheme)})` : themeName(theme);
	}

	function choose(theme: ThemePreference) {
		appSettings.set('theme', theme);
	}
</script>

<div class="p-4 sm:p-5">
	<div class="flex items-center gap-4">
		<div class="bg-primary text-primary-content flex size-11 shrink-0 items-center justify-center rounded-full">
			<EyeIcon class="size-5" />
		</div>
		<div>
			<h3 class="font-bold">Theme</h3>
			<p class="text-base-content/55 text-sm">Choose a look for Better Music.</p>
		</div>
	</div>

	<div class="mt-4 grid gap-4 sm:grid-cols-[1fr_4fr] sm:gap-6">
		<section>
			<p class="text-base-content/50 mb-2 text-xs font-bold tracking-wider uppercase">Current</p>
			<ThemePalette
				theme={appSettings.values.theme === 'system' ? appSettings.deviceTheme : appSettings.values.theme}
				label={themeLabel(appSettings.values.theme)}
				variant="current"
			/>
		</section>

		<section class="border-base-300 border-t pt-4 sm:border-t-0 sm:border-l sm:pt-0 sm:pl-6">
			<p class="text-base-content/50 mb-2 text-xs font-bold tracking-wider uppercase">Choose a theme</p>
			<div class="grid grid-cols-2 gap-3 sm:grid-cols-4">
				{#each quickThemes as theme}
					<ThemeOption
						theme={theme === 'system' ? appSettings.deviceTheme : theme}
						label={themeLabel(theme)}
						variant="quick"
						selected={appSettings.values.theme === theme}
						onclick={() => choose(theme)}
					/>
				{/each}
				<button
					type="button"
					class="bg-base-100 text-base-content focus-visible:outline-base-content cursor-pointer rounded-xl p-3 text-left shadow-sm transition duration-200 hover:-translate-y-0.5 hover:shadow-lg focus-visible:outline-2 focus-visible:outline-offset-2 active:translate-y-0 active:scale-[0.98] motion-reduce:transform-none"
					onclick={() => themePickerDialog.open()}
				>
					<GridIcon class="mb-3 size-4" />
					<span class="text-sm font-bold">More themes</span>
				</button>
			</div>
		</section>
	</div>
</div>

<ThemePickerDialog bind:this={themePickerDialog} />
