<script lang="ts">
	import SearchBar from '../common/SearchBar.svelte';
	import SpotifySearchResult from './SpotifySearchResult.svelte';
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
			<SpotifySearchResult {row} {type} {loading} {message} bind:selected />
		{/each}
	</ul>
</div>
