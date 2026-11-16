<script lang="ts">
	import { Button, Input, Range, Textarea } from 'flowbite-svelte';
	import { onMount } from 'svelte';
	import {
		ArrowDownUp,
		ArrowLeft,
		ArrowUpRight,
		ChevronRight,
		Disc3,
		Headphones,
		Plus,
		Search,
		Star,
		UsersRound,
	} from '@lucide/svelte';

	const artists = [
		{ name: 'North Atlas', added: '2026-11-12', initials: 'NA', color: '#dfcbbb', albums: 7, score: 9.2 },
		{ name: 'Aster Vale', added: '2026-11-08', initials: 'AV', color: '#dcd5e7', albums: 4, score: 8.7 },
		{ name: 'June Static', added: '2026-10-28', initials: 'JS', color: '#e8c4ad', albums: 3, score: 8.3 },
		{ name: 'Soft Signal', added: '2026-10-19', initials: 'SS', color: '#d1dac5', albums: 6, score: 8.1 },
		{ name: 'Milo Finch', added: '2026-09-24', initials: 'MF', color: '#e9d6a8', albums: 2, score: 7.9 },
		{ name: 'The Field Notes', added: '2026-09-16', initials: 'FN', color: '#d5dce0', albums: 5, score: 7.6 },
	];
	const addedDate = new Intl.DateTimeFormat('en-US', { dateStyle: 'medium', timeZone: 'UTC' });
	let screen = $state<'artists' | 'album'>('artists');
	type PreviewTabs = { postMessage(index: number): void };
	type PreviewWindow = Window & {
		webkit?: {
			messageHandlers?: { betterMusicPreviewTabs?: PreviewTabs; betterMusicLivePreviewTabs?: PreviewTabs };
		};
	};
	let nativeTabs = $state.raw<PreviewTabs>();
	let livePreview = $state(false);
	let query = $state('');
	let alphabetical = $state(false);
	let saved = $state(true);
	let notes = $state('A slow burn. The second half feels like stepping outside after a storm.');
	let editing = $state(false);
	let rating = $state(9);
	const matches = $derived(
		artists
			.filter((artist) => artist.name.toLowerCase().includes(query.trim().toLowerCase()))
			.sort((a, b) => (alphabetical ? a.name.localeCompare(b.name) : b.score - a.score)),
	);

	function show(next: 'artists' | 'album') {
		screen = next;
		window.scrollTo({ top: 0 });
	}

	onMount(() => {
		const handlers = (window as PreviewWindow).webkit?.messageHandlers;
		const bridge = handlers?.betterMusicLivePreviewTabs ?? handlers?.betterMusicPreviewTabs;
		if (!bridge) return;

		livePreview = !!handlers?.betterMusicLivePreviewTabs;
		nativeTabs = bridge;
		const sync = () => bridge.postMessage(screen === 'artists' ? 0 : 1);
		const hide = () => bridge.postMessage(-1);
		const select = (event: Event) => {
			const index = (event as CustomEvent<{ index: number }>).detail.index;
			if (index === 0) show('artists');
			if (index === 1) show('album');
		};
		window.addEventListener('nativeTabSelected', select);
		window.addEventListener('pageshow', sync);
		window.addEventListener('pagehide', hide);
		return () => {
			window.removeEventListener('nativeTabSelected', select);
			window.removeEventListener('pageshow', sync);
			window.removeEventListener('pagehide', hide);
			hide();
		};
	});

	$effect(() => {
		nativeTabs?.postMessage(screen === 'artists' ? 0 : 1);
	});
</script>

<svelte:head>
	<title>Better Music · Redesign preview</title>
	<meta name="robots" content="noindex" />
</svelte:head>

{#snippet dateAdded(date: string)}
	<time datetime={date}>{addedDate.format(new Date(date))}</time>
{/snippet}

<div class="design dark" class:native-tabs={!!nativeTabs}>
	<aside>
		<div class="brand"><span class="brand-mark"><Disc3 size={23} /></span> Better Music</div>
		<nav aria-label="Preview screens">
			<button
				class:active={screen === 'artists'}
				aria-current={screen === 'artists' ? 'page' : undefined}
				onclick={() => show('artists')}><UsersRound size={18} /> Artists</button
			>
			<button
				class:active={screen === 'album'}
				aria-current={screen === 'album' ? 'page' : undefined}
				onclick={() => show('album')}><Disc3 size={18} /> Album detail</button
			>
		</nav>
		{#if !livePreview}
			<a class="exit" href="/preview">Component preview <ArrowUpRight size={15} /></a>
		{/if}
	</aside>
	<main>
		{#if screen === 'artists'}
			<header class="page-title">
				<h1>Artists</h1>
				<p>{artists.length} artists in your library</p>
			</header>
			<div class="mb-4 flex items-center gap-2 sm:gap-3">
				<div class="min-w-0 flex-1">
					<Input
						type="search"
						aria-label="Search artists"
						placeholder="Search artists"
						bind:value={query}
						class="ps-9"
					>
						{#snippet left()}<Search size={18} />{/snippet}
					</Input>
				</div>
				<Button
					color="alternative"
					class="gap-2 whitespace-nowrap"
					onclick={() => (alphabetical = !alphabetical)}
				>
					<ArrowDownUp size={16} />
					{alphabetical ? 'Name' : 'Rating'}
				</Button>
			</div>
			<div
				class="hidden grid-cols-[minmax(0,1fr)_3.5rem_5rem_6.5rem_2rem] items-center gap-2.5 py-2.5 text-xs text-(--preview-muted) min-[681px]:grid"
			>
				<span>Artist</span><span>Albums</span><span>Avg. rating</span><span>Added</span>
			</div>
			{#each matches as artist (artist.name)}
				<div
					class="grid min-h-[54px] grid-cols-[minmax(0,1fr)_auto_43px_24px] items-center gap-[7px] border-t border-(--preview-border) py-[9px] min-[681px]:grid-cols-[minmax(0,1fr)_3.5rem_5rem_6.5rem_2rem] min-[681px]:gap-2.5 min-[681px]:py-2.5"
				>
					<div class="flex min-w-0 items-center gap-2 max-[680px]:text-sm min-[681px]:gap-3">
						<span
							class="grid size-[35px] shrink-0 place-items-center rounded-[10px] text-[13px] font-[750] text-[#172234] min-[681px]:size-[34px]"
							style:background={artist.color}>{artist.initials}</span
						>
						<span>
							<strong class="block">{artist.name}</strong>
							<small class="mt-0.5 block text-xs text-(--preview-muted) min-[681px]:hidden"
								>Added {@render dateAdded(artist.added)}</small
							>
						</span>
					</div>
					<span class="max-[680px]:text-xs"
						>{artist.albums}<span class="min-[681px]:hidden"> albums</span></span
					>
					<span class="flex items-center gap-[5px] tabular-nums max-[680px]:text-xs"
						><Star size={14} class="text-(--color-warning)" /> {artist.score.toFixed(1)}</span
					>
					<span class="hidden text-xs text-(--preview-muted) min-[681px]:block"
						>{@render dateAdded(artist.added)}</span
					>
					{#if artist.name === 'North Atlas'}
						<button
							class="grid min-h-10 place-items-center rounded-lg hover:bg-(--preview-active)"
							aria-label="Open North Atlas sample album"
							onclick={() => show('album')}><ChevronRight size={19} /></button
						>
					{/if}
				</div>
			{:else}
				<p class="empty">No artists match “{query}”.</p>
			{/each}
		{:else}
			<button class="back" onclick={() => show('artists')}><ArrowLeft size={17} /> Artists</button>
			<div class="album">
				<img
					src="/redesign-preview/valley.webp"
					alt="Sunlit mountains and a lake on the sample album cover"
					width="720"
					height="720"
				/>
				<div class="album-info">
					<p class="artist-name">North Atlas</p>
					<h1>Between<br class="desktop-break" /> the Pines</h1>
					<p class="metadata">2021 <span>·</span> Ambient folk</p>
					<div class="album-actions">
						<Button color="primary" class="gap-2" aria-pressed={saved} onclick={() => (saved = !saved)}
							>{#if saved}<Headphones size={18} /> Listened{:else}<Plus size={18} /> Mark listened{/if}</Button
						>
					</div>
				</div>
			</div>
			<div class="journal">
				<div class="album-score">
					<div class="rating-heading">
						<label for="preview-rating">Your rating</label>
						<output for="preview-rating">{rating}<small> / 10</small></output>
					</div>
					<Range
						id="preview-rating"
						min={1}
						max={10}
						step={1}
						appearance="auto"
						bind:value={rating}
						class="mt-2"
					/>
				</div>
				<div class="notes">
					<div class="note-title">
						<h2>Listening notes</h2>
						<button onclick={() => (editing = !editing)}>{editing ? 'Done' : 'Edit'}</button>
					</div>
					{#if editing}<Textarea
							aria-label="Listening notes"
							bind:value={notes}
							rows={3}
							class="mt-2"
						/>{:else}<p>
							{notes || 'No notes yet.'}
						</p>{/if}
				</div>
			</div>
		{/if}
	</main>
</div>

<style>
	.design {
		--preview-muted: color-mix(in srgb, var(--color-base-content) 78%, var(--color-base-200));
		--preview-border: color-mix(in srgb, var(--color-base-content) 16%, var(--color-base-200));
		--preview-active: color-mix(in srgb, var(--color-primary) 16%, var(--color-base-300));
		min-height: 100dvh;
		background: var(--color-base-100);
		color: var(--color-base-content);
		display: grid;
		grid-template-columns: 232px minmax(0, 1fr);
		font-family: 'Instrument Sans', system-ui, sans-serif;
		font-weight: 500;
	}
	.design button,
	.exit {
		cursor: pointer;
	}
	aside {
		position: sticky;
		top: 0;
		height: 100dvh;
		display: flex;
		flex-direction: column;
		padding: 34px 21px 25px;
		background: var(--color-base-300);
		color: var(--color-base-content);
	}
	.brand {
		display: flex;
		gap: 11px;
		align-items: center;
		font-size: 19px;
		font-weight: 750;
		letter-spacing: -0.04em;
	}
	.brand-mark {
		display: grid;
		place-items: center;
		background: var(--color-primary);
		color: var(--color-primary-content);
		width: 36px;
		height: 36px;
		border-radius: 50%;
	}
	nav {
		display: grid;
		gap: 7px;
		margin-top: 58px;
	}
	.native-tabs nav {
		display: none;
	}
	nav button {
		display: flex;
		align-items: center;
		gap: 12px;
		border-radius: 9px;
		padding: 12px;
		color: var(--preview-muted);
		text-align: left;
		font-size: 14px;
	}
	nav button:hover,
	nav button.active {
		background: var(--preview-active);
		color: var(--color-base-content);
	}
	nav button.active {
		color: var(--color-primary);
	}
	.exit {
		margin-top: auto;
		display: flex;
		align-items: center;
		gap: 6px;
		font-size: 13px;
		color: var(--preview-muted);
	}
	main {
		width: 100%;
		max-width: 1100px;
		margin-inline: auto;
		padding: 38px clamp(28px, 5vw, 80px) 68px;
	}
	.metadata span {
		margin-inline: 6px;
	}
	.page-title {
		margin: 0 0 29px;
	}
	h1 {
		font-family: 'Instrument Serif', Georgia, serif;
		font-weight: 400;
		font-size: clamp(58px, 8vw, 100px);
		letter-spacing: -0.065em;
		line-height: 0.92;
	}
	.page-title p,
	.metadata {
		margin-top: 14px;
		color: var(--preview-muted);
	}

	.empty {
		padding-block: 22px;
		color: var(--preview-muted);
	}
	.back {
		display: flex;
		align-items: center;
		gap: 8px;
		margin: 0 0 18px;
		color: var(--color-primary);
	}
	.album {
		display: grid;
		grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
		align-items: center;
		gap: clamp(22px, 4vw, 55px);
	}
	.album img {
		display: block;
		width: 100%;
		height: auto;
		border: 1px solid var(--preview-border);
		aspect-ratio: 1;
		object-fit: cover;
		border-radius: 15px;
		background: var(--color-base-300);
	}
	.artist-name {
		margin-bottom: 11px;
		color: var(--color-primary);
		font-size: 15px;
	}
	.album h1 {
		font-size: clamp(48px, 6.3vw, 86px);
	}
	.album-actions {
		margin-top: 24px;
	}
	.journal {
		display: grid;
		grid-template-columns: 1fr 1.65fr;
		gap: 28px;
		margin-top: 28px;
		border-top: 1px solid var(--preview-border);
		padding-top: 20px;
	}
	.album-score,
	.notes {
		min-width: 0;
	}
	.rating-heading,
	.note-title {
		display: flex;
		justify-content: space-between;
		gap: 10px;
		align-items: center;
		font-size: 14px;
	}
	.album-score output {
		font-size: 25px;
		font-weight: 700;
	}
	.album-score small {
		color: var(--preview-muted);
		font-size: 12px;
	}
	.notes p {
		margin-top: 10px;
		color: var(--preview-muted);
		font-size: 14px;
	}
	.note-title button {
		color: var(--color-primary);
		text-decoration: underline;
	}
	@media (max-width: 680px) {
		.design {
			display: block;
		}
		aside {
			display: contents;
		}
		.brand {
			display: none;
		}
		nav {
			position: fixed;
			z-index: 10;
			inset: auto 0 0;
			display: grid;
			grid-template-columns: repeat(2, minmax(0, 1fr));
			gap: 8px;
			margin: 0;
			background: var(--color-base-300);
			border-top: 1px solid var(--preview-border);
			padding: 8px max(16px, env(safe-area-inset-right)) calc(8px + env(safe-area-inset-bottom))
				max(16px, env(safe-area-inset-left));
		}
		nav button {
			flex-direction: column;
			justify-content: center;
			min-height: 52px;
			padding: 6px 10px;
			gap: 4px;
			font-size: 12px;
		}
		.exit {
			display: none;
		}
		main {
			padding: calc(20px + env(safe-area-inset-top)) 16px calc(92px + env(safe-area-inset-bottom));
		}
		.native-tabs main {
			padding-bottom: 20px;
		}
		.page-title {
			margin-bottom: 22px;
		}
		.page-title h1 {
			font-size: 42px;
		}
		.page-title p {
			margin-top: 8px;
		}
		.album {
			grid-template-columns: minmax(0, 1fr);
			gap: 18px;
		}
		.album img {
			max-width: 360px;
		}
		.desktop-break {
			display: none;
		}
		.album-actions {
			margin-top: 14px;
		}
		.journal {
			grid-template-columns: minmax(0, 1fr);
			gap: 18px;
		}
		.notes {
			border-top: 1px solid var(--preview-border);
			padding-top: 16px;
		}
	}
</style>
