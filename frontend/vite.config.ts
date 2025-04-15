import tailwindcss from '@tailwindcss/vite';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [tailwindcss(), sveltekit()],
	server: { // Add server configuration
		hmr: {
			clientPort: 5173 // Explicitly set the HMR client port
		}
	}
});
