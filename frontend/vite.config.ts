import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { VitePWA, type ManifestOptions } from 'vite-plugin-pwa';
import { defineConfig } from 'vite';

const manifest: Partial<ManifestOptions> = {
	name: 'Better Music',
	short_name: 'Better Music',
	description: 'Track artists, albums, and listening history.',
	start_url: '/',
	scope: '/',
	display: 'standalone',
	background_color: '#0f172a',
	theme_color: '#0f172a',
	icons: [
		{
			src: '/pwa-icon-192.png',
			sizes: '192x192',
			type: 'image/png',
			purpose: 'any maskable',
		},
		{
			src: '/pwa-icon-512.png',
			sizes: '512x512',
			type: 'image/png',
			purpose: 'any maskable',
		},
	],
};

export default defineConfig({
	plugins: [
		tailwindcss(),
		sveltekit(),
		{
			name: 'dev-manifest',
			apply: 'serve',
			configureServer(server) {
				server.middlewares.use('/manifest.webmanifest', (_request, response) => {
					response.setHeader('Content-Type', 'application/manifest+json');
					response.end(JSON.stringify(manifest));
				});
			},
		},
		VitePWA({
			registerType: 'autoUpdate',
			manifest,
			workbox: {
				globPatterns: ['**/*.{css,js,html,svg,png,ico,webmanifest}'],
				// The static adapter writes the fallback after Workbox scans the client output.
				additionalManifestEntries: [{ url: '/200.html', revision: Date.now().toString() }],
				navigateFallback: '/200.html',
				runtimeCaching: [
					{
						urlPattern: /^https:\/\/i\.scdn\.co\/image\//,
						handler: 'CacheFirst',
						options: {
							cacheName: 'better-music-cover-art',
							cacheableResponse: { statuses: [0, 200] },
							expiration: {
								maxEntries: 2000,
							},
						},
					},
				],
			},
		}),
	],
	server: {
		allowedHosts: ['chanadu-laptop-linux.tail559d08.ts.net', 'web-pi.tail559d08.ts.net'],
		proxy: { '/api': 'http://localhost:8080' },
	},
});
