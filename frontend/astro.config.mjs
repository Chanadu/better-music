// @ts-check
import { defineConfig } from 'astro/config';

import tailwindcss from '@tailwindcss/vite';

// https://astro.build/config
export default defineConfig({
	vite: {
		server: {
			proxy: {
			  '/api': 'http://localhost:8080',
			},
		},
		plugins: [tailwindcss()],
	}
});
