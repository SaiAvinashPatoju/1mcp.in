import { defineConfig } from 'vitest/config';
import { sveltekit } from '@sveltejs/kit/vite';

export default defineConfig({
	plugins: [sveltekit()],
	test: {
		include: ['src/**/*.test.ts', 'src/**/*.test.svelte'],
		environment: 'jsdom',
		globals: true,
		setupFiles: ['./src/test-setup.ts'],
	},
	resolve: {
		conditions: ['browser'],
	},
});
