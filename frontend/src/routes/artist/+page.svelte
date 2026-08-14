<script lang="ts">
	import { page } from '$app/state';
	import { onMount } from 'svelte';
	import TopNav from '$lib/components/navigation/TopNav.svelte';
	import { ApiError, artistsApi } from '$lib/scripts/api';

	let name = $state('');
	let status = $state('Loading artist...');

	onMount(async () => {
		const raw = page.url.searchParams.get('id');
		const id = Number(raw);

		if (!raw || !Number.isInteger(id) || id < 1) {
			status = 'Artist not found.';
			return;
		}

		try {
			name = (await artistsApi.get(id)).name;
			status = '';
		} catch (error) {
			status = error instanceof ApiError && error.status === 404 ? 'Artist not found.' : 'Could not load artist.';
		}
	});
</script>

<TopNav breadcrumbs={['Artists', name || 'Artist']} />

<div class="divider mt-2 mb-0"></div>

<section class="mt-6 px-2">
	{#if status}
		<p class="text-base-content/60">{status}</p>
	{:else}
		<h2 class="text-3xl font-bold wrap-break-word">
			{name}
		</h2>
	{/if}
</section>
