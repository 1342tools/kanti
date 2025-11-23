<script lang="ts">
  import { scopeStore } from '$lib/stores/scope';
  import proxyStore from '$lib/stores/proxy';

  export let standalone = false;
  
  // Search/filter state
  let searchText = '';
  let useRegex = false;
  let statusFilter = '';
  let scopeOnly = false;
  let showFilterOptions = false;

  // Proxy store
  const proxyStatus = $proxyStore.status;
  const proxySettings = $proxyStore.settings;

  // Clear requests
  async function clearRequests() {
    await proxyStore.clearRequests();
  }

  // Toggle filter options visibility
  function toggleFilterOptions() {
    showFilterOptions = !showFilterOptions;
  }
</script>

<div class="request-controls">
  <div class="control-group">
    <div class="status-indicator" class:running={proxyStatus.isRunning} 
         title={proxyStatus.isRunning ? 'Proxy Running' : 'Proxy Stopped'}>
      <div class="status-dot"></div>
      <span>Proxy {proxyStatus.isRunning ? 'Running' : 'Stopped'}</span>
    </div>
    
    <div class="control-buttons">
      <button class="control-button" on:click={clearRequests}>
        Clear Requests
      </button>
      <div class="filter-dropdown">
        <button class="control-button" 
                class:active={scopeOnly || searchText || statusFilter} 
                on:click={toggleFilterOptions}>
          Filter {(scopeOnly || searchText || statusFilter) ? '(Active)' : ''}
          <span class="dropdown-arrow">▼</span>
        </button>
        
        {#if showFilterOptions}
          <div class="dropdown-content">
            <div class="search-filter">
              <input 
                type="text" 
                placeholder="Search requests..." 
                bind:value={searchText} 
                class="search-input"
              />
              <label class="regex-option">
                <input type="checkbox" bind:checked={useRegex}>
                <span>Use regex</span>
              </label>
            </div>
            
            <div class="status-filter">
              <input 
                type="text" 
                placeholder="Filter by status code" 
                bind:value={statusFilter} 
                class="status-input"
              />
            </div>
            
            <label class="filter-option">
              <input type="checkbox" bind:checked={scopeOnly}>
              <span>In scope items only</span>
            </label>
          </div>
        {/if}
      </div>
    </div>
  </div>
</div>

<style>
  .request-controls {
    padding: 8px 15px;
    background-color: var(--bg-primary);
    border-radius: 4px;
    margin-bottom: 10px;
    display: flex;
    flex-direction: column;
    gap: 8px;
    box-shadow: 0 4px 8px var(--shadow-md);
    border: 1px solid var(--border-primary);
  }

  .control-group {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }
  
  .status-indicator {
    display: flex;
    align-items: center;
    gap: 10px;
    font-weight: bold;
    max-width: 300px;
    white-space: nowrap;
    text-overflow: ellipsis;
    color: var(--text-primary);
  }
  
  .status-dot {
    width: 12px;
    height: 12px;
    border-radius: 50%;
    background-color: var(--text-muted);
  }
  
  .status-indicator.running .status-dot {
    background-color: var(--accent-primary);
    box-shadow: 0 0 8px var(--accent-primary);
  }
  
  .control-buttons {
    display: flex;
    gap: 10px;
    max-width: 300px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  
  .control-button {
    padding: 4px 14px;
    background-color: var(--bg-secondary);
    border: none;
    border-radius: 4px;
    color: var(--text-primary);
    cursor: pointer;
    transition: background-color 0.2s ease;
    border: 1px solid var(--border-primary);
  }
  
  .control-button:hover {
    background-color: var(--bg-hover);
  }
  
  .control-button.active {
    background-color: var(--accent-primary);
    color: white;
  }
  
  .dropdown-content {
    width: 250px;
    padding: 10px;
  }

  /* Search filter styles */
  .search-filter, .status-filter {
    margin-bottom: 8px;
    padding: 5px;
  }
  
  .search-input, .status-input {
    width: 100%;
    padding: 6px 8px;
    background-color: var(--input-bg);
    border: 1px solid var(--input-border);
    border-radius: 4px;
    color: var(--text-primary);
    margin-bottom: 4px;
  }
  
  .search-input:focus, .status-input:focus {
    border-color: var(--input-focus);
    outline: none;
  }
  
  .regex-option {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-top: 4px;
    font-size: 12px;
    color: var(--text-secondary);
  }
</style>
