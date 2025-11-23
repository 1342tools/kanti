import { visualSettings, type Theme, type CustomTheme } from '$lib/stores/settings';
import { browser } from '$app/environment';
import { applyCustomTheme, resetToDefaultTheme, getThemeById } from '$lib/utils/theme-manager';

// Track system preference
const prefersDarkMode = browser ? window.matchMedia('(prefers-color-scheme: dark)') : null;

// Compute the actual theme to apply
function getEffectiveTheme(userTheme: Theme): 'light' | 'dark' {
  if (userTheme === 'system') {
    return prefersDarkMode?.matches ? 'dark' : 'light';
  }
  if (userTheme === 'custom') {
    // For custom themes, we'll handle them separately
    return prefersDarkMode?.matches ? 'dark' : 'light';
  }
  return userTheme;
}

// Reactive state for the current theme
export let preferredTheme = $state({ 
  theme: 'dark' as 'light' | 'dark'
});

// Apply theme to document
function applyTheme(theme: 'light' | 'dark', customTheme?: CustomTheme) {
  if (!browser) return;
  
  const html = document.documentElement;
  
  // Reset custom theme variables first
  resetToDefaultTheme();
  
  if (theme === 'dark') {
    html.classList.add('dark');
    html.classList.remove('light');
  } else {
    html.classList.add('light');
    html.classList.remove('dark');
  }
  
  // Apply custom theme if provided
  if (customTheme) {
    applyCustomTheme(customTheme);
  }
  
  preferredTheme.theme = theme;
}

// Handle visual settings changes
function handleVisualSettingsChange($settings: any) {
  if ($settings.theme === 'custom' && $settings.customThemeId) {
    // Apply custom theme
    const customTheme = getThemeById($settings.customThemeId);
    if (customTheme) {
      // For custom themes, we need to determine if it's light or dark based on background color
      const isDarkTheme = isColorDark(customTheme.colors.bgPrimary);
      applyTheme(isDarkTheme ? 'dark' : 'light', customTheme);
    } else {
      // Fallback to system if custom theme not found
      const effectiveTheme = getEffectiveTheme('system');
      applyTheme(effectiveTheme);
    }
  } else {
    // Apply regular theme (light/dark/system)
    const effectiveTheme = getEffectiveTheme($settings.theme);
    applyTheme(effectiveTheme);
  }
}

// Helper function to determine if a color is dark
function isColorDark(color: string): boolean {
  // Convert hex to RGB
  const hex = color.replace('#', '');
  const r = parseInt(hex.substr(0, 2), 16);
  const g = parseInt(hex.substr(2, 2), 16);
  const b = parseInt(hex.substr(4, 2), 16);
  
  // Calculate luminance
  const luminance = (0.299 * r + 0.587 * g + 0.114 * b) / 255;
  
  return luminance < 0.5;
}

// Initialize and subscribe to changes
if (browser) {
  // Apply initial theme on startup
  visualSettings.subscribe($settings => {
    handleVisualSettingsChange($settings);
  })();
  
  // Subscribe to visual settings changes
  visualSettings.subscribe($settings => {
    handleVisualSettingsChange($settings);
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
