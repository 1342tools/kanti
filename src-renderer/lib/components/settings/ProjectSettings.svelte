<script lang="ts">
  import { onMount } from 'svelte';
  import { projectState } from '$lib/stores/project';
  
  // Project state
  let activeProject: Project | null = null;
  let projectName = 'Untitled Project';
  let editingName = false;
  let projectSaving = false;
  let projectError: string | null = null;
  
  // Function to save the current project
  async function saveProject() {
    if (!activeProject) {
      projectError = 'No active project to save';
      return;
    }
    
    try {
      projectSaving = true;
      projectError = null;
      
      const result = await projectState.save();
      
      if (!result.success) {
        projectError = result.error || 'Failed to save project';
      }
    } catch (err: any) {
      console.error('Error saving project:', err);
      projectError = err?.message || 'Failed to save project';
    } finally {
      projectSaving = false;
    }
  }
  
  // Function to save the current project with a new name/location
  async function saveProjectAs() {
    if (!activeProject) {
      projectError = 'No active project to save';
      return;
    }
    
    try {
      projectSaving = true;
      projectError = null;
      
      const result = await projectState.saveAs();
      
      if (!result.success) {
        projectError = result.error || 'Failed to save project';
      }
    } catch (err: any) {
      console.error('Error saving project as:', err);
      projectError = err?.message || 'Failed to save project';
    } finally {
      projectSaving = false;
    }
  }
  
  // Function to update project name
  function updateProjectName() {
    if (!activeProject) return;
    
    editingName = false;
    
    if (projectName.trim() === '') {
      projectName = 'Untitled Project';
    }
    
    if (projectName !== activeProject.name) {
      activeProject.name = projectName;
      projectState.update(activeProject);
      saveProject();
    }
  }
  
  // Function to handle Enter key press when editing name
  function handleKeyDown(event: KeyboardEvent) {
    if (event.key === 'Enter') {
      updateProjectName();
    } else if (event.key === 'Escape') {
      editingName = false;
      if (activeProject) {
        projectName = activeProject.name || 'Untitled Project';
      }
    }
  }
  
  // Initialize the component
  onMount(() => {
    // Subscribe to project state changes
    const unsubscribe = projectState.subscribe(project => {
      activeProject = project;
      if (activeProject) {
        projectName = activeProject.name || 'Untitled Project';
      }
    });
    
    // Listen for project state change events from the main process
    if (window.electronAPI) {
      window.electronAPI.receive('project-state-changed', (data) => {
        if (data && data.project) {
          projectState.initialize(data.project);
        }
      });
    }
    
    // Return cleanup function
    return () => {
      unsubscribe();
    };
  });
</script>

<div class="project-settings">
  <div class="form-group">
    <h3>Project Information</h3>
    
    <div class="project-info">
      {#if activeProject}
        <div class="info-row">
          <span class="label">Name:</span>
          {#if editingName}
            <div class="edit-name-container">
              <input 
                type="text" 
                bind:value={projectName} 
                on:blur={updateProjectName}
                on:keydown={handleKeyDown}
                class="name-input"
                autofocus
              />
              <button class="save-name-btn" on:click={updateProjectName}>
                Save
              </button>
            </div>
          {:else}
            <div class="editable-value">
              <span class="value">{projectName}</span>
              <button class="edit-btn" on:click={() => editingName = true}>
                ✏️
              </button>
            </div>
          {/if}
        </div>
        <div class="info-row">
        </div>
      {:else}
        <p class="no-project">No active project</p>
      {/if}
    </div>
  </div>
  
  <div class="form-group">
    <h3>Project Actions</h3>
    
    <div class="actions-grid">
      <button 
        class="action-button" 
        on:click={() => {
          window.electronAPI?.project.initialize()
        }}
      >
        <span class="icon">📄</span>
        <span>New Project</span>
      </button>
      
      <button 
        class="action-button" 
        on:click={() => window.electronAPI?.project.open()}
      >
        <span class="icon">📂</span>
        <span>Open Project</span>
      </button>
      
      <button 
        class="action-button" 
        class:disabled={!activeProject} 
        on:click={saveProject}
      >
        <span class="icon">💾</span>
        <span>Save Project</span>
      </button>
      
      <button 
        class="action-button" 
        class:disabled={!activeProject} 
        on:click={saveProjectAs}
      >
        <span class="icon">📋</span>
        <span>Save As...</span>
      </button>
    </div>
    
    {#if projectError}
      <div class="project-error">
        <span class="error-icon">⚠️</span>
        <span>{projectError}</span>
      </div>
    {/if}
    
    {#if projectSaving}
      <div class="project-saving">
        <span class="spinner"></span>
        <span>Saving...</span>
      </div>
    {/if}
  </div>
</div>

<style>
  .project-settings {
    display: flex;
    flex-direction: column;
    gap: 20px;
  }
  
  h3 {
    margin-top: 0;
    margin-bottom: 15px;
    color: var(--text-primary);
    font-size: 18px;
  }
  
  .project-info {
    background-color: var(--bg-secondary);
    border-radius: 8px;
    padding: 15px;
    border: 1px solid var(--border-primary);
  }
  
  .info-row {
    display: flex;
    margin-bottom: 10px;
  }
  
  .info-row:last-child {
    margin-bottom: 0;
  }
  
  .label {
    width: 100px;
    color: var(--text-secondary);
  }
  
  .value {
    flex: 1;
    color: var(--text-primary);
    word-break: break-all;
  }
  
  .editable-value {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: space-between;
  }
  
  .edit-btn {
    background: none;
    border: none;
    color: var(--text-secondary);
    cursor: pointer;
    font-size: 14px;
    padding: 2px 6px;
    margin-left: 8px;
    border-radius: 4px;
    transition: all 0.2s ease;
  }
  
  .edit-btn:hover {
    color: var(--text-primary);
    background-color: var(--bg-hover);
  }
  
  .edit-name-container {
    flex: 1;
    display: flex;
    gap: 8px;
  }
  
  .name-input {
    flex: 1;
    background-color: var(--input-bg);
    border: 1px solid var(--input-border);
    border-radius: 6px;
    color: var(--text-primary);
    padding: 6px 10px;
    font-size: 14px;
    transition: border-color 0.2s ease;
  }
  
  .name-input:focus {
    outline: none;
    border-color: var(--input-focus);
  }
  
  .save-name-btn {
    background-color: var(--accent-primary);
    color: white;
    border: none;
    border-radius: 6px;
    padding: 6px 12px;
    cursor: pointer;
    font-size: 14px;
    transition: background-color 0.2s ease;
  }
  
  .save-name-btn:hover {
    background-color: var(--accent-hover);
  }
  
  .no-project {
    color: var(--text-secondary);
    font-style: italic;
    text-align: center;
    margin: 10px 0;
  }
  
  .actions-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
    gap: 15px;
    margin-bottom: 15px;
  }
  
  .action-button {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    background-color: var(--bg-secondary);
    border: none;
    color: var(--text-primary);
    padding: 15px;
    border-radius: 8px;
    border: 1px solid var(--border-primary);
    cursor: pointer;
    transition: all 0.2s ease;
    text-align: center;
    gap: 8px;
  }
  
  .action-button:hover {
    background-color: var(--bg-hover);
    border-color: var(--border-hover);
    transform: translateY(-2px);
  }
  
  .action-button.disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
  
  .action-button.disabled:hover {
    background-color: var(--bg-secondary);
    border-color: var(--border-primary);
    transform: none;
  }
  
  .icon {
    font-size: 24px;
  }
  
  .project-error {
    background-color: rgba(220, 53, 69, 0.1);
    color: #dc3545;
    padding: 10px;
    border-radius: 8px;
    margin-top: 15px;
    display: flex;
    align-items: center;
    gap: 8px;
    border: 1px solid rgba(220, 53, 69, 0.2);
  }
  
  .error-icon {
    font-size: 18px;
  }
  
  .project-saving {
    display: flex;
    align-items: center;
    gap: 10px;
    color: var(--text-secondary);
    margin-top: 15px;
  }
  
  .spinner {
    width: 18px;
    height: 18px;
    border: 2px solid transparent;
    border-top-color: var(--text-primary);
    border-radius: 50%;
    animation: spin 1s linear infinite;
  }
  
  @keyframes spin {
    to { transform: rotate(360deg); }
  }
</style>
