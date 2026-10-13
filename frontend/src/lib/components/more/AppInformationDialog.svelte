<script lang="ts">
	import ModalShell from '$lib/components/common/ModalShell.svelte';
	import MusicIcon from '$lib/components/icons/MusicIcon.svelte';
	import GitHubIcon from '$lib/components/icons/GitHubIcon.svelte';
	import AlbumIcon from '$lib/components/icons/AlbumIcon.svelte';
	import StarIcon from '$lib/components/icons/StarIcon.svelte';
	import SyncIcon from '$lib/components/icons/SyncIcon.svelte';
	import DownloadIcon from '$lib/components/icons/DownloadIcon.svelte';

	let dialog = $state<HTMLDialogElement>();

	const features = [
		{ icon: AlbumIcon, title: 'Your library', description: 'Albums and artists, from Spotify or added by you.' },
		{ icon: StarIcon, title: 'Every listen', description: 'Keep ratings, notes, dates, and listening stats.' },
		{ icon: SyncIcon, title: 'Ready offline', description: 'Save edits offline and sync when you reconnect.' },
		{ icon: DownloadIcon, title: 'Make it yours', description: 'Custom themes and ratings, with JSON export.' },
	];

	export function open() {
		dialog?.showModal();
	}
</script>

<ModalShell bind:dialog class="w-[calc(100%-2rem)] max-w-xl p-5 sm:p-7">
	<header class="flex items-center gap-4">
		<div
			class="from-primary/20 to-accent/15 text-primary flex size-14 shrink-0 items-center justify-center rounded-2xl bg-linear-to-br"
		>
			<MusicIcon class="size-7" />
		</div>
		<div>
			<p class="text-base-content/50 text-xs font-bold tracking-widest uppercase">App information</p>
			<h2 class="mt-1 text-2xl font-bold tracking-tight">Better Music</h2>
		</div>
	</header>

	<p class="text-base-content/70 mt-5 text-sm leading-relaxed">
		A home for your music journey. Queue your next discovery, remember your favorite listens, and build a personal
		library at your own pace.
	</p>
	<p class="text-base-content/50 mt-2 text-xs">Self-hosted. Built for desktop and mobile.</p>

	<div class="mt-6 grid grid-cols-2">
		{#each features as feature, index}
			<div
				class={`relative flex gap-3 py-4 ${index % 2 === 0 ? 'after:bg-base-content/10 pr-3 after:absolute after:inset-y-4 after:right-0 after:w-px sm:pr-4' : 'pl-3 sm:pl-4'} ${index < 2 ? 'before:bg-base-content/10 before:absolute before:inset-x-8 before:bottom-0 before:h-px' : ''}`}
			>
				<feature.icon class="text-primary mt-0.5 size-5 shrink-0" />
				<div>
					<h3 class="text-sm font-bold">{feature.title}</h3>
					<p class="text-base-content/60 mt-1 text-xs leading-relaxed">{feature.description}</p>
				</div>
			</div>
		{/each}
	</div>

	<div
		class="before:bg-base-content/10 relative flex items-center justify-between gap-4 py-3 pr-8 pl-8 before:absolute before:inset-x-8 before:top-0 before:h-px sm:pr-16"
	>
		<div class="max-w-xs min-w-0">
			<h3 class="text-sm font-bold">Find us on GitHub</h3>
			<p class="text-base-content/60 mt-1 text-xs leading-relaxed">
				Source code, documentation, and issue reporting.
			</p>
			<p class="text-primary mt-2 text-xs font-medium break-all">Chanadu/better-music</p>
		</div>
		<a
			class="hover-lift focus-visible:outline-primary flex size-20 shrink-0 items-center justify-center rounded-full focus-visible:outline-2 focus-visible:outline-offset-2"
			href="https://github.com/Chanadu/better-music"
			target="_blank"
			rel="noopener noreferrer"
			aria-label="Open Better Music on GitHub (opens in a new tab)"
			title="Open on GitHub"
		>
			<GitHubIcon class="size-16" />
		</a>
	</div>

	<div class="modal-action mt-6">
		<button type="button" class="btn hover-lift btn-primary rounded-full px-7" onclick={() => dialog?.close()}
			>Close</button
		>
	</div>
</ModalShell>
