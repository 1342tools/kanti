import { writable } from 'svelte/store';

// Initialize theme from localStorage if available
const initialTheme = (() => {
    if (typeof window !== 'undefined') {
        const savedTheme = localStorage.getItem('theme');
        if (savedTheme === 'dark' || savedTheme === 'light') {
            return savedTheme;
        }
    }
    // Default to dark mode if no theme is saved in localStorage
    return 'dark';
})();

export const theme = writable<'light' | 'dark'>(initialTheme);

// Watch for theme changes and save to localStorage
if (typeof window !== 'undefined') {
    theme.subscribe(value => {
        localStorage.setItem('theme', value);
        document.documentElement.setAttribute('data-theme', value);
    });
}
