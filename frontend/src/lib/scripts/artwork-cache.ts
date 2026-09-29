type ArtworkLibrary = {
	artists: { cover_url?: string | null }[];
	albums: { cover_url?: string | null }[];
};

const spotifyArtworkPrefix = 'https://i.scdn.co/image/';
const pending = new Set<string>();
let running = false;
let listening = false;

const cachePendingArtwork = async () => {
	if (running || !navigator.onLine || !navigator.serviceWorker.controller || !('caches' in globalThis)) return;

	running = true;
	const urls = [...pending];
	const attempted = new Set(urls);
	let next = 0;

	const worker = async () => {
		while (next < urls.length) {
			const url = urls[next++];
			if (await caches.match(url)) {
				pending.delete(url);
				continue;
			}

			try {
				await fetch(url, { mode: 'no-cors' });
				pending.delete(url);
			} catch {}
		}
	};

	try {
		await Promise.all(Array.from({ length: Math.min(6, urls.length) }, worker));
	} finally {
		running = false;
		if ([...pending].some((url) => !attempted.has(url))) void cachePendingArtwork();
	}
};

export const cacheLibraryArtwork = (library: ArtworkLibrary) => {
	if (!('serviceWorker' in navigator)) return;

	for (const item of [...library.artists, ...library.albums]) {
		if (item.cover_url?.startsWith(spotifyArtworkPrefix)) pending.add(item.cover_url);
	}

	if (!listening) {
		listening = true;
		window.addEventListener('online', cachePendingArtwork);
		navigator.serviceWorker.addEventListener('controllerchange', cachePendingArtwork);
	}

	void cachePendingArtwork();
};
