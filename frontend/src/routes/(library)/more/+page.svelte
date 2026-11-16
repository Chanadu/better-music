<script lang="ts">
	import { onMount } from 'svelte';
	import LibraryStats from '$lib/components/more/LibraryStats.svelte';
	import AccountSettings from '$lib/components/more/AccountSettings.svelte';
	import AppSettings from '$lib/components/more/AppSettings.svelte';

	type LivePreview = { postMessage(url: string): void };
	type PreviewWindow = Window & {
		webkit?: { messageHandlers?: { betterMusicLivePreview?: LivePreview } };
	};
	const previewUrlKey = 'better-music-live-preview-url';
	const invalidPreviewUrl = 'Enter a Tailscale HTTPS address or an HTTP URL with a Tailscale IPv4 address.';
	let livePreview = $state.raw<LivePreview>();
	let laptopUrl = $state('');
	let previewError = $state('');

	onMount(() => {
		livePreview = (window as PreviewWindow).webkit?.messageHandlers?.betterMusicLivePreview;
		if (livePreview) laptopUrl = localStorage.getItem(previewUrlKey) ?? '';
	});

	function openLivePreview(event: SubmitEvent) {
		event.preventDefault();
		let url: URL;
		try {
			url = new URL(laptopUrl.trim());
		} catch {
			previewError = invalidPreviewUrl;
			return;
		}
		const tailscaleHttps = url.protocol === 'https:' && url.hostname.endsWith('.ts.net') && !url.port;
		const [, secondOctet, thirdOctet, fourthOctet] = url.hostname.split('.').map(Number);
		const tailscaleIp =
			url.protocol === 'http:' &&
			/^100(?:\.\d{1,3}){3}$/.test(url.hostname) &&
			secondOctet >= 64 &&
			secondOctet <= 127 &&
			thirdOctet <= 255 &&
			fourthOctet <= 255 &&
			url.port !== '0';
		if ((!tailscaleHttps && !tailscaleIp) || url.username || url.password) {
			previewError = invalidPreviewUrl;
			return;
		}
		previewError = '';
		laptopUrl = url.origin;
		localStorage.setItem(previewUrlKey, laptopUrl);
		livePreview?.postMessage(new URL('/preview/redesign', url.origin).href);
	}
</script>

<svelte:head>
	<title>More · Better Music</title>
</svelte:head>

<div class="mx-auto flex w-full flex-col py-4 sm:py-6 lg:w-4/5">
	<LibraryStats />

	<div class="divider my-8 sm:my-10"></div>

	<AppSettings />

	<div class="divider my-8 sm:my-10"></div>

	<AccountSettings />

	<a class="btn btn-outline btn-sm mt-8 self-start" href="/preview">Component preview</a>

	{#if livePreview}
		<details class="border-base-300 mt-8 border-t pt-4">
			<summary class="cursor-pointer font-semibold">Developer</summary>
			<form class="mt-4 flex max-w-md flex-col items-start gap-2" onsubmit={openLivePreview}>
				<label for="live-preview-url" class="font-medium">Live redesign preview URL</label>
				<input
					id="live-preview-url"
					class="input input-bordered w-full"
					type="url"
					placeholder="https://your-laptop.example.ts.net"
					autocapitalize="none"
					autocomplete="off"
					spellcheck={false}
					bind:value={laptopUrl}
					required
				/>
				{#if previewError}<p class="text-error text-sm" role="alert">{previewError}</p>{/if}
				<button class="btn btn-outline btn-sm mt-1" type="submit">Open live preview</button>
			</form>
		</details>
	{/if}
</div>
