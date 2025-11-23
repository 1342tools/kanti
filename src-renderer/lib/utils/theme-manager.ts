import { visualSettings, type CustomTheme } from '$lib/stores/settings';

// Generate a unique ID for themes
function generateId(): string {
  return `theme_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
}

// Default theme colors (light theme as base)
export const defaultThemeColors = {
  // Background colors
  bgPrimary: '#ffffff',
  bgSecondary: '#f8f9fa',
  bgTertiary: '#e9ecef',
  bgHover: '#f1f3f4',
  bgActive: '#e8eaed',
  
  // Text colors
  textPrimary: '#1a1a1a',
  textSecondary: '#5f6368',
  textTertiary: '#80868b',
  textMuted: '#9aa0a6',
  
  // Border colors
  borderPrimary: '#dadce0',
  borderSecondary: '#e8eaed',
  borderHover: '#c6c6c6',
  
  // Accent colors
  accentPrimary: '#1a73e8',
  accentHover: '#1669d6',
  accentLight: '#e8f0fe',
  
  // Shadow colors
  shadowSm: 'rgba(0, 0, 0, 0.1)',
  shadowMd: 'rgba(0, 0, 0, 0.15)',
  shadowLg: 'rgba(0, 0, 0, 0.2)',
  
  // Scrollbar colors
  scrollbarTrack: '#f1f1f1',
  scrollbarThumb: '#c1c1c1',
  scrollbarThumbHover: '#a8a8a8',
  
  // Input colors
  inputBg: '#ffffff',
  inputBorder: '#dadce0',
  inputFocus: '#1a73e8',
  
  // Titlebar colors
  titlebarBg: '#f8f9fa',
  titlebarText: '#1a1a1a'
};

// Create a new custom theme
export function createCustomTheme(name: string = 'New Theme'): CustomTheme {
  return {
    id: generateId(),
    name,
    colors: { ...defaultThemeColors },
    createdAt: Date.now(),
    updatedAt: Date.now()
  };
}

// Add a custom theme
export function addCustomTheme(theme: CustomTheme): void {
  visualSettings.update(settings => ({
    ...settings,
    customThemes: [...settings.customThemes, theme]
  }));
}

// Update a custom theme
export function updateCustomTheme(themeId: string, updates: Partial<CustomTheme>): void {
  visualSettings.update(settings => ({
    ...settings,
    customThemes: settings.customThemes.map(theme => 
      theme.id === themeId 
        ? { ...theme, ...updates, updatedAt: Date.now() }
        : theme
    )
  }));
}

// Delete a custom theme
export function deleteCustomTheme(themeId: string): void {
  visualSettings.update(settings => {
    const updatedCustomThemes = settings.customThemes.filter(theme => theme.id !== themeId);
    const shouldResetTheme = settings.customThemeId === themeId;
    
    return {
      ...settings,
      customThemes: updatedCustomThemes,
      ...(shouldResetTheme && {
        theme: 'system',
        customThemeId: undefined
      })
    };
  });
}

// Set custom theme as active
export function setCustomTheme(themeId: string): void {
  visualSettings.update(settings => ({
    ...settings,
    theme: 'custom',
    customThemeId: themeId
  }));
}

// Apply custom theme CSS variables
export function applyCustomTheme(theme: CustomTheme): void {
  if (typeof document === 'undefined') return;
  
  const root = document.documentElement;
  
  // Apply all color variables
  Object.entries(theme.colors).forEach(([key, value]) => {
    const cssVarName = `--${key.replace(/([A-Z])/g, '-$1').toLowerCase()}`;
    root.style.setProperty(cssVarName, value);
  });
}

// Reset to default theme
export function resetToDefaultTheme(): void {
  if (typeof document === 'undefined') return;
  
  const root = document.documentElement;
  
  // Remove all custom theme variables
  Object.keys(defaultThemeColors).forEach(key => {
    const cssVarName = `--${key.replace(/([A-Z])/g, '-$1').toLowerCase()}`;
    root.style.removeProperty(cssVarName);
  });
}

// Export theme as JSON
export function exportTheme(theme: CustomTheme): string {
  return JSON.stringify(theme, null, 2);
}

// Import theme from JSON
export function importTheme(json: string): CustomTheme | null {
  try {
    const theme = JSON.parse(json) as CustomTheme;
    
    // Validate theme structure
    if (!theme.id || !theme.name || !theme.colors) {
      throw new Error('Invalid theme format');
    }
    
    // Generate new ID to avoid conflicts
    theme.id = generateId();
    theme.createdAt = Date.now();
    theme.updatedAt = Date.now();
    
    return theme;
  } catch (error) {
    console.error('Failed to import theme:', error);
    return null;
  }
}

// Get theme by ID
export function getThemeById(themeId: string): CustomTheme | undefined {
  let theme: CustomTheme | undefined;
  
  visualSettings.subscribe(settings => {
    theme = settings.customThemes.find(t => t.id === themeId);
  })();
  
  return theme;
}

// Duplicate a theme
export function duplicateTheme(themeId: string): void {
  const originalTheme = getThemeById(themeId);
  if (!originalTheme) return;
  
  const duplicatedTheme: CustomTheme = {
    ...originalTheme,
    id: generateId(),
    name: `${originalTheme.name} (Copy)`,
    createdAt: Date.now(),
    updatedAt: Date.now()
  };
  
  addCustomTheme(duplicatedTheme);
}
