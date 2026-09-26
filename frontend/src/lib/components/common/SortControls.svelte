<script lang="ts" generics="T extends string">
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
		(event.currentTarget as HTMLElement).closest('details')?.removeAttribute('open');
	}
</script>

<div class="join relative has-[details[open]]:z-20" class:w-full={fullWidth}>
	{#if appSettings.values.useNativeDropdowns}
		<select
			class="select select-secondary text-secondary join-item"
			class:flex-1={fullWidth}
			class:min-w-0={fullWidth}
			style:width={fullWidth ? '100%' : `calc(${labelWidth} + 4rem)`}
			style:min-width={fullWidth ? '0' : `calc(${labelWidth} + 4rem)`}
			aria-label={name}
			bind:value={sort}
		>
			{#each options as option}
				<option value={option.value}>{option.label}</option>
			{/each}
		</select>
	{:else}
		<details class="dropdown" class:flex-1={fullWidth} class:min-w-0={fullWidth}>
			<summary
				class="btn btn-outline btn-secondary join-item justify-between"
				style:width={fullWidth ? '100%' : `calc(${labelWidth} + 4rem)`}
				style:min-width={fullWidth ? '0' : `calc(${labelWidth} + 4rem)`}
			>
				{options.find((option) => option.value === sort)?.label}
			</summary>

			<ul
				class="dropdown-content menu bg-base-100 border-secondary text-secondary rounded-box z-10 mt-2 w-full border-2 shadow"
			>
				{#each options as option}
					<li>
						<label>
							<input
								type="radio"
								class="radio radio-secondary radio-xs"
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

	<label class="join-item btn btn-square btn-outline btn-secondary swap swap-rotate shrink-0">
		<input type="checkbox" bind:checked={reversed} aria-label="Reverse sort order" />
		<span class="swap-on" aria-hidden="true">↑</span>
		<span class="swap-off" aria-hidden="true">↓</span>
	</label>
</div>
