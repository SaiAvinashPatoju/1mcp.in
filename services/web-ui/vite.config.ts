import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [sveltekit()],
	clearScreen: false,
	server: {
		port: 1420,
		strictPort: true
	},
	envPrefix: ['VITE_', 'TAURI_'],
	build: {
		target: 'es2021',
		minify: 'esbuild',
		cssMinify: true,
		reportCompressedSize: false,
		sourcemap: false,
		emptyOutDir: true,
		rollupOptions: {
			output: {
				manualChunks: {
					vendor: ['lucide-svelte', 'clsx']
				}
			}
		}
	}
});
