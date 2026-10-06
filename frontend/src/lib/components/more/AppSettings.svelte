<script lang="ts">
	import EditIcon from '$lib/components/icons/EditIcon.svelte';
	import GridIcon from '$lib/components/icons/GridIcon.svelte';
	import MoreIcon from '$lib/components/icons/MoreIcon.svelte';
	import StarIcon from '$lib/components/icons/StarIcon.svelte';
	import { appSettings } from '$lib/scripts/app-settings.svelte';
	import ExportLibrarySetting from './ExportLibrarySetting.svelte';
	import RatingSettingsDialog from './RatingSettingsDialog.svelte';
	import SettingRow from './SettingRow.svelte';
	import SyncStatusSetting from './SyncStatusSetting.svelte';
	import ThemeSetting from './ThemeSetting.svelte';

	let ratingSettingsDialog: RatingSettingsDialog;

	function setDropdownStyle(event: Event) {
		appSettings.set('useNativeDropdowns', (event.currentTarget as HTMLInputElement).checked);
	}

	const unavailableSettings = [
		{
			title: 'Default library layout',
			description: 'Choose grid or compact list views for albums and artists.',
			icon: GridIcon,
		},
	] as const;

	const unavailableTools = [
		{
			title: 'Help, shortcuts, and app information',
			description: 'Learn how Better Music works and find app details.',
			icon: MoreIcon,
		},
	] as const;
</script>

<section aria-labelledby="settings-heading">
	<div class="mb-5">
		<h2 id="settings-heading" class="text-base-content text-lg font-bold tracking-[0.14em] uppercase sm:text-xl">
			Settings
		</h2>
		<p class="text-base-content/55 mt-1 text-sm">Preferences are saved on this device.</p>
	</div>

	<div class="bg-base-200 divide-base-300 divide-y rounded-xl shadow-sm">
		<ThemeSetting />
		{#each unavailableSettings as setting}
			<SettingRow {...setting} />
		{/each}
		<SettingRow
			title="Rating labels and colors"
			description="Personalize rating names and colors while keeping scores numeric."
			icon={StarIcon}
		>
			<button
				type="button"
				class="btn hover-lift btn-primary btn-md gap-2 rounded-full shadow-sm"
				onclick={() => ratingSettingsDialog.open()}
			>
				<EditIcon class="size-4" />
				Customize
			</button>
		</SettingRow>
		<SettingRow
			title="Dropdown menu style"
			description="Choose native controls or custom-styled menus."
			icon={MoreIcon}
		>
			<label class="flex cursor-pointer items-center gap-3">
				<span
					class={`text-sm font-bold transition-colors ${
						appSettings.values.useNativeDropdowns ? 'text-base-content/40' : 'text-base-content'
					}`}>Custom</span
				>
				<input
					type="checkbox"
					class="toggle toggle-lg"
					style:--input-color={'var(--color-primary)'}
					checked={appSettings.values.useNativeDropdowns}
					onchange={setDropdownStyle}
					aria-label="Use native dropdown menus"
				/>
				<span
					class={`text-sm font-bold transition-colors ${
						appSettings.values.useNativeDropdowns ? 'text-base-content' : 'text-base-content/40'
					}`}>Native</span
				>
			</label>
		</SettingRow>
	</div>

	<h3 class="text-base-content/45 mt-7 mb-3 px-1 text-xs font-bold tracking-[0.14em] uppercase">Data & support</h3>
	<div class="bg-base-200 divide-base-300 divide-y rounded-xl shadow-sm">
		<ExportLibrarySetting />
		<SyncStatusSetting />
		{#each unavailableTools as setting}
			<SettingRow {...setting} tone="accent" />
		{/each}
	</div>
</section>

<RatingSettingsDialog bind:this={ratingSettingsDialog} />
