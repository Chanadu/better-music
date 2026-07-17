// @ts-check
import { defineConfig } from 'astro/config';
import tailwindcss from '@tailwindcss/vite';
import { VitePWA } from 'vite-plugin-pwa';

// https://astro.build/config
export default defineConfig({
	vite: {
		plugins: [
			tailwindcss(),
			VitePWA({
				registerType: 'autoUpdate',
				includeAssets: ['favicon.svg', 'favicon.ico', 'apple-touch-icon.png'],
				manifest: {
					name: 'Better Music',
					short_name: 'Better Music',
					description: 'Track artists, albums, and listening history.',
					start_url: '/listen',
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
				},
				workbox: {
					globPatterns: ['**/*.{css,js,html,svg,png,ico,webmanifest}'],
					navigateFallback: '/listen',
				},
			}),
		],
		server: {
			proxy: {
				'/api': 'http://localhost:8080',
			},
		},
	},
});
