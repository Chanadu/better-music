<script lang="ts">
	import SearchBar from '../common/SearchBar.svelte';
	import MediaThumbnail from '../common/MediaThumbnail.svelte';
	import { spotifyApi } from '$lib/scripts/api';
	import type { SpotifyRow as Row } from '$lib/scripts/types';

	let { type, selected = $bindable(undefined) }: { type: 'artist' | 'album'; selected?: Row } = $props();

	let query = $state('');
	let rows = $state<Row[]>([]);
	let loading = $state(false);
	let message = $state('Start searching...');
	let timer: ReturnType<typeof setTimeout>;
	let count = $derived(type === 'artist' ? 5 : 3);

	async function search() {
		clearTimeout(timer);

		const current = query.trim();
		if (!current) {
			rows = [];
			message = 'Start searching...';
			return;
		}

		loading = true;
		message = '';

		try {
			if (type === 'artist')
				rows = (await spotifyApi.searchArtists(current, count)).map((a) => ({
					id: a.id,
					name: a.name,
					imageUrl: a.images[0]?.url,
				}));
			else
				rows = (await spotifyApi.searchAlbums(current, count)).map((a) => ({
					id: a.id,
					name: a.name,
					imageUrl: a.images[0]?.url,
					artistId: a.artists[0]?.id,
					artistName: a.artists[0]?.name,
					releaseYear: a.release_date.split('-')[0],
					meta: [a.artists.map((artist) => artist.name).join(', '), a.release_date.split('-')[0]]
						.filter(Boolean)
						.join(' • '),
				}));

			message = rows.length ? '' : `No ${type}s found.`;
		} catch {
			rows = [];
			message = 'Spotify search failed.';
		} finally {
			loading = false;
		}
	}

	function changed() {
		clearTimeout(timer);
		timer = setTimeout(search, 250);
	}

	export function reset() {
		clearTimeout(timer);
		query = '';
		rows = [];
		selected = undefined;
		message = 'Start searching...';
	}

	let displayRows = $derived.by(() => {
		let visible = rows;
		const selection = selected;

		if (selection && !rows.some((row) => row.id === selection.id)) {
			visible = [...rows.slice(0, count - 1), selection];
		}

		return Array.from({ length: count }, (_, i) => visible[i]);
	});
</script>

<div role="tabpanel" class="tab-content px-2 pt-4">
	<div oninput={changed}>
		<SearchBar placeholder={`${type} name...`} bind:value={query} />
	</div>

	<ul class="list bg-base-200 rounded-box mt-4 shadow-md" aria-busy={loading}>
		{#each displayRows as row}
			<li class="p-0">
				<label class={row ? 'block cursor-pointer' : 'block cursor-default'}>
					<input
						type="radio"
						name={`spotify_${type}`}
						class="peer sr-only"
						value={row?.id ?? ''}
						checked={!!row && selected?.id === row.id}
						disabled={!row || loading}
						onclick={(event) => {
							if (row && selected?.id === row.id) {
								event.preventDefault();
								selected = undefined;
							}
						}}
						onchange={(event) => {
							if (row && event.currentTarget.checked) selected = row;
						}}
					/>

					<div
						class="list-row hover:bg-base-300/50 peer-checked:bg-primary peer-checked:text-primary-content peer-checked:hover:bg-primary peer-checked:hover:text-primary-content items-center"
					>
						<MediaThumbnail
							variant="result"
							imageUrl={loading ? '' : (row?.imageUrl ?? '')}
							label={loading ? '' : (row?.name ?? '')}
							alt={row ? `${row.name} ${type} image` : ''}
							fallbackText={loading || !row ? '' : undefined}
						/>

						<div class="min-w-0">
							{#if loading}
								<span class="loading loading-dots loading-md"></span>
							{:else}
								<div class={`truncate ${!row ? 'text-base-content/35' : ''}`}>
									{row?.name ?? message}
								</div>

								<div class={`truncate text-xs ${!row ? 'text-base-content/35' : ''}`}>
									{row?.meta ?? ''}
								</div>
							{/if}
						</div>
					</div>
				</label>
			</li>
		{/each}
	</ul>
</div>
