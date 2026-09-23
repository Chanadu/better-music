<script lang="ts" generics="T extends string">
	let {
		options,
		sort = $bindable(),
		reversed = $bindable(false),
		name,
	}: {
		options: readonly { label: string; value: T }[];
		sort: T;
		reversed?: boolean;
		name: string;
	} = $props();
	function chooseSort(event: Event) {
		sort = (event.currentTarget as HTMLInputElement).value as T;
		(event.currentTarget as HTMLElement).closest('details')?.removeAttribute('open');
	}
</script>

<div class="join">
	<details class="dropdown">
		<summary
			class="btn btn-outline btn-secondary join-item justify-between"
			style="width: calc(6ch + 4rem); min-width: calc(6ch + 4rem);"
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

	<label class="join-item btn btn-square btn-outline btn-secondary swap swap-rotate shrink-0">
		<input type="checkbox" bind:checked={reversed} aria-label="Reverse sort order" />
		<span class="swap-on" aria-hidden="true">↑</span>
		<span class="swap-off" aria-hidden="true">↓</span>
	</label>
</div>
