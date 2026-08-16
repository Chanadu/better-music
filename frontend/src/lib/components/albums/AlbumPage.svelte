<script lang="ts">
	import SearchBar from '../common/SearchBar.svelte';
	import AlbumGrid from './AlbumGrid.svelte';
	import CreateFab from '../create/CreateFab.svelte';
	let { mode }: { mode: 'listened' | 'unlistened' } = $props();
	let query = $state('');
	let sort = $state<'album' | 'artist' | 'added'>('added');
	let reversed = $state(false);

	const options = [
		{ label: 'Album', value: 'album' },
		{ label: 'Artist', value: 'artist' },
		{ label: 'Added', value: 'added' },
	] as const;

	function chooseSort(event: Event) {
		sort = (event.currentTarget as HTMLInputElement).value as typeof sort;
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
								name={`album-sort-${mode}`}
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
			<input type="checkbox" bind:checked={reversed} />
			<span class="swap-on">↑</span>
			<span class="swap-off">↓</span>
		</label>
	</div>
</div>

<AlbumGrid {mode} {query} {sort} {reversed} />

<CreateFab type="album" />
