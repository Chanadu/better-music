<script lang="ts">
	import type { Component } from 'svelte';

	import Specimen from '$lib/components/preview/Specimen.svelte';
	import NextAlbum from '$lib/components/albums/NextAlbum.svelte';
	import AlbumCard from '$lib/components/albums/AlbumCard.svelte';
	import AlbumActions from '$lib/components/albums/AlbumActions.svelte';
	import AlbumStats from '$lib/components/albums/AlbumStats.svelte';
	import AlbumNotes from '$lib/components/albums/AlbumNotes.svelte';
	import AlbumArtistCard from '$lib/components/albums/AlbumArtistCard.svelte';
	import AddAlbumCard from '$lib/components/albums/AddAlbumCard.svelte';
	import EditAlbumModal from '$lib/components/albums/EditAlbumModal.svelte';
	import ArtistListItem from '$lib/components/artists/ArtistListItem.svelte';
	import EditArtistModal from '$lib/components/artists/EditArtistModal.svelte';
	import MediaHero from '$lib/components/common/MediaHero.svelte';
	import MediaThumbnail from '$lib/components/common/MediaThumbnail.svelte';
	import RecordSyncBadge from '$lib/components/common/RecordSyncBadge.svelte';
	import StatCard from '$lib/components/common/StatCard.svelte';
	import SearchBar from '$lib/components/common/SearchBar.svelte';
	import SortControls from '$lib/components/common/SortControls.svelte';
	import YearField from '$lib/components/common/YearField.svelte';
	import FloatingField from '$lib/components/common/FloatingField.svelte';
	import DeleteConfirmationDialog from '$lib/components/common/DeleteConfirmationDialog.svelte';
	import FormModalShell from '$lib/components/create/FormModalShell.svelte';
	import ManualAlbumForm from '$lib/components/create/ManualAlbumForm.svelte';
	import ManualArtistForm from '$lib/components/create/ManualArtistForm.svelte';
	import ListenedFields from '$lib/components/create/ListenedFields.svelte';
	import RatingInput from '$lib/components/create/RatingInput.svelte';
	import SpotifySearchResult from '$lib/components/create/SpotifySearchResult.svelte';
	import AuthForm from '$lib/components/auth/AuthForm.svelte';
	import PasswordField from '$lib/components/auth/PasswordField.svelte';
	import SettingRow from '$lib/components/more/SettingRow.svelte';
	import AccountSettingRow from '$lib/components/more/AccountSettingRow.svelte';
	import AccountActionDialog from '$lib/components/more/AccountActionDialog.svelte';
	import RatingSettingsDialog from '$lib/components/more/RatingSettingsDialog.svelte';
	import ThemePickerDialog from '$lib/components/more/ThemePickerDialog.svelte';
	import ExportLibrarySetting from '$lib/components/more/ExportLibrarySetting.svelte';
	import ThemePalette from '$lib/components/more/ThemePalette.svelte';
	import ThemeOption from '$lib/components/more/ThemeOption.svelte';
	import RatingDistributionChart from '$lib/components/more/RatingDistributionChart.svelte';
	import StorageErrorAlert from '$lib/components/common/StorageErrorAlert.svelte';
	import SyncQueueList from '$lib/components/more/SyncQueueList.svelte';
	import ConflictMutationModal from '$lib/components/more/ConflictMutationModal.svelte';
	import DiscardMutationModal from '$lib/components/more/DiscardMutationModal.svelte';
	import BottomNav from '$lib/components/navigation/BottomNav.svelte';
	import AlbumIcon from '$lib/components/icons/AlbumIcon.svelte';
	import StarIcon from '$lib/components/icons/StarIcon.svelte';
	import SpotifyIcon from '$lib/components/icons/SpotifyIcon.svelte';
	import { daisyThemes } from '$lib/scripts/app-settings.svelte';
	import { ratingColor, ratingLabel } from '$lib/scripts/rating-colors';
	import { getChangesNeedingAttention } from '$lib/scripts/queue-state';
	import type { RecoveryPlan } from '$lib/scripts/queue-recovery';
	import type { Album, Artist, PendingMutation, RecordSyncStatus, SpotifyRow } from '$lib/scripts/types';
	let theme = $state('bettermusic');
	let sort = $state<'album' | 'artist'>('album');
	const noop = () => {};
	const sections = [
		'Buttons',
		'Forms',
		'Media',
		'Ratings',
		'Navigation',
		'Settings',
		'Online & offline',
		'Dialogs',
		'Feedback',
		'Icons',
	];
	const slug = (s: string) => s.toLowerCase().replace(/[^a-z]+/g, '-');
	const artwork =
		'data:image/svg+xml,' +
		encodeURIComponent(
			'<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 400 400"><rect width="400" height="400" fill="#172234"/><circle cx="200" cy="200" r="155" fill="#9bc0ff"/><circle cx="200" cy="200" r="110" fill="#283142"/><circle cx="200" cy="200" r="65" fill="#fdab8f"/><circle cx="200" cy="200" r="15" fill="#172234"/></svg>',
		);
	const artist: Artist = {
		id: 900001,
		version: 1,
		name: 'Preview artist',
		cover_url: artwork,
		spotify_id: 'preview',
		created_at: '2026-01-02T12:00:00Z',
	};
	const album: Album = {
		id: 900001,
		version: 1,
		artist_id: artist.id,
		title: 'Sample record',
		cover_url: artwork,
		spotify_id: 'preview',
		year: 2024,
		created_at: artist.created_at,
		listened: true,
		listened_at: '2026-01-03T12:00:00Z',
		rating: 8,
		comment: 'A favorite worth another listen.\nNotes preserve line breaks.',
	};
	const row: SpotifyRow = { id: 'sample', name: album.title, meta: 'Preview artist • 2024', imageUrl: artwork };
	const recordStates: RecordSyncStatus[] = [
		'synced',
		'pending',
		'syncing',
		'paused',
		'failed',
		'conflict',
		'blocked',
	];
	const queue: PendingMutation[] = (
		['pending', 'syncing', 'failed', 'conflict', 'blocked', 'unknown', 'retry'] as const
	).map((status, i) => ({
		mutationId: `preview-${i}`,
		userId: 0,
		entity: 'artist',
		operation: 'update',
		entityId: 900010 + i,
		payload: { name: `${status} artist` },
		after: { ...artist, name: `${status} artist` },
		createdAt: Date.parse(artist.created_at),
		attempts: 1,
		status: status === 'unknown' || status === 'retry' ? 'pending' : status,
		outcomeUnknown: status === 'unknown',
		blockedReason: status === 'blocked' ? 'Resolve the earlier change first.' : undefined,
		nextAttemptAt: status === 'retry' ? Date.parse('2099-01-01') : undefined,
		failure:
			status === 'failed' ?
				{ kind: 'invalid-data', message: 'An artist with this name already exists.' }
			:	undefined,
	}));
	const plan: RecoveryPlan = {
		userId: 0,
		session: null,
		generation: 'preview',
		revision: 0,
		mutation: queue[2],
		baseline: artist,
		messages: ['Your local edit will be removed.', 'Later edits will be preserved.'],
		action: 'discard',
	};
	const icons = Object.entries(
		import.meta.glob<{ default: Component<{ class?: string }> }>('$lib/components/icons/*.svelte', { eager: true }),
	).map(([path, module]) => ({ name: path.split('/').pop()!.replace('Icon.svelte', ''), component: module.default }));
	function guard(node: HTMLElement) {
		const prevent = (e: Event) => {
			if (e.type === 'click' && (e.target as HTMLElement).closest('.source-select, .field-option')) return;

			if (e.type === 'submit' || (e.target as HTMLElement).closest('button,a,summary')) {
				e.preventDefault();
				e.stopPropagation();
			}
		};
		node.addEventListener('click', prevent, true);
		node.addEventListener('submit', prevent, true);
		node.querySelectorAll('details').forEach((el) => el.setAttribute('open', ''));
		return {
			destroy() {
				node.removeEventListener('click', prevent, true);
				node.removeEventListener('submit', prevent, true);
			},
		};
	}
</script>

<svelte:head>
	<title>Visual preview · Better Music</title>

	<meta name="robots" content="noindex" />
</svelte:head>

<div data-theme={theme} class="bg-base-100 text-base-content min-h-screen pb-12">
	<header class="border-base-300 border-b px-4 py-8 sm:px-8">
		<div class="mx-auto max-w-7xl">
			<a class="link link-primary" href="/more"> ← Back to Better Music </a>

			<div class="mt-4 flex flex-wrap items-end justify-between gap-4">
				<div>
					<h1 class="text-3xl font-black sm:text-4xl">Visual component preview</h1>

					<p class="text-base-content/60 mt-2 max-w-2xl">
						Distinct components and states, grouped for quick visual checks. Sample data, inline dialogs,
						and simultaneous online and offline variants. Preview actions do not save changes.
					</p>
				</div>

				<label class="fieldset">
					<span class="fieldset-legend"> Preview theme </span>

					<select name="theme" class="select" bind:value={theme}>
						<option value="bettermusic"> Better Music </option>

						{#each daisyThemes as item}
							<option value={item}>
								{item}
							</option>
						{/each}
					</select>
				</label>
			</div>
		</div>
	</header>

	<nav
		aria-label="Preview sections"
		class="bg-base-100/95 border-base-300 sticky top-0 z-30 border-b px-4 py-3 backdrop-blur"
	>
		<div class="mx-auto flex max-w-7xl flex-wrap gap-x-4 gap-y-2">
			{#each sections as title, i}
				<a class="link link-hover text-sm" href={`#${slug(title)}`}>
					{i + 1}. {title}
				</a>
			{/each}
		</div>
	</nav>

	<main use:guard class="mx-auto max-w-7xl space-y-12 px-4 pt-8 sm:px-8">
		<section id="buttons">
			<h2>01 · Buttons</h2>

			<p class="section-copy">Colors, sizes, shapes, disabled states, progress, and album actions.</p>

			<div class="grid gap-6 md:grid-cols-2">
				<Specimen label="Colors">
					<div class="flex flex-wrap gap-2">
						{#each ['btn-primary', 'btn-secondary', 'btn-accent', 'btn-success', 'btn-warning', 'btn-error', 'btn-ghost', 'btn-soft', 'btn-outline'] as tone}
							<button class={`btn hover-lift ${tone}`}>
								{tone.replace('btn-', '')}
							</button>
						{/each}
					</div>
				</Specimen>

				<Specimen label="Size / shape / state">
					<div class="flex flex-wrap items-center gap-2">
						<button class="btn btn-xs hover-lift"> XS </button>

						<button class="btn btn-sm hover-lift"> Small </button>

						<button class="btn btn-md hover-lift"> Medium </button>

						<button class="btn btn-lg hover-lift"> Large </button>

						<button class="btn btn-square btn-secondary hover-lift" aria-label="Edit"> ✎ </button>

						<button class="btn btn-lg btn-circle btn-accent hover-lift" aria-label="Create"> + </button>

						<button class="btn btn-primary hover-lift" disabled> Disabled </button>

						<button class="btn btn-primary hover-lift" disabled>
							<span class="loading loading-sm"> </span>Saving…
						</button>
					</div>
				</Specimen>

				<Specimen label="Album actions · enabled">
					<AlbumActions canShuffle canMarkListened spotifyId="preview" onshuffle={noop} onlistened={noop} />
				</Specimen>

				<Specimen label="Album actions · unavailable">
					<AlbumActions canShuffle={false} canMarkListened={false} onshuffle={noop} onlistened={noop} />
				</Specimen>

				<Specimen label="Add tile">
					<div class="max-w-48">
						<AddAlbumCard />
					</div>
				</Specimen>
			</div>
		</section>

		<section id="forms">
			<h2>02 · Forms</h2>

			<p class="section-copy">Search, sorting, manual forms, listening, authentication, and validation.</p>

			<div class="grid gap-6 md:grid-cols-2 xl:grid-cols-3">
				<Specimen label="Search / expanded custom sort">
					<SearchBar placeholder="Search albums…" />

					<div class="mt-4">
						<SortControls
							name="preview-sort"
							options={[
								{ label: 'Album', value: 'album' },
								{ label: 'Artist', value: 'artist' },
							]}
							bind:sort
						/>
					</div>
				</Specimen>

				<Specimen label="Manual album · filled">
					<ManualAlbumForm
						artists={[artist]}
						artistId={String(artist.id)}
						name={album.title}
						year="2024"
						comment="Classic album"
						coverUrl={artwork}
					/>
				</Specimen>

				<Specimen label="Manual album · locked artist / empty">
					<ManualAlbumForm artists={[artist]} artistId={String(artist.id)} lockArtist />
				</Specimen>

				<Specimen label="Manual artist · fallback">
					<ManualArtistForm name={artist.name} />
				</Specimen>

				<Specimen label="Artist · Spotify refresh">
					<ManualArtistForm
						name={artist.name}
						coverUrl={artwork}
						spotifyId="preview"
						showSpotifyRefresh
						onrefresh={noop}
					/>
				</Specimen>

				<Specimen label="Artist · offline / unlinked" online={false}>
					<ManualArtistForm name={artist.name} coverUrl={artwork} showSpotifyRefresh />
				</Specimen>

				<Specimen label="Artist · refreshing">
					<ManualArtistForm
						name={artist.name}
						coverUrl={artwork}
						spotifyId="preview"
						showSpotifyRefresh
						refreshing
					/>
				</Specimen>

				<Specimen label="Listened">
					<ListenedFields listened listenedAt="2026-01-03" rating={8} />
				</Specimen>

				<Specimen label="Not listened">
					<ListenedFields />
				</Specimen>

				<Specimen label="Validation / shown password">
					<YearField value="9999" />

					<p class="text-error text-sm">Enter a valid year</p>

					<FloatingField label="Invalid email">
						<input
							name="preview-field-2"
							class="input input-error w-full"
							aria-invalid="true"
							value="invalid"
						/>
					</FloatingField>

					<p class="text-error text-sm">Enter valid email address</p>

					<PasswordField value="ab" />

					<p class="text-error text-sm">Enter at least 4 characters</p>

					<PasswordField value="sample-password" shown />
				</Specimen>

				{#each ['login', 'register'] as mode}
					<Specimen label={mode === 'login' ? 'Login' : 'Create account'}>
						<div class="card bg-base-200">
							<div class="card-body">
								<AuthForm mode={mode as 'login' | 'register'} />

								<p class="text-center text-sm">
									{mode === 'login' ? 'Need an account?' : 'Already have an account?'}
									<a class="link link-primary" href="/login">
										{mode === 'login' ? 'Create account' : 'Log in'}
									</a>
								</p>
							</div>
						</div>
					</Specimen>
				{/each}
			</div>
		</section>

		<section id="media">
			<h2>03 · Media</h2>

			<p class="section-copy">Artwork, fallbacks, cards, compact variants, artist rows, heroes, and notes.</p>

			<Specimen
				label="Recommended album"
				library={{ artists: [artist], albums: [{ ...album, listened: false, rating: null }], loadedAt: 0 }}
			>
				<div class="mb-6 max-w-3xl">
					<NextAlbum />
				</div>
			</Specimen>

			<div class="grid grid-cols-2 gap-6 md:grid-cols-4">
				<Specimen label="Album · rated / newly added" newlyAdded>
					<AlbumCard {album} subtitle={artist.name} showRating />
				</Specimen>

				<Specimen label="Album · queued / fallback">
					<AlbumCard
						album={{ ...album, cover_url: null, listened: false, rating: null }}
						subtitle="Unknown artist"
					/>
				</Specimen>

				<Specimen label="Compact · rated">
					<AlbumCard {album} subtitle="2024" compact showRating />
				</Specimen>

				<Specimen label="Compact · long title / unrated">
					<AlbumCard
						album={{
							...album,
							title: 'A longer title wrapping onto more than one line',
							rating: null,
							cover_url: null,
						}}
						subtitle="Year unknown"
						compact
					/>
				</Specimen>
			</div>

			<div class="mt-6 grid gap-6 md:grid-cols-2">
				<Specimen label="Artist · rated / newly added" newlyAdded>
					<ul class="list">
						<ArtistListItem {artist} albumCount={12} averageRating={8.2} />
					</ul>
				</Specimen>

				<Specimen label="Artist · empty / fallback">
					<ul class="list">
						<ArtistListItem artist={{ ...artist, cover_url: null }} albumCount={0} averageRating={null} />
					</ul>
				</Specimen>

				<Specimen label="Artist link">
					<AlbumArtistCard {artist} />
				</Specimen>

				<Specimen label="Artist link · fallback">
					<AlbumArtistCard artist={{ ...artist, cover_url: null }} />
				</Specimen>

				<Specimen label="Hero · linked / artwork">
					<div class="overflow-hidden rounded-xl px-4 pt-2">
						<MediaHero
							title={album.title}
							subtitle="2024"
							imageUrl={artwork}
							imageAlt="Sample record"
							editLabel="Edit album"
							deleteLabel="Delete album"
							backHref="/preview"
							spotifyHref="https://open.spotify.com"
						/>
					</div>
				</Specimen>

				<Specimen label="Hero · unlinked / fallback">
					<div class="overflow-hidden rounded-xl px-4 pt-2">
						<MediaHero
							title={artist.name}
							imageAlt="Sample artist"
							editLabel="Edit artist"
							deleteLabel="Delete artist"
							backHref="/preview"
						/>
					</div>
				</Specimen>

				<Specimen label="Notes">
					<AlbumNotes comment={album.comment} />
				</Specimen>

				<Specimen label="Notes · empty">
					<AlbumNotes />
				</Specimen>
			</div>

			<div class="mt-6 grid grid-cols-2 gap-6 md:grid-cols-5">
				{#each ['card', 'list', 'artist-preview', 'album-preview'] as variant}
					<Specimen label={`Thumbnail · ${variant}`}>
						<MediaThumbnail
							variant={variant as 'card' | 'list' | 'artist-preview' | 'album-preview'}
							label="Sample"
							alt="Fallback"
						/>
					</Specimen>
				{/each}

				<Specimen label="Empty thumbnail">
					<MediaThumbnail />
				</Specimen>
			</div>
		</section>

		<section id="ratings">
			<h2>04 · Ratings & statistics</h2>

			<p class="section-copy">All rating colors, stat tones, listened states, and distribution charts.</p>

			<div class="flex flex-wrap gap-2">
				{#each Array.from({ length: 10 }, (_, i) => i + 1) as rating}
					<span class="badge badge-outline" style:color={ratingColor(rating)}>
						{rating}/10 · {ratingLabel(rating)}
					</span>
				{/each}
			</div>

			<div class="mt-6 grid gap-6 md:grid-cols-2">
				<Specimen label="Album stats · listened">
					<AlbumStats {album} addedDate="Jan 2, 2026" listenedDate="Jan 3, 2026" />
				</Specimen>

				<Specimen label="Album stats · add rating">
					<AlbumStats
						album={{ ...album, listened: false, rating: null }}
						addedDate="Jan 2, 2026"
						listenedDate={null}
					/>
				</Specimen>

				<Specimen label="Stat tones / colored value / empty">
					<div class="space-y-3">
						{#each ['primary', 'secondary', 'accent'] as tone}
							<StatCard
								value={12}
								label={`${tone} statistic`}
								icon={AlbumIcon}
								tone={tone as 'primary' | 'secondary' | 'accent'}
							/>
						{/each}

						<StatCard value="8.2" label="Average rating" icon={StarIcon} color={ratingColor(8)} />

						<StatCard value="—" label="No ratings" icon={StarIcon} />
					</div>
				</Specimen>

				<Specimen label="Statistics skeleton">
					<div class="bg-base-200 flex min-h-28 items-center gap-4 rounded-xl px-5 py-4 shadow-sm">
						<div class="skeleton size-11 shrink-0 rounded-full"></div>

						<div>
							<div class="skeleton h-3 w-16"></div>

							<div class="skeleton mt-2 h-7 w-12"></div>
						</div>
					</div>
				</Specimen>

				<Specimen label="Rating · enabled / disabled">
					<RatingInput value={8} />

					<RatingInput disabled />
				</Specimen>

				<Specimen label="Distribution · populated">
					<RatingDistributionChart
						distribution={Array.from({ length: 10 }, (_, i) => ({
							rating: i + 1,
							count: [0, 1, 2, 3, 4, 8, 12, 9, 6, 2][i],
						}))}
					/>
				</Specimen>

				<Specimen label="Distribution · empty">
					<RatingDistributionChart
						distribution={Array.from({ length: 10 }, (_, i) => ({ rating: i + 1, count: 0 }))}
					/>
				</Specimen>
			</div>
		</section>

		<section id="navigation">
			<h2>05 · Navigation & filters</h2>

			<p class="section-copy">Navigation, active items, discography and time filters.</p>

			<Specimen label="Bottom navigation">
				<BottomNav />
			</Specimen>

			<div class="mt-6 grid gap-6 md:grid-cols-2">
				<Specimen label="Active navigation">
					<nav class="dock relative">
						<a class="dock-active" href="/preview">
							<AlbumIcon class="size-5" />

							<span class="dock-label"> Albums </span>
						</a>
					</nav>
				</Specimen>

				<Specimen label="Filters">
					<div class="join w-full">
						{#each ['All', 'Listened', 'Unlistened'] as text, i}
							<input
								class="join-item btn btn-ghost border-base-300 checked:bg-primary checked:text-primary-content hover-lift flex-1 border"
								type="radio"
								name="preview-filter"
								aria-label={text}
								checked={i === 0}
							/>
						{/each}
					</div>

					<div class="mt-4 flex flex-wrap gap-2">
						{#each ['This month', 'This year', 'All time'] as text, i}
							<button
								class="btn btn-sm hover-lift"
								class:btn-secondary={i === 2}
								class:btn-soft={i !== 2}
							>
								{text}
							</button>
						{/each}
					</div>
				</Specimen>
			</div>
		</section>

		<section id="settings">
			<h2>06 · Settings & themes</h2>

			<p class="section-copy">
				Setting rows, account tones, export, theme tile variants and rating customization.
			</p>

			<div class="grid gap-6 md:grid-cols-2">
				<Specimen label="Setting · primary">
					<SettingRow
						title="Native dropdowns"
						description="Use your device’s dropdown menus."
						icon={AlbumIcon}
					>
						<input
							name="preview-field-3"
							class="toggle toggle-primary"
							type="checkbox"
							aria-label="Native dropdowns"
						/>
					</SettingRow>
				</Specimen>

				<Specimen label="Setting · accent / unavailable">
					<SettingRow
						title="Default library layout"
						description="Choose grid or compact views."
						icon={AlbumIcon}
						tone="accent"
					>
						<span class="badge badge-ghost"> Coming soon </span>
					</SettingRow>
				</Specimen>

				<Specimen label="Account · all tones / offline">
					<div class="space-y-2">
						{#each ['primary', 'secondary', 'accent', 'error'] as tone}
							<AccountSettingRow
								title={tone === 'error' ? 'Delete account'
								: tone === 'accent' ? 'Password'
								: tone === 'secondary' ? 'Sign out'
								: 'Email'}
								description="preview@example.com"
								actionLabel="Manage"
								icon={AlbumIcon}
								tone={tone as 'primary' | 'secondary' | 'accent' | 'error'}
								onclick={noop}
							/>
						{/each}

						<AccountSettingRow
							title="Account offline"
							description="Reconnect to manage your account."
							actionLabel="Manage"
							icon={AlbumIcon}
							disabled
							onclick={noop}
						/>
					</div>
				</Specimen>

				<Specimen label="Export">
					<ExportLibrarySetting />
				</Specimen>

				<Specimen label="Theme tile variants">
					<div class="grid grid-cols-2 gap-3">
						<ThemePalette {theme} label="Current" variant="current" />

						<ThemeOption theme="light" label="Quick" variant="quick" onclick={noop} />

						<ThemeOption theme="dark" label="Quick selected" variant="quick" selected onclick={noop} />

						<ThemeOption {theme} label="Catalog selected" variant="catalog" selected onclick={noop} />

						<ThemeOption theme="light" label="Catalog" variant="catalog" onclick={noop} />
					</div>
				</Specimen>

				<Specimen label="Rating customization">
					<RatingSettingsDialog />
				</Specimen>

				<Specimen label="Theme catalog">
					<ThemePickerDialog />
				</Specimen>
			</div>
		</section>

		<section id="online-offline">
			<h2>07 · Online & offline</h2>

			<p class="section-copy">Connection states, all record indicators, sync recovery and Spotify states.</p>

			<div class="grid gap-6 md:grid-cols-2">
				<Specimen label="Library sync states">
					<div class="space-y-2">
						{#each ['Your library is up to date', 'Offline', 'Syncing your library…', '2 changes waiting to sync', '2 changes need attention', 'Sync paused · Sign in to continue'] as description, i}
							<SettingRow title="Library sync" {description} icon={AlbumIcon} tone="accent">
								<button class="btn btn-accent hover-lift rounded-full" disabled={i === 1 || i === 2}>
									{#if i === 2}
										<span class="loading loading-sm"> </span>
									{/if}{i === 2 ? 'Syncing…' : 'Sync now'}
								</button>
							</SettingRow>
						{/each}
					</div>
				</Specimen>

				<div class="grid grid-cols-2 gap-4">
					{#each recordStates as status}
						<Specimen label={`Record · ${status}`} recordStatus={status}>
							<div class="flex items-center gap-3">
								<RecordSyncBadge entity="album" id={album.id} />

								<RecordSyncBadge entity="album" id={album.id} />

								{#if status === 'synced'}
									<span class="text-base-content/50 text-sm"> No indicator </span>
								{/if}
							</div>
						</Specimen>
					{/each}
				</div>

				<Specimen label="Conflict · changed server record">
					<ConflictMutationModal
						plan={{
							...plan,
							action: 'server',
							mutation: queue[2],
							baseline: { ...artist, name: 'Server artist', version: 3 },
							messages: ['Later edits are preserved. Review both versions before choosing.'],
						}}
						busy={false}
						error=""
						onresolve={noop}
					/>
				</Specimen>

				<Specimen label="Conflict · server deleted">
					<ConflictMutationModal
						plan={{
							...plan,
							action: 'server',
							baseline: null,
							messages: [
								'Recreating this record uses a new ID. Dependent changes are remapped or blocked.',
							],
						}}
						busy={false}
						error=""
						onresolve={noop}
					/>
				</Specimen>

				<Specimen label="Conflict · album">
					<ConflictMutationModal
						plan={{
							...plan,
							action: 'server',
							baseline: album,
							mutation: {
								...queue[2],
								entity: 'album',
								operation: 'update',
								entityId: album.id,
								payload: { artist_id: album.artist_id, title: 'Local album' },
								before: album,
								after: { ...album, title: 'Local album' },
							},
						}}
						busy={false}
						error=""
						onresolve={noop}
					/>
				</Specimen>

				<Specimen label="Conflict · deleted album">
					<ConflictMutationModal
						plan={{
							...plan,
							action: 'server',
							baseline: null,
							mutation: {
								...queue[2],
								entity: 'album',
								operation: 'update',
								entityId: album.id,
								payload: { artist_id: album.artist_id, title: 'Local album' },
								before: album,
								after: { ...album, title: 'Local album' },
							},
						}}
						busy={false}
						error=""
						onresolve={noop}
					/>
				</Specimen>

				<Specimen label="Sync queue · online">
					<p class="text-sm">
						{getChangesNeedingAttention(queue).length} changes need attention
					</p>

					<div class="queue-preview">
						<SyncQueueList
							mutations={queue}
							status={{ state: 'synced', lastSyncedAt: 0 }}
							recoveryDisabled={false}
							sendingDisabled={false}
							onretry={noop}
							onreconcile={noop}
							oncorrect={noop}
							ondiscard={noop}
						/>
					</div>
				</Specimen>

				<Specimen label="Sync queue · offline / disabled recovery">
					<div class="queue-preview">
						<SyncQueueList
							mutations={[queue[0], queue[2], queue[5]]}
							status={{ state: 'offline', lastSyncedAt: null }}
							recoveryDisabled
							sendingDisabled
							onretry={noop}
							onreconcile={noop}
							oncorrect={noop}
							ondiscard={noop}
						/>
					</div>
				</Specimen>

				<Specimen label="Spotify · unselected / selected / fallback">
					<ul class="list bg-base-200 rounded-box">
						<SpotifySearchResult type="album" {row} loading={false} message="" />

						<SpotifySearchResult
							type="album"
							row={{ ...row, id: 'selected' }}
							selected={{ ...row, id: 'selected' }}
							loading={false}
							message=""
						/>

						<SpotifySearchResult
							type="artist"
							row={{ id: 'fallback', name: 'Artist without artwork' }}
							loading={false}
							message=""
						/>
					</ul>

					<FloatingField label="Attribute album to">
						<select name="preview-field-4" class="select w-full">
							<option> Preview artist </option>

							<option> Guest artist </option>
						</select>
					</FloatingField>
				</Specimen>

				<Specimen label="Spotify · loading / empty / error / offline">
					<ul class="list bg-base-200 rounded-box">
						<SpotifySearchResult type="artist" loading message="" />

						{#each ['Start searching...', 'No albums found.', 'Spotify search failed.'] as message}
							<SpotifySearchResult type="album" loading={false} {message} />
						{/each}
					</ul>

					<div class="tabs tabs-border mt-4">
						<button class="tab tab-active hover-lift"> Manual </button>

						<button class="tab hover-lift" disabled> Spotify </button>
					</div>
				</Specimen>
			</div>
		</section>

		<section id="dialogs">
			<h2>08 · Dialogs</h2>

			<p class="section-copy">
				Visible inline, including edit, correction, destructive, account and busy variants.
			</p>

			<div class="grid items-start gap-6 md:grid-cols-2 xl:grid-cols-3">
				<Specimen label="Create form · ready">
					<FormModalShell title="Artist" canSave onsave={noop}>
						<div class="tabs tabs-border">
							<button class="tab tab-active hover-lift"> Manual </button>

							<button class="tab hover-lift"> Spotify </button>
						</div>

						<ManualArtistForm name="Preview artist" />
					</FormModalShell>
				</Specimen>

				<Specimen label="Create form · saving / error">
					<FormModalShell
						title="Album"
						saving
						canSave={false}
						error="Album has already been added"
						onsave={noop}
					>
						<p class="text-base-content/60 py-4">Shared album fields are shown in Forms.</p>
					</FormModalShell>
				</Specimen>

				<Specimen label="Edit album">
					<EditAlbumModal {album} {artist} onsave={async () => album} />
				</Specimen>

				<Specimen label="Edit artist">
					<EditArtistModal {artist} onsave={async () => artist} />
				</Specimen>

				<Specimen label="Correct album">
					<EditAlbumModal
						{album}
						{artist}
						mode="correct"
						recoveryMessages={['Your correction replaces the failed edit. Later changes are preserved.']}
						onsave={async () => album}
					/>
				</Specimen>

				<Specimen label="Correct artist">
					<EditArtistModal
						{artist}
						mode="correct"
						recoveryMessages={['Your correction replaces the failed edit.']}
						onsave={async () => artist}
					/>
				</Specimen>

				<Specimen label="Delete">
					<DeleteConfirmationDialog
						title="Delete album?"
						description="This removes the album from your library."
						confirmLabel="Delete album"
						onconfirm={async () => {}}
					/>
				</Specimen>

				<Specimen label="Delete · blocked">
					<DeleteConfirmationDialog
						title="Delete artist?"
						description="This also removes the artist’s albums."
						confirmLabel="Delete artist"
						disabledReason="Resolve pending changes first."
						onconfirm={async () => {}}
					/>
				</Specimen>

				<Specimen label="Discard">
					<DiscardMutationModal {plan} busy={false} error="" onconfirm={noop} />
				</Specimen>

				<Specimen label="Discard · busy / error">
					<DiscardMutationModal
						{plan}
						busy
						error="Recovery could not complete. Try again."
						onconfirm={noop}
					/>
				</Specimen>

				{#each ['email', 'password', 'delete'] as action}
					<Specimen label={`Account · ${action}`} accountAction={action as 'email' | 'password' | 'delete'}>
						<AccountActionDialog />
					</Specimen>
				{/each}
			</div>
		</section>

		<section id="feedback">
			<h2>09 · Feedback & empty states</h2>

			<p class="section-copy">
				Storage, alerts, progress, export, authentication, loading and library empty states.
			</p>

			<div class="grid gap-6 md:grid-cols-2">
				<Specimen label="Alerts">
					<div class="space-y-3">
						<StorageErrorAlert
							message="Enable browser storage for this site before saving changes."
							onretry={async () => {}}
						/>

						<div class="alert alert-error alert-soft">Could not save your changes.</div>

						<div class="alert alert-warning alert-soft">Resolve an earlier change first.</div>

						<p class="text-success">Correction saved on this device. Later edits were preserved.</p>

						<p class="text-error">Could not load your account details.</p>

						<p class="text-base-content/60">Account actions require an internet connection.</p>
					</div>
				</Specimen>

				<Specimen label="Progress / export / authentication">
					<div class="flex gap-6">
						<span class="loading" aria-label="Loading library"> </span>

						<span class="loading loading-dots" aria-label="Logging in"> </span>

						<span class="loading loading-xs" aria-label="Refreshing"> </span>
					</div>

					<div class="mt-4 space-y-3">
						<button class="btn btn-accent hover-lift rounded-full" disabled>
							<span class="loading loading-sm"> </span>Preparing…
						</button>

						<p class="text-base-content/55">Preparing your library, including queued changes…</p>

						<p class="text-base-content/55">Downloaded better-music-library.json</p>

						<p class="text-error">Could not export your library. Please try again.</p>

						<p class="text-error text-center">Could not reach the server. Please try again.</p>
					</div>
				</Specimen>

				<Specimen label="Queue empty">
					<div class="py-12 text-center">
						<h3 class="text-xl font-semibold">Your queue is all caught up</h3>

						<p class="text-base-content/60 mt-2">Add an album to find your next listen.</p>

						<button class="btn btn-primary hover-lift mt-5"> Add an album </button>
					</div>
				</Specimen>

				<Specimen label="Recently added / listened empty">
					<div class="mb-3 flex items-center justify-between gap-3">
						<h3 class="text-xl font-semibold">Recently listened</h3>

						<a class="text-base-content/60 text-sm" href="/albums"> View all → </a>
					</div>

					<p class="text-base-content/60 py-4 text-sm">
						Your recently listened albums will appear here once you mark an album as listened.
					</p>

					<p class="text-base-content/60 py-4 text-sm">
						Your recently added unrated albums will appear here.
					</p>

					<div class="divider divider-primary text-xl" style:color={ratingColor(8)}>
						8/10 • {ratingLabel(8)} • 1 album
					</div>

					<div class="divider text-xl">Unrated • 1 album</div>
				</Specimen>

				<Specimen label="Library empty / error">
					<div class="py-10 text-center">
						<p class="text-base-content/60">Loading albums...</p>

						<p class="text-base-content/60 mt-2">Loading artists...</p>

						<p class="text-base-content/60 mt-2">No listened albums yet.</p>

						<p class="text-base-content/60 mt-2">No unlistened albums match your search.</p>

						<p class="text-base-content/60 mt-2">No artists yet.</p>

						<p class="text-base-content/60 mt-2">No artists match your search.</p>

						<p class="mt-4">Could not load your albums. Please try again.</p>

						<button class="btn btn-soft hover-lift mt-4"> Try again </button>
					</div>
				</Specimen>
			</div>
		</section>

		<section id="icons">
			<h2>10 · Icons</h2>

			<p class="section-copy">Every application icon plus filled and unbranded variants.</p>

			<div class="grid grid-cols-3 gap-4 sm:grid-cols-6 lg:grid-cols-9">
				{#each icons as icon}
					<Specimen label={icon.name}>
						<icon.component class="size-7" />
					</Specimen>
				{/each}

				<Specimen label="Star · filled">
					<StarIcon filled class="size-7" />
				</Specimen>

				<Specimen label="Spotify · unbranded">
					<SpotifyIcon branded={false} class="size-7" />
				</Specimen>
			</div>
		</section>
	</main>
</div>

<style>
	section {
		scroll-margin-top: 8rem;
	}
	h2 {
		font-size: 1.5rem;
		font-weight: 800;
	}
	.section-copy {
		margin: 0.5rem 0 1.5rem;
		opacity: 0.6;
		font-size: 0.875rem;
	}
</style>
