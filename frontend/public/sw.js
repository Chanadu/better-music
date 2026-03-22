const SHELL_CACHE = 'better-music-shell-v3';
const PAGE_CACHE = 'better-music-pages-v3';
const IMAGE_CACHE = 'better-music-images-v3';
const OFFLINE_FALLBACK_PATH = '/offline';
const APP_ROUTES = ['/', '/albums', '/listened', '/artists', '/register', OFFLINE_FALLBACK_PATH];
const STATIC_ASSETS = ['/manifest.webmanifest', '/favicon.ico', '/favicon.svg'];
const SYNC_REQUEST_TAG = 'better-music-library-sync';
const SYNC_MESSAGE_TYPE = 'better-music-sync-request';

self.addEventListener('install', (event) => {
	event.waitUntil(
		Promise.all([
			caches.open(PAGE_CACHE).then((cache) => cache.addAll(APP_ROUTES)),
			caches.open(SHELL_CACHE).then((cache) => cache.addAll(STATIC_ASSETS)),
		]),
	);
	self.skipWaiting();
});

self.addEventListener('activate', (event) => {
	event.waitUntil(
		(async () => {
			const keys = await caches.keys();
			await Promise.all(
				keys
					.filter((key) => ![SHELL_CACHE, PAGE_CACHE, IMAGE_CACHE].includes(key))
					.map((key) => caches.delete(key)),
			);
			await self.clients.claim();
		})(),
	);
});

const isAppPageRequest = (request) => {
	const url = new URL(request.url);
	return (
		request.method === 'GET' &&
		url.origin === self.location.origin &&
		!url.pathname.startsWith('/api') &&
		!/\.[a-z0-9]+$/i.test(url.pathname)
	);
};

const getOfflineFallback = async (request) => {
	const url = new URL(request.url);
	const fallbackPath = url.pathname === '/login' ? '/' : url.pathname;

	return (
		(await caches.match(fallbackPath)) ||
		(await caches.match('/')) ||
		(await caches.match(OFFLINE_FALLBACK_PATH)) ||
		Response.error()
	);
};

const notifyClientsToSync = async () => {
	const clients = await self.clients.matchAll({
		type: 'window',
		includeUncontrolled: true,
	});

	for (const client of clients) {
		client.postMessage({ type: SYNC_MESSAGE_TYPE });
	}
};

self.addEventListener('fetch', (event) => {
	const { request } = event;
	const url = new URL(request.url);

	if (request.method !== 'GET' || url.pathname.startsWith('/api')) {
		return;
	}

	if (isAppPageRequest(request)) {
		event.respondWith(
			(async () => {
				try {
					const response = await fetch(request);
					if (response.ok) {
						const cache = await caches.open(PAGE_CACHE);
						cache.put(request, response.clone());
					}
					return response;
				} catch {
					return getOfflineFallback(request);
				}
			})(),
		);
		return;
	}

	if (request.destination === 'image') {
		event.respondWith(
			(async () => {
				const cached = await caches.match(request);
				const fetchPromise = fetch(request)
					.then(async (response) => {
						if (response.ok || response.type === 'opaque') {
							const cache = await caches.open(IMAGE_CACHE);
							cache.put(request, response.clone());
						}
						return response;
					})
					.catch(() => cached);

				return cached || fetchPromise || Response.error();
			})(),
		);
		return;
	}

	if (
		url.origin === self.location.origin &&
		(request.destination === 'script' || request.destination === 'style' || request.destination === 'font')
	) {
		event.respondWith(
			(async () => {
				const cached = await caches.match(request);
				if (cached) return cached;
				const response = await fetch(request);
				if (response.ok) {
					const cache = await caches.open(SHELL_CACHE);
					cache.put(request, response.clone());
				}
				return response;
			})(),
		);
	}
});

self.addEventListener('sync', (event) => {
	if (event.tag !== SYNC_REQUEST_TAG) return;
	event.waitUntil(notifyClientsToSync());
});
