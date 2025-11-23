<script lang="ts">
  import { createCustomTheme, importTheme, exportTheme, defaultThemeColors } from '$lib/utils/theme-manager';
  import type { CustomTheme } from '$lib/stores/settings';

  export let theme: CustomTheme | null = null;
  export let onSave: (theme: CustomTheme) => void;
  export let onCancel: () => void;
  export let onDelete: (themeId: string) => void;

  // Local state
  let currentTheme: CustomTheme;
  let showImportDialog = false;
  let showExportDialog = false;
  let importJson = '';
  let exportJson = '';

  // Initialize current theme
  if (theme) {
    currentTheme = { ...theme };
  } else {
    currentTheme = createCustomTheme();
  }

  // Color groups for organized display
  const colorGroups = [
    {
      name: 'Background',
      colors: [
        { key: 'bgPrimary', label: 'Primary Background' },
        { key: 'bgSecondary', label: 'Secondary Background' },
        { key: 'bgTertiary', label: 'Tertiary Background' },
        { key: 'bgHover', label: 'Hover Background' },
        { key: 'bgActive', label: 'Active Background' }
      ]
    },
    {
      name: 'Text',
      colors: [
        { key: 'textPrimary', label: 'Primary Text' },
        { key: 'textSecondary', label: 'Secondary Text' },
        { key: 'textTertiary', label: 'Tertiary Text' },
        { key: 'textMuted', label: 'Muted Text' }
      ]
    },
    {
      name: 'Borders',
      colors: [
        { key: 'borderPrimary', label: 'Primary Border' },
        { key: 'borderSecondary', label: 'Secondary Border' },
        { key: 'borderHover', label: 'Hover Border' }
      ]
    },
    {
      name: 'Accent',
      colors: [
        { key: 'accentPrimary', label: 'Primary Accent' },
        { key: 'accentHover', label: 'Hover Accent' },
        { key: 'accentLight', label: 'Light Accent' }
      ]
    },
    {
      name: 'Shadows',
      colors: [
        { key: 'shadowSm', label: 'Small Shadow' },
        { key: 'shadowMd', label: 'Medium Shadow' },
        { key: 'shadowLg', label: 'Large Shadow' }
      ]
    },
    {
      name: 'Scrollbar',
      colors: [
        { key: 'scrollbarTrack', label: 'Track' },
        { key: 'scrollbarThumb', label: 'Thumb' },
        { key: 'scrollbarThumbHover', label: 'Thumb Hover' }
      ]
    },
    {
      name: 'Input',
      colors: [
        { key: 'inputBg', label: 'Background' },
        { key: 'inputBorder', label: 'Border' },
        { key: 'inputFocus', label: 'Focus' }
      ]
    },
    {
      name: 'Titlebar',
      colors: [
        { key: 'titlebarBg', label: 'Background' },
        { key: 'titlebarText', label: 'Text' }
      ]
    }
  ];

  // Handle color change
  function handleColorChange(key: keyof typeof defaultThemeColors, value: string) {
    currentTheme = {
      ...currentTheme,
      colors: {
        ...currentTheme.colors,
        [key]: value
      }
    };
  }

  // Handle name change
  function handleNameChange(value: string) {
    currentTheme = {
      ...currentTheme,
      name: value
    };
  }

  // Save theme
  function saveTheme() {
    onSave(currentTheme);
  }

  // Reset to defaults
  function resetToDefaults() {
    if (confirm('Reset all colors to default values?')) {
      currentTheme = {
        ...currentTheme,
        colors: { ...defaultThemeColors }
      };
    }
  }

  // Import theme
  function handleImport() {
    const importedTheme = importTheme(importJson);
    if (importedTheme) {
      currentTheme = importedTheme;
      showImportDialog = false;
      importJson = '';
    } else {
      alert('Failed to import theme. Please check the JSON format.');
    }
  }

  // Export theme
  function handleExport() {
    exportJson = exportTheme(currentTheme);
    showExportDialog = true;
  }

  // Close export dialog
  function closeExportDialog() {
    showExportDialog = false;
    exportJson = '';
  }

  // Close import dialog
  function closeImportDialog() {
    showImportDialog = false;
    importJson = '';
  }

  // Copy to clipboard
  function copyToClipboard(text: string) {
    navigator.clipboard.writeText(text).then(() => {
      alert('Theme JSON copied to clipboard!');
    }).catch(() => {
      alert('Failed to copy to clipboard');
    });
  }
</script>

<!-- Modal Overlay -->
<div class="modal-overlay" on:click={onCancel}>
  <div class="modal" on:click|stopPropagation>
    <div class="modal-header">
      <h3>{theme ? 'Edit Theme' : 'Create New Theme'}</h3>
      <button class="close-btn" on:click={onCancel}>×</button>
    </div>
    
    <div class="modal-content">
      <!-- Theme Name -->
      <div class="form-group">
        <label>
          <span class="label-text">Theme Name</span>
        </label>
        <input
          type="text"
          bind:value={currentTheme.name}
          on:input={(e) => handleNameChange((e.target as HTMLInputElement).value)}
          placeholder="Enter theme name"
          class="theme-name-input"
        />
      </div>

      <!-- Color Groups -->
      {#each colorGroups as group}
        <div class="color-group">
          <h4 class="group-title">{group.name}</h4>
          <div class="color-grid">
            {#each group.colors as color}
              <div class="color-input-group">
                <label class="color-label">{color.label}</label>
                <div class="color-input-row">
                  <input
                    type="color"
                    value={currentTheme.colors[color.key as keyof typeof defaultThemeColors]}
                    on:input={(e) => handleColorChange(color.key as keyof typeof defaultThemeColors, (e.target as HTMLInputElement).value)}
                    class="color-picker"
                  />
                  <input
                    type="text"
                    value={currentTheme.colors[color.key as keyof typeof defaultThemeColors]}
                    on:input={(e) => handleColorChange(color.key as keyof typeof defaultThemeColors, (e.target as HTMLInputElement).value)}
                    class="color-hex-input"
                    maxlength="7"
                  />
                </div>
              </div>
            {/each}
          </div>
        </div>
      {/each}
    </div>
    
    <!-- Action Buttons -->
    <div class="modal-actions">
      <div class="left-actions">
        <button class="action-btn secondary" on:click={handleExport}>Export</button>
        <button class="action-btn secondary" on:click={() => showImportDialog = true}>Import</button>
        <button class="action-btn secondary" on:click={resetToDefaults}>Reset</button>
        {#if theme}
          <button class="action-btn danger" on:click={() => onDelete(theme!.id)}>Delete</button>
        {/if}
      </div>
      <div class="right-actions">
        <button class="action-btn secondary" on:click={onCancel}>Cancel</button>
        <button class="action-btn primary" on:click={saveTheme}>
          {theme ? 'Update Theme' : 'Create Theme'}
        </button>
      </div>
    </div>
  </div>
</div>

  <!-- Import Dialog -->
  {#if showImportDialog}
    <div class="nested-modal-overlay" on:click={closeImportDialog}>
      <div class="nested-modal" on:click|stopPropagation>
        <div class="modal-header">
          <h4>Import Theme</h4>
          <button class="close-btn" on:click={closeImportDialog}>×</button>
        </div>
        <div class="modal-content">
          <p>Paste your theme JSON below:</p>
          <textarea
            bind:value={importJson}
            placeholder="Paste theme JSON here..."
            class="import-textarea"
            rows="10"
          ></textarea>
        </div>
        <div class="modal-actions">
          <button class="action-btn secondary" on:click={closeImportDialog}>Cancel</button>
          <button class="action-btn primary" on:click={handleImport}>Import</button>
        </div>
      </div>
    </div>
  {/if}

  <!-- Export Dialog -->
  {#if showExportDialog}
    <div class="nested-modal-overlay" on:click={closeExportDialog}>
      <div class="nested-modal" on:click|stopPropagation>
        <div class="modal-header">
          <h4>Export Theme</h4>
          <button class="close-btn" on:click={closeExportDialog}>×</button>
        </div>
        <div class="modal-content">
          <p>Copy the JSON below to share your theme:</p>
          <textarea
            bind:value={exportJson}
            readonly
            class="export-textarea"
            rows="10"
          ></textarea>
        </div>
        <div class="modal-actions">
          <button class="action-btn secondary" on:click={() => copyToClipboard(exportJson)}>
            Copy to Clipboard
          </button>
          <button class="action-btn primary" on:click={closeExportDialog}>Close</button>
        </div>
      </div>
    </div>
  {/if}

<style>
  /* Main Modal Styles */
  .modal-overlay {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background-color: rgba(0, 0, 0, 0.5);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
  }

  .modal {
    background-color: var(--bg-primary);
    border-radius: 8px;
    padding: 0;
    width: 90%;
    max-width: 800px;
    max-height: 90vh;
    overflow: auto;
    border: 1px solid var(--border-primary);
    box-shadow: 0 10px 30px rgba(0, 0, 0, 0.3);
  }

  .modal-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 20px;
    border-bottom: 1px solid var(--border-primary);
  }

  .modal-header h3 {
    margin: 0;
    color: var(--text-primary);
    font-size: 18px;
  }

  .modal-header h4 {
    margin: 0;
    color: var(--text-primary);
  }

  .close-btn {
    background: none;
    border: none;
    font-size: 24px;
    color: var(--text-secondary);
    cursor: pointer;
    padding: 0;
    width: 30px;
    height: 30px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 4px;
  }

  .close-btn:hover {
    color: var(--text-primary);
    background-color: var(--bg-hover);
  }

  .modal-content {
    max-height: calc(90vh - 150px);
    overflow-y: auto;
    padding: 20px;
  }

  .modal-content::-webkit-scrollbar {
    width: 6px;
  }

  .modal-content::-webkit-scrollbar-track {
    background: var(--scrollbar-track);
  }

  .modal-content::-webkit-scrollbar-thumb {
    background: var(--scrollbar-thumb);
    border-radius: 3px;
  }

  .modal-content::-webkit-scrollbar-thumb:hover {
    background: var(--scrollbar-thumb-hover);
  }

  /* Form Styles */
  .form-group {
    margin-bottom: 20px;
  }

  .label-text {
    display: block;
    font-weight: bold;
    color: var(--text-primary);
    margin-bottom: 8px;
  }

  .theme-name-input {
    width: 100%;
    padding: 10px;
    border: 1px solid var(--border-primary);
    border-radius: 6px;
    background-color: var(--input-bg);
    color: var(--text-primary);
    font-size: 14px;
  }

  .theme-name-input:focus {
    outline: none;
    border-color: var(--input-focus);
  }

  /* Color Groups */
  .color-group {
    margin-bottom: 25px;
    background-color: var(--bg-secondary);
    padding: 15px;
    border-radius: 6px;
    border: 1px solid var(--border-secondary);
  }

  .group-title {
    color: var(--text-primary);
    font-size: 16px;
    margin-bottom: 15px;
    padding-bottom: 8px;
    border-bottom: 1px solid var(--border-secondary);
  }

  .color-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
    gap: 15px;
  }

  .color-input-group {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .color-label {
    font-size: 12px;
    color: var(--text-secondary);
    font-weight: 500;
  }

  .color-input-row {
    display: flex;
    gap: 8px;
    align-items: center;
  }

  .color-picker {
    width: 40px;
    height: 40px;
    border: 1px solid var(--border-primary);
    border-radius: 6px;
    cursor: pointer;
  }

  .color-hex-input {
    flex: 1;
    padding: 8px;
    border: 1px solid var(--border-primary);
    border-radius: 6px;
    background-color: var(--input-bg);
    color: var(--text-primary);
    font-size: 12px;
    font-family: monospace;
  }

  .color-hex-input:focus {
    outline: none;
    border-color: var(--input-focus);
  }

  /* Modal Actions */
  .modal-actions {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 20px;
    border-top: 1px solid var(--border-primary);
  }

  .left-actions {
    display: flex;
    gap: 10px;
  }

  .right-actions {
    display: flex;
    gap: 10px;
  }

  /* Button Styles */
  .action-btn {
    padding: 8px 16px;
    border: 1px solid var(--border-primary);
    border-radius: 6px;
    background-color: var(--bg-secondary);
    color: var(--text-primary);
    cursor: pointer;
    font-size: 14px;
    font-weight: 500;
    transition: all 0.2s ease;
  }

  .action-btn:hover {
    background-color: var(--bg-hover);
    border-color: var(--border-hover);
  }

  .action-btn.primary {
    background-color: var(--accent-primary);
    color: white;
    border-color: var(--accent-primary);
  }

  .action-btn.primary:hover {
    background-color: var(--accent-hover);
  }

  .action-btn.danger {
    background-color: #dc3545;
    color: white;
    border-color: #dc3545;
  }

  .action-btn.danger:hover {
    background-color: #c82333;
  }

  .action-btn.secondary {
    background-color: var(--bg-secondary);
    color: var(--text-primary);
  }

  /* Nested Modal for Import/Export */
  .nested-modal-overlay {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background-color: rgba(0, 0, 0, 0.7);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1001;
  }

  .nested-modal {
    background-color: var(--bg-primary);
    border-radius: 8px;
    padding: 0;
    width: 90%;
    max-width: 600px;
    max-height: 80vh;
    overflow: auto;
    border: 1px solid var(--border-primary);
    box-shadow: 0 10px 30px rgba(0, 0, 0, 0.4);
  }

  .import-textarea,
  .export-textarea {
    width: 100%;
    padding: 12px;
    border: 1px solid var(--border-primary);
    border-radius: 6px;
    background-color: var(--input-bg);
    color: var(--text-primary);
    font-family: monospace;
    font-size: 12px;
    resize: vertical;
  }

  .export-textarea {
    background-color: var(--bg-secondary);
  }
</style>
