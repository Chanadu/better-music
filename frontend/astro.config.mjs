// @ts-check
import { defineConfig } from 'astro/config';
import { VitePWA } from 'vite-plugin-pwa';

import tailwindcss from '@tailwindcss/vite';

// https://astro.build/config
export default defineConfig({
	vite: {
		server: {
			proxy: {
				'/api': 'http://localhost:8080',
			},
		},
		plugins: [
			tailwindcss(),
			VitePWA({
				registerType: 'autoUpdate',
				injectRegister: 'auto',
				includeAssets: ['favicon.ico', 'favicon.svg'],
				manifest: {
					name: 'Better-Music',
					short_name: 'Better-Music',
					description: 'Track artists and albums',
					theme_color: '#3b82f6',
					background_color: '#f6f7fb',
					display: 'standalone',
					start_url: '/',
					icons: [
						{
							src: 'favicon.svg',
							sizes: 'any',
							type: 'image/svg+xml',
							purpose: 'any',
						},
					],
				},
				workbox: {
					globPatterns: ['**/*.{js,css,html,ico,svg,png,webp}'],
					navigateFallback: '/offline/',
				},
			}),
		],
	},
});
