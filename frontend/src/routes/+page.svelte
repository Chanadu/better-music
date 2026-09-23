<script lang="ts">
	import NextAlbum from '$lib/components/albums/NextAlbum.svelte';
	import RecentAlbums from '$lib/components/albums/RecentAlbums.svelte';
</script>

<svelte:head><title>Home · Better Music</title></svelte:head>

<div class="mx-auto flex w-full flex-col gap-8 py-4 sm:gap-10 sm:py-6 lg:w-4/5">
	<NextAlbum />
	<div class="flex flex-col gap-7 sm:gap-8">
		<RecentAlbums mode="listened" />
		<RecentAlbums mode="added" />
	</div>
</div>
