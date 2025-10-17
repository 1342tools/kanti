<script lang="ts">
  import { visualSettings, type Theme } from '$lib/stores/settings';
  
  // Local state bound to the store
  let hideTopTabs = $visualSettings.hideTopTabs;
  let selectedTheme = $visualSettings.theme;
  
  // Handle toggle change
  function handleHideTabsChange() {
    visualSettings.update(settings => ({
      ...settings,
      hideTopTabs: hideTopTabs
    }));
  }
  
  // Handle theme change
  function handleThemeChange(theme: Theme) {
    selectedTheme = theme;
    visualSettings.update(settings => ({
      ...settings,
      theme: theme
    }));
  }
</script>

<div class="theme-settings">
  <h3>Visual Options</h3>
  
  <!-- Theme Selection -->
  <div class="form-group">
    <label>
      <span class="label-text">Theme</span>
      <p class="description">Choose your preferred color theme</p>
    </label>
    
    <div class="theme-options">
      <button 
        class="theme-option"
        class:selected={selectedTheme === 'light'}
        on:click={() => handleThemeChange('light')}
      >
        <div class="theme-preview light-preview">
          <div class="preview-header"></div>
          <div class="preview-content"></div>
        </div>
        <span class="theme-name">Light</span>
      </button>
      
      <button 
        class="theme-option"
        class:selected={selectedTheme === 'dark'}
        on:click={() => handleThemeChange('dark')}
      >
        <div class="theme-preview dark-preview">
          <div class="preview-header"></div>
          <div class="preview-content"></div>
        </div>
        <span class="theme-name">Dark</span>
      </button>
      
      <button 
        class="theme-option"
        class:selected={selectedTheme === 'system'}
        on:click={() => handleThemeChange('system')}
      >
        <div class="theme-preview system-preview">
          <div class="preview-header"></div>
          <div class="preview-content"></div>
        </div>
        <span class="theme-name">System</span>
      </button>
    </div>
  </div>
  
  <!-- Navigation Options -->
  <div class="form-group">
    <label>
      <span class="label-text">Navigation</span>
      <p class="description">Configure how navigation elements are displayed</p>
    </label>
    
    <div class="option-row">
      <div class="option-info">
        <span class="option-label">Hide Top Tabs</span>
        <p class="option-description">Hide the tab bar at the top of the page (use sidebar for navigation)</p>
      </div>
      <label class="toggle-switch">
        <input 
          type="checkbox" 
          bind:checked={hideTopTabs}
          on:change={handleHideTabsChange}
        />
        <span class="slider"></span>
      </label>
    </div>
  </div>
</div>

<style>
  .theme-settings {
    color: var(--text-primary);
  }
  
  .theme-settings h3 {
    margin-bottom: 20px;
    padding-bottom: 10px;
    border-bottom: 1px solid var(--border-primary);
    color: var(--text-primary);
    font-size: 18px;
  }
  
  .form-group {
    margin-bottom: 25px;
    background-color: var(--bg-tertiary);
    padding: 20px;
    border-radius: 8px;
    border: 1px solid var(--border-primary);
  }
  
  .label-text {
    display: block;
    font-weight: bold;
    color: var(--text-primary);
    margin-bottom: 5px;
    font-size: 16px;
  }
  
  .description {
    color: var(--text-secondary);
    font-size: 14px;
    margin: 5px 0 15px 0;
  }
  
  /* Theme Options */
  .theme-options {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 15px;
  }
  
  .theme-option {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 10px;
    padding: 15px;
    background-color: var(--bg-secondary);
    border: 2px solid var(--border-primary);
    border-radius: 8px;
    cursor: pointer;
    transition: all 0.2s ease;
  }
  
  .theme-option:hover {
    background-color: var(--bg-hover);
    border-color: var(--border-hover);
    transform: translateY(-2px);
  }
  
  .theme-option.selected {
    border-color: var(--accent-primary);
    background-color: var(--bg-hover);
  }
  
  .theme-preview {
    width: 100%;
    height: 80px;
    border-radius: 6px;
    overflow: hidden;
    border: 1px solid rgba(0, 0, 0, 0.1);
  }
  
  .preview-header {
    height: 25%;
    width: 100%;
  }
  
  .preview-content {
    height: 75%;
    width: 100%;
  }
  
  .light-preview {
    background-color: #ffffff;
  }
  
  .light-preview .preview-header {
    background-color: #f0f0f0;
  }
  
  .light-preview .preview-content {
    background-color: #ffffff;
  }
  
  .dark-preview {
    background-color: #1e1e1e;
  }
  
  .dark-preview .preview-header {
    background-color: #212121;
  }
  
  .dark-preview .preview-content {
    background-color: #1e1e1e;
  }
  
  .system-preview {
    background: linear-gradient(to right, #ffffff 50%, #1e1e1e 50%);
  }
  
  .system-preview .preview-header {
    background: linear-gradient(to right, #f0f0f0 50%, #212121 50%);
  }
  
  .system-preview .preview-content {
    background: linear-gradient(to right, #ffffff 50%, #1e1e1e 50%);
  }
  
  .theme-name {
    font-size: 14px;
    font-weight: 500;
    color: var(--text-primary);
  }
  
  /* Navigation Options */
  .option-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 15px;
    background-color: var(--bg-secondary);
    border-radius: 6px;
    margin-bottom: 10px;
  }
  
  .option-row:last-child {
    margin-bottom: 0;
  }
  
  .option-info {
    flex: 1;
  }
  
  .option-label {
    display: block;
    color: var(--text-primary);
    font-weight: 500;
    margin-bottom: 4px;
  }
  
  .option-description {
    color: var(--text-secondary);
    font-size: 13px;
    margin: 0;
  }
  
  /* Toggle Switch Styles */
  .toggle-switch {
    position: relative;
    display: inline-block;
    width: 50px;
    height: 26px;
    flex-shrink: 0;
  }
  
  .toggle-switch input {
    opacity: 0;
    width: 0;
    height: 0;
  }
  
  .slider {
    position: absolute;
    cursor: pointer;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background-color: var(--border-primary);
    transition: 0.3s;
    border-radius: 26px;
  }
  
  .slider:before {
    position: absolute;
    content: "";
    height: 20px;
    width: 20px;
    left: 3px;
    bottom: 3px;
    background-color: white;
    transition: 0.3s;
    border-radius: 50%;
  }
  
  input:checked + .slider {
    background-color: var(--accent-primary);
  }
  
  input:checked + .slider:before {
    transform: translateX(24px);
  }
  
  .slider:hover {
    opacity: 0.9;
  }
  
  input:checked + .slider:hover {
    background-color: var(--accent-hover);
  }
</style>
