<script lang="ts">
	import MediaThumbnail from '../common/MediaThumbnail.svelte';
	import SpotifySearch from './SpotifySearch.svelte';
	import FloatingField from '../common/FloatingField.svelte';
	import { albumsApi, artistsApi, spotifyApi } from '$lib/scripts/api';
	import { fetchDatabaseData, refreshDatabaseData } from '$lib/scripts/database';
	import type { Artist, SpotifyRow as Row } from '$lib/scripts/types';

	let {
		type,
		dialog = $bindable(),
		onclose,
	}: { type: 'artist' | 'album'; dialog?: HTMLDialogElement; onclose?: () => void } = $props();

	let tab = $state<'manual' | 'spotify'>('manual');
	let name = $state('');
	let selected = $state<Row | undefined>();
	let error = $state('');
	let saving = $state(false);
	let spotify: SpotifySearch;
	let artists = $state<Artist[]>([]);
	let artistId = $state('');
	let year = $state('');
	let comment = $state('');
	let listened = $state(false);
	let listenedAt = $state('');
	let rating = $state(5);
	let currentYear = new Date().getFullYear();
	let entityLabel = $derived(type === 'artist' ? 'Artist' : 'Album');

	function isDigit(character: string) {
		return character >= '0' && character <= '9';
	}

	let yearValid = $derived(
		!year || (year.length === 4 && [...year].every(isDigit) && Number(year) >= 1000 && Number(year) <= currentYear),
	);
	let canSave = $derived.by(() => {
		if (saving) return false;
		if (tab === 'spotify') return Boolean(selected);
		return Boolean(name.trim()) && (type === 'artist' || Boolean(artistId)) && yearValid;
	});

	$effect(() => {
		if (type === 'album') {
			fetchDatabaseData()
				.then((data) => (artists = data.artists))
				.catch((e) => (error = formatError(e, 'Failed to load artists')));
		}
	});

	function formatError(value: unknown, fallback: string) {
		const text = value instanceof Error ? value.message : fallback;
		return text.charAt(0).toUpperCase() + text.slice(1);
	}

	function today() {
		const date = new Date();
		return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`;
	}

	function toggleListened() {
		if (listened && !listenedAt) listenedAt = today();
	}

	function reset() {
		tab = 'manual';
		name = '';
		selected = undefined;
		error = '';
		artistId = '';
		year = '';
		comment = '';
		listened = false;
		listenedAt = '';
		rating = 5;
		spotify?.reset();
	}

	function cleanYear(event: Event) {
		year = [...(event.currentTarget as HTMLInputElement).value].filter(isDigit).join('');
	}

	function getAlbumYear() {
		if (tab === 'spotify' && selected?.releaseYear) {
			return Number(selected.releaseYear);
		}

		if (year) return Number(year);
		return undefined;
	}

	async function findOrCreateArtist(row: Row) {
		const existing = artists.find(
			(artist) =>
				(row.artistId && artist.spotify_id === row.artistId) ||
				artist.name.trim().toLowerCase() === (row.artistName ?? '').trim().toLowerCase(),
		);
		if (existing) return existing;

		let cover_url: string | undefined;
		if (row.artistId && row.artistName) {
			try {
				cover_url = (await spotifyApi.searchArtists(row.artistName, 10)).find(
					(artist) => artist.id === row.artistId,
				)?.images[0]?.url;
			} catch {}
		}

		const created = await artistsApi.create({
			name: row.artistName ?? '',
			spotify_id: row.artistId,
			cover_url,
		});
		artists = [...artists, created];
		return created;
	}

	async function save() {
		if (!canSave) return;

		saving = true;
		error = '';

		try {
			if (type === 'artist') {
				if (tab === 'manual') {
					await artistsApi.create({ name: name.trim() });
				} else {
					await artistsApi.create({
						name: selected!.name,
						cover_url: selected!.imageUrl,
						spotify_id: selected!.id,
					});
				}
			} else {
				const chosenArtist = tab === 'spotify' ? await findOrCreateArtist(selected!) : undefined;
				const id = chosenArtist?.id ?? Number(artistId);
				const title = tab === 'spotify' ? selected!.name : name.trim();
				const album = await albumsApi.create({
					artist_id: id,
					title,
					spotify_id: tab === 'spotify' ? selected!.id : undefined,
				});

				const metadata = {
					artist_id: id,
					cover_url: tab === 'spotify' ? selected!.imageUrl : undefined,
					year: getAlbumYear(),
					listened: listened || undefined,
					rating: listened ? rating : undefined,
					comment: comment.trim() || undefined,
					listened_at: listened ? listenedAt || undefined : undefined,
				};

				if (Object.values(metadata).some((value, index) => index > 0 && value !== undefined)) {
					await albumsApi.update(album.id, metadata);
				}
			}

			await refreshDatabaseData();
			dialog?.close();
			reset();
		} catch (e) {
			console.error(`Failed to save ${type}`, e);
			error = formatError(e, `Failed to save ${type}`);
		} finally {
			saving = false;
		}
	}
</script>

<dialog class="modal" bind:this={dialog} {onclose}>
	<div class="modal-box">
		<form method="dialog">
			<button class="btn btn-sm btn-circle btn-ghost absolute top-2 right-2">✕</button>
		</form>

		<h2 class="mb-2 text-center text-xl font-semibold">
			Create {entityLabel}
		</h2>

		<div role="tablist" class="tabs tabs-border flex">
			<input
				type="radio"
				name={`${type}_modal_tabs`}
				role="tab"
				class="tab flex-1"
				aria-label="Manual"
				value="manual"
				bind:group={tab}
			/>
			<div role="tabpanel" class="tab-content px-2 pt-4">
				<div class="flex flex-col gap-3">
					{#if type === 'album'}
						<FloatingField label="Artist">
							<select class="select w-full" bind:value={artistId}>
								<option value="">Select artist</option>
								{#each artists as artist}
									<option value={artist.id}>
										{artist.name}
									</option>
								{/each}
							</select>
						</FloatingField>

						<div class="flex items-center gap-4">
							<div class="basis-1/3 pt-3">
								<MediaThumbnail
									variant="album-preview"
									label={name}
									alt={name ? `${name} album preview` : ''}
									emptyFallback=""
								/>
							</div>

							<div class="flex min-w-0 flex-1 flex-col gap-3">
								<FloatingField label="Album Title">
									<input
										type="text"
										class="input w-full"
										placeholder="e.g. 2014 Forest Hills Drive"
										bind:value={name}
									/>
								</FloatingField>

								<FloatingField label="Year">
									<input
										type="text"
										inputmode="numeric"
										pattern="[0-9]{4}"
										maxlength="4"
										class="input w-full"
										class:validator={!!year}
										placeholder="2014"
										value={year}
										oninput={cleanYear}
									/>
									<div class="validator-hint hidden">Enter a valid year</div>
								</FloatingField>
							</div>
						</div>

						<FloatingField label="Comment">
							<textarea class="textarea w-full" placeholder="Classic album" bind:value={comment}
							></textarea>
						</FloatingField>
					{:else}
						<FloatingField label="Artist Name">
							<input type="text" class="input w-full" placeholder="e.g. J. Cole" bind:value={name} />
						</FloatingField>

						<div class="flex justify-center">
							<MediaThumbnail
								variant="artist-preview"
								label={name}
								alt={name ? `${name} artist preview` : ''}
								emptyFallback=""
							/>
						</div>
					{/if}
				</div>
			</div>

			<input
				type="radio"
				name={`${type}_modal_tabs`}
				role="tab"
				class="tab flex-1"
				aria-label="Spotify"
				value="spotify"
				bind:group={tab}
			/>

			<SpotifySearch bind:this={spotify} {type} bind:selected />
		</div>

		{#if type === 'album'}
			<div class="px-2 pt-4">
				<label class="group divider mb-1 cursor-pointer">
					<span class="flex items-center gap-3">
						<span class="toggle has-checked:bg-primary has-checked:text-primary-content">
							<input type="checkbox" bind:checked={listened} onchange={toggleListened} />

							<svg
								aria-label="disabled"
								viewBox="0 0 24 24"
								fill="none"
								stroke="currentColor"
								stroke-width="4"
								stroke-linecap="round"
								stroke-linejoin="round"
							>
								<path d="M18 6 6 18"></path>
								<path d="m6 6 12 12"></path>
							</svg>

							<svg aria-label="enabled" class="text-primary" viewBox="0 0 24 24">
								<g
									stroke-linejoin="round"
									stroke-linecap="round"
									stroke-width="4"
									fill="none"
									stroke="currentColor"
								>
									<path d="M20 6 9 17l-5-5"></path>
								</g>
							</svg>
						</span>

						<span class="text-neutral-content group-has-checked:text-primary"> Listened </span>
					</span>
				</label>

				<div class="flex flex-col gap-2" class:opacity-40={!listened}>
					<fieldset class="fieldset">
						<legend class="fieldset-legend">Rating</legend>

						<div class="w-full">
							<input
								type="range"
								min="1"
								max="10"
								step="1"
								class="range range-accent range-lg w-full"
								disabled={!listened}
								bind:value={rating}
							/>

							<div class="px-4">
								<div class="relative mt-1 h-4 text-center text-xs">
									{#each Array.from({ length: 10 }, (_, i) => i + 1) as value}
										<span
											class="absolute -translate-x-1/2"
											class:text-accent={value === rating}
											style={`left: ${((value - 1) / 9) * 100}%`}
										>
											{value === rating ? '↓' : '|'}
										</span>
									{/each}
								</div>
							</div>

							<div class="px-4">
								<div class="relative h-4 text-center text-xs">
									{#each Array.from({ length: 10 }, (_, i) => i + 1) as value}
										<span
											class="absolute -translate-x-1/2"
											class:text-accent={value === rating}
											class:text-xl={value === rating}
											style={`left: ${((value - 1) / 9) * 100}%`}
										>
											{value}
										</span>
									{/each}
								</div>
							</div>
						</div>
					</fieldset>

					<FloatingField label="Listened At">
						<input type="date" class="input w-full" disabled={!listened} bind:value={listenedAt} />
					</FloatingField>
				</div>
			</div>
		{/if}

		{#if error}
			<div class="alert alert-error alert-soft mt-4 justify-center text-center" role="alert">
				<span>{error}</span>
			</div>
		{/if}

		<div class="modal-action mt-6 flex gap-4 px-2">
			<form method="dialog" class="flex-1">
				<button class="btn btn-soft btn-secondary w-full">Cancel</button>
			</form>

			<button type="button" class="btn btn-primary flex-1" disabled={!canSave} onclick={save}>
				{saving ? 'Saving...' : `Save ${entityLabel}`}
			</button>
		</div>
	</div>

	<form method="dialog" class="modal-backdrop"><button>close</button></form>
</dialog>
