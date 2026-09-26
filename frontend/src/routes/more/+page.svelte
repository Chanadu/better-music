<script lang="ts">
	import LibraryStats from '$lib/components/more/LibraryStats.svelte';
	import AccountSettings from '$lib/components/more/AccountSettings.svelte';
</script>

<svelte:head><title>More · Better Music</title></svelte:head>

<div class="mx-auto flex w-full flex-col py-4 sm:py-6 lg:w-4/5">
	<LibraryStats />
	<div class="divider my-8 sm:my-10"></div>
	<AccountSettings />
</div>
