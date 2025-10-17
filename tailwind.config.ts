import type { Config } from 'tailwindcss';

const config: Config = {
	content: ['./src-renderer/**/*.{html,js,svelte,ts}'],
	darkMode: 'class',
	theme: {
		extend: {
			colors: {
				'bg-primary': 'var(--bg-primary)',
				'bg-secondary': 'var(--bg-secondary)',
				'bg-tertiary': 'var(--bg-tertiary)',
				'text-primary': 'var(--text-primary)',
				'text-secondary': 'var(--text-secondary)',
				'accent-primary': 'var(--accent-primary)',
			},
		},
	},
	plugins: [],
}

export default config;
