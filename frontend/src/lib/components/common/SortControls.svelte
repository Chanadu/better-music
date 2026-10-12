<script lang="ts" generics="T extends string">
	import { getContext } from 'svelte';
	import { previewContext, type PreviewContext } from '$lib/components/preview/context';
	const preview = getContext<PreviewContext | undefined>(previewContext);
	import { appSettings } from '$lib/scripts/app-settings.svelte';

	let {
		options,
		sort = $bindable(),
		reversed = $bindable(false),
		name,
		labelWidth = '6ch',
		fullWidth = false,
	}: {
		options: readonly { label: string; value: T }[];
		sort: T;
		reversed?: boolean;
		name: string;
		labelWidth?: string;
		fullWidth?: boolean;
	} = $props();
	function chooseSort(event: Event) {
		sort = (event.currentTarget as HTMLInputElement).value as T;
		if (!preview) (event.currentTarget as HTMLElement).closest('details')?.removeAttribute('open');
	}
</script>

<div class="join relative has-[details[open]]:z-20" class:w-full={fullWidth}>
	{#if preview?.nativeDropdowns ?? appSettings.values.useNativeDropdowns}
		<select
			style:background-image={'linear-gradient(45deg, transparent 50%, currentColor 50%), linear-gradient(135deg, currentColor 50%, transparent 50%)'}
			style:background-position="calc(100% - 20px) 50%, calc(100% - 16px) 50%"
			style:background-size="4px 4px"
			style:background-repeat="no-repeat"
			class="btn btn-secondary join-item hover-lift appearance-none justify-between pr-10 text-left shadow-none"
			class:flex-1={fullWidth}
			class:min-w-0={fullWidth}
			style:width={fullWidth ? '100%' : `calc(${labelWidth} + 4rem)`}
			style:min-width={fullWidth ? '0' : `calc(${labelWidth} + 4rem)`}
			aria-label={name}
			bind:value={sort}
		>
			{#each options as option}
				<option value={option.value}>
					{option.label}
				</option>
			{/each}
		</select>
	{:else}

		<details open={preview ? true : undefined} class="dropdown" class:flex-1={fullWidth} class:min-w-0={fullWidth}>
			<summary
				class="btn btn-secondary join-item hover-lift justify-between shadow-none"
				style:width={fullWidth ? '100%' : `calc(${labelWidth} + 4rem)`}
				style:min-width={fullWidth ? '0' : `calc(${labelWidth} + 4rem)`}
			>
				{options.find((option) => option.value === sort)?.label}
			</summary>

			<ul
				class="dropdown-content menu bg-base-100 border-base-300 text-base-content rounded-box z-10 mt-2 w-full border-2 shadow"
			>
				{#each options as option}
					<li>
						<label>
							<input
								type="radio"
								class="radio radio-primary radio-xs"
								{name}
								value={option.value}
								checked={sort === option.value}
								onchange={chooseSort}
							/>
							{option.label}
						</label>
					</li>
				{/each}
			</ul>
		</details>
	{/if}

	<label class="join-item btn btn-square btn-secondary swap swap-rotate hover-lift shadow-none">
		<input name="reversed" type="checkbox" bind:checked={reversed} aria-label="Reverse sort order" />

		<span class="swap-on" aria-hidden="true">
			↑
		</span>

		<span class="swap-off" aria-hidden="true">
			↓
		</span>
	</label>
</div>
