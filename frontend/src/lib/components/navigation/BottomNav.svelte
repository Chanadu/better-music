<script lang="ts">
	import { page } from '$app/state';
	import AlbumIcon from '$lib/components/icons/AlbumIcon.svelte';

	const items = [
		{ href: '/', label: 'Home', icon: 'M3 12h18M5 12v7h14v-7M7 9l5-5 5 5' },
		{
			href: '/albums',
			label: 'Albums',
			icon: '',
		},
		{
			href: '/artists',
			label: 'Artists',
			icon: 'M16 7a4 4 0 1 1-8 0 4 4 0 0 1 8 0zM5 21a7 7 0 0 1 14 0',
		},
		{
			href: '/listen',
			label: 'Listen',
			icon: 'M9 18V5l12-2v13M9 18a3 3 0 1 1-6 0 3 3 0 0 1 6 0zM21 16a3 3 0 1 1-6 0 3 3 0 0 1 6 0z',
		},
		{ href: '/more', label: 'More', icon: 'M5 12h.01M12 12h.01M19 12h.01' },
	];
</script>

<nav class="dock dock-sm sm:dock-lg">
	{#each items as item}
		{@const isActive = page.url.pathname === item.href}
		<a
			href={item.href}
			class={isActive ? 'dock-active text-secondary' : 'text-primary'}
			aria-current={isActive ? 'page' : undefined}
		>
			{#if item.href === '/albums'}
				<AlbumIcon class="h-5 w-5" />
			{:else}
				<svg
					viewBox="0 0 24 24"
					fill="none"
					stroke="currentColor"
					stroke-width="2"
					stroke-linecap="round"
					stroke-linejoin="round"
					class="h-5 w-5"
				>
					<path d={item.icon}></path>
				</svg>
			{/if}
			<span class="dock-label">{item.label}</span>
		</a>
	{/each}
</nav>
