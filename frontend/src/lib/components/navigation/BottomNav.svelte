<script lang="ts">
	import { page } from '$app/state';
	import AlbumIcon from '$lib/components/icons/AlbumIcon.svelte';
	import ArtistIcon from '$lib/components/icons/ArtistIcon.svelte';
	import HomeIcon from '$lib/components/icons/HomeIcon.svelte';
	import MoreIcon from '$lib/components/icons/MoreIcon.svelte';
	import MusicIcon from '$lib/components/icons/MusicIcon.svelte';

	const items = [
		{ href: '/', label: 'Home', icon: HomeIcon },
		{ href: '/albums', label: 'Albums', icon: AlbumIcon },
		{ href: '/artists', label: 'Artists', icon: ArtistIcon },
		{ href: '/listen', label: 'Listen', icon: MusicIcon },
		{ href: '/more', label: 'More', icon: MoreIcon },
	];
</script>

<nav class="dock dock-sm sm:dock-lg">
	{#each items as item}
		{@const isActive = page.url.pathname === item.href}
		{@const Icon = item.icon}
		<a
			href={item.href}
			class={isActive ? 'dock-active text-secondary' : 'text-primary'}
			aria-current={isActive ? 'page' : undefined}
		>
			<Icon class="h-5 w-5" />
			<span class="dock-label">{item.label}</span>
		</a>
	{/each}
</nav>
