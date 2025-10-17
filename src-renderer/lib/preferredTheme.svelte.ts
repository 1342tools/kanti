import { visualSettings, type Theme } from '$lib/stores/settings';
import { browser } from '$app/environment';

// Track system preference
const prefersDarkMode = browser ? window.matchMedia('(prefers-color-scheme: dark)') : null;

// Compute the actual theme to apply
function getEffectiveTheme(userTheme: Theme): 'light' | 'dark' {
  if (userTheme === 'system') {
    return prefersDarkMode?.matches ? 'dark' : 'light';
  }
  return userTheme;
}

// Reactive state for the current theme
export let preferredTheme = $state({ 
  theme: 'dark' as 'light' | 'dark'
});

// Apply theme to document
function applyTheme(theme: 'light' | 'dark') {
  if (!browser) return;
  
  const html = document.documentElement;
  if (theme === 'dark') {
    html.classList.add('dark');
    html.classList.remove('light');
  } else {
    html.classList.add('light');
    html.classList.remove('dark');
  }
  
  preferredTheme.theme = theme;
}

// Initialize and subscribe to changes
if (browser) {
  // Subscribe to visual settings changes
  visualSettings.subscribe($settings => {
    const effectiveTheme = getEffectiveTheme($settings.theme);
    applyTheme(effectiveTheme);
  });
  
  // Listen for system theme changes
  prefersDarkMode?.addEventListener('change', (e) => {
    visualSettings.subscribe($settings => {
      if ($settings.theme === 'system') {
        applyTheme(e.matches ? 'dark' : 'light');
      }
    })();
  });
}
