<script lang="ts">
	import { useSortPreference } from '$lib/scripts/sort-preferences.svelte';
	import SearchBar from '../common/SearchBar.svelte';
	import AlbumGrid from './AlbumGrid.svelte';
	import CreateFab from '../create/CreateFab.svelte';
	let { mode }: { mode: 'listened' | 'unlistened' } = $props();
	let query = $state('');

	const options = [
		{ label: 'Album', value: 'album' },
		{ label: 'Artist', value: 'artist' },
		{ label: 'Added', value: 'added' },
	] as const;
	let sorting = useSortPreference(
		() => `bettermusic:sort:${mode === 'listened' ? 'albums' : 'listen'}`,
		options.map((option) => option.value),
		{ sort: 'added', reversed: false },
	);

	function chooseSort(event: Event) {
		sorting.sort = (event.currentTarget as HTMLInputElement).value as typeof sorting.sort;
		(event.currentTarget as HTMLElement).closest('details')?.removeAttribute('open');
	}
</script>

<div class="navbar flex gap-2">
	<SearchBar placeholder="album name..." bind:value={query} />

	<div class="join pt-3">
		<details class="dropdown">
			<summary
				class="btn btn-outline btn-secondary join-item justify-between"
				style="width: calc(6ch + 4rem); min-width: calc(6ch + 4rem);"
			>
				{options.find((option) => option.value === sorting.sort)?.label}
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
								name={`album-sort-${mode}`}
								value={option.value}
								checked={sorting.sort === option.value}
								onchange={chooseSort}
							/>
							{option.label}
						</label>
					</li>
				{/each}
			</ul>
		</details>

		<label class="join-item btn btn-square btn-outline btn-secondary swap swap-rotate shrink-0">
			<input type="checkbox" bind:checked={sorting.reversed} />
			<span class="swap-on">↑</span>
			<span class="swap-off">↓</span>
		</label>
	</div>
</div>

<AlbumGrid {mode} {query} sort={sorting.sort} reversed={sorting.reversed} />

<CreateFab type="album" />
