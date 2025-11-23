<script lang="ts">
  import { onMount } from 'svelte';
  import { fade } from 'svelte/transition';
  import { marked } from 'marked';
  import type { MarkedOptions } from 'marked';
  import hljs from 'highlight.js';
  import 'highlight.js/styles/atom-one-dark.css';
  import { browser } from '$app/environment';
  import { clickOutside } from '$lib/actions/clickOutside';
  
  // Import our stores
  import { apiKeys, currentProvider, currentModel, modelConfigs, defaultModelConfigs, type Provider } from '$lib/stores/settings';
  import { chatStore, type Message, type Conversation } from '$lib/stores/chat';
  import { projectState } from '$lib/stores/project';

  // Props
  export let standalone = false;

  // Local state
  let input = '';
  let isLoading = false;
  let chatContainer: HTMLElement | null = null;
  let isSettingsOpen = false;
  let newChatName = '';
  let isEditingTitle = false;
  let activeConversation: Conversation | null = null;
  let isSidebarOpen = true;
  let editingConversationId: string | null = null;
  let editingConversationName = '';

  // Define a proper type for the highlight function
  type HighlightFunction = (code: string, lang: string) => string;

  // Create properly typed options
  const markedOptions: MarkedOptions & { highlight?: HighlightFunction } = {
    highlight: function(code: string, lang: string) {
      if (lang && hljs.getLanguage(lang)) {
        return hljs.highlight(code, { language: lang }).value;
      }
      return hljs.highlightAuto(code).value;
    },
    breaks: true
  };

  // Set the options
  marked.setOptions(markedOptions);

  // Subscribe to get the active conversation
  chatStore.activeConversation.subscribe((conversation) => {
    activeConversation = conversation;
  });

  onMount(() => {
    // Initialize chat container ref
    if (browser) {
      // If no active conversation, create one
      if (!activeConversation && $chatStore.conversations.length === 0) {
        chatStore.createNewConversation();
      }
    }
  });

  // Submit message to the selected AI provider
  async function handleSubmit(): Promise<void> {
    if (!input.trim() || isLoading) return;
    
    // Check if API key is set
    if (!$apiKeys[$currentProvider]) {
      chatStore.addMessage({
        role: 'system',
        content: `Error: No API key set for ${$currentProvider}. Please configure your API key in settings.`,
        timestamp: new Date()
      });
      return;
    }

    const userMessage: Message = {
      role: 'user',
      content: input,
      timestamp: new Date()
    };
    
    chatStore.addMessage(userMessage);
    const userInput = input;
    input = '';
    isLoading = true;
    
    try {
      // Get chat history to send
      const history = activeConversation?.messages.slice(0, -1) || [];
      
      // Send request to backend
      const response = await fetch('/api/chat', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({
          message: userInput,
          provider: $currentProvider,
          apiKey: $apiKeys[$currentProvider],
          model: $currentModel,
          history: history
        })
      });
      
      if (!response.ok) {
        throw new Error(`API request failed with status ${response.status}`);
      }
      
      const data = await response.json();
      
      const assistantMessage: Message = {
        role: 'assistant',
        content: data.response,
        timestamp: new Date()
      };
      
      chatStore.addMessage(assistantMessage);
      
      // Scroll to bottom
      setTimeout(() => {
        if (chatContainer) {
          chatContainer.scrollTop = chatContainer.scrollHeight;
        }
      }, 100);
      
    } catch (error: unknown) {
      console.error('Error calling AI API:', error);
      chatStore.addMessage({
        role: 'system',
        content: `Error: Failed to get response from ${$currentProvider}. ${error instanceof Error ? error.message : 'Unknown error'}`,
        timestamp: new Date()
      });
    } finally {
      isLoading = false;
    }
  }

  // Create a new conversation
  function createNewChat(): void {
    const name = newChatName.trim() || `Chat ${$chatStore.conversations.length + 1}`;
    chatStore.createNewConversation(name);
    newChatName = '';
  }

  // Start editing a conversation title
  function startEditConversation(id: string, name: string): void {
    editingConversationId = id;
    editingConversationName = name;
  }

  // Save conversation title edit
  function saveConversationEdit(): void {
    if (editingConversationId && editingConversationName.trim()) {
      chatStore.renameConversation(editingConversationId, editingConversationName.trim());
    }
    editingConversationId = null;
  }

  // Cancel conversation title edit
  function cancelConversationEdit(): void {
    editingConversationId = null;
  }

  // Delete the current conversation
  function deleteCurrentChat(): void {
    chatStore.clearActiveConversation();
  }

  // Toggle sidebar visibility
  function toggleSidebar(): void {
    isSidebarOpen = !isSidebarOpen;
  }

  // Toggle settings panel
  function toggleSettings(): void {
    isSettingsOpen = !isSettingsOpen;
  }

  // Update API key
  function updateApiKey(provider: Provider, value: string): void {
    apiKeys.update(keys => ({
      ...keys,
      [provider]: value
    }));
  }

  // Change the current provider
  function changeProvider(provider: Provider): void {
    currentProvider.set(provider);
  }
  
  // Change the current model
  function changeModel(newModel: string): void {
    currentModel.set(newModel);
  }

  // Handle key press events
  function handleKeydown(event: KeyboardEvent): void {
    if (event.key === 'Enter' && !event.shiftKey) {
      event.preventDefault();
      handleSubmit();
    }
  }

  // Format timestamp
  function formatTime(date: Date): string {
    return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  }

  // Format date
  function formatDate(date: Date): string {
    return date.toLocaleDateString([], { month: 'short', day: 'numeric' });
  }

  // Process message content with markdown
  function processContent(content: string): string {
    return marked(content) as string;
  }

  // Handle input change for API keys
  function handleInputChange(event: Event, provider: Provider): void {
    const target = event.target as HTMLInputElement;
    if (target) {
      updateApiKey(provider, target.value);
    }
  }

  // Select a conversation
  function selectConversation(id: string): void {
    chatStore.setActiveConversation(id);
  }

  // Get truncated preview of last message in a conversation
  function getConversationPreview(conversation: Conversation): string {
    if (conversation.messages.length === 0) return "No messages";
    const lastMessage = conversation.messages[conversation.messages.length - 1];
    const content = lastMessage.content;
    return content.length > 40 ? content.substring(0, 40) + "..." : content;
  }

  // Auto-scroll when messages change
  $: if (activeConversation?.messages && chatContainer && browser) {
    setTimeout(() => {
      if (chatContainer) {
        chatContainer.scrollTop = chatContainer.scrollHeight;
      }
    }, 0);
  }
</script>

<div class="chat-terminal {standalone ? 'standalone' : ''}">
  <div class="terminal-layout {isSidebarOpen ? 'with-sidebar' : 'sidebar-collapsed'}">
    <div class="sidebar" class:hidden={!isSidebarOpen}>
      <div class="sidebar-header">
        <h3>Conversations</h3>
        <button class="new-chat-btn" on:click={createNewChat}>
          <span class="icon">+</span>
          <span class="label">New Chat</span>
        </button>
      </div>
      
      <div class="conversations-list">
        {#if $chatStore.conversations.length === 0}
          <div class="empty-state">No conversations yet</div>
        {:else}
          {#each $chatStore.conversations as conversation (conversation.id)}
            <div 
              class="conversation-item {$chatStore.activeConversationId === conversation.id ? 'active' : ''}"
              on:click={() => selectConversation(conversation.id)}
            >
              {#if editingConversationId === conversation.id}
                <!-- svelte-ignore a11y-click-events-have-key-events -->
                <div class="edit-title-container" use:clickOutside={saveConversationEdit}>
                  <input 
                    type="text" 
                    bind:value={editingConversationName} 
                    on:keydown={(e) => e.key === 'Enter' && saveConversationEdit()}
                    on:blur={saveConversationEdit}
                    autofocus
                  />
                </div>
              {:else}
                <div class="conversation-info">
                  <div class="conversation-title" on:dblclick={() => startEditConversation(conversation.id, conversation.name)}>
                    {conversation.name}
                  </div>
                  <div class="conversation-preview">
                    {getConversationPreview(conversation)}
                  </div>
                  <div class="conversation-date">
                    {formatDate(conversation.updatedAt)}
                  </div>
                </div>
              {/if}
            </div>
          {/each}
        {/if}
      </div>
    </div>
    
    <div class="chat-content">
      <div class="terminal-header">
        <div class="header-left">
          <button class="toggle-sidebar-btn" on:click={toggleSidebar}>
            {isSidebarOpen ? '◀' : '▶'}
          </button>
          <div class="terminal-title">
            {activeConversation ? activeConversation.name : 'Start a new chat'} - {$currentProvider.toUpperCase()} ({$currentModel})
          </div>
        </div>
        <div class="terminal-controls">
          <button class="terminal-btn" on:click={toggleSettings}>⚙️</button>
          <button class="terminal-btn" on:click={deleteCurrentChat}>🗑️</button>
        </div>
      </div>
      
      {#if isSettingsOpen}
        <div class="settings-panel" transition:fade={{ duration: 150 }}>
          <h3>API Settings</h3>
          <div class="provider-selector">
            {#each Object.keys($apiKeys) as provider}
              <button 
                class="provider-btn {$currentProvider === provider ? 'active' : ''}" 
                on:click={() => changeProvider(provider as Provider)}
              >
                {provider}
              </button>
            {/each}
          </div>
          
          <div class="model-selector">
            <label for="model-select">Model:</label>
            <select 
              id="model-select" 
              bind:value={$currentModel}
              on:change={(e) => {
                const select = e.target as HTMLSelectElement;
                changeModel(select.value);
              }}
            >
              {#if $modelConfigs && $modelConfigs[$currentProvider]}
                {#each $modelConfigs[$currentProvider].models as model}
                  <option value={model}>{model}</option>
                {/each}
              {:else}
                <!-- Fallback to default models if modelConfigs is missing -->
                {#each defaultModelConfigs[$currentProvider].models as model}
                  <option value={model}>{model}</option>
                {/each}
              {/if}
            </select>
          </div>
          
          <div class="api-key-inputs">
            {#each Object.entries($apiKeys) as [provider, key]}
              <div class="api-key-input">
                <label for="{provider}-key">{provider} API Key:</label>
                <input 
                  type="password" 
                  id="{provider}-key" 
                  value={key} 
                  on:change={(e) => handleInputChange(e, provider as Provider)}
                  placeholder="Enter your API key"
                />
              </div>
            {/each}
          </div>
        </div>
      {/if}
      
      <div class="chat-container" bind:this={chatContainer}>
        {#if !activeConversation || activeConversation.messages.length === 0}
          <div class="welcome-message">
            <p>Welcome to the AI Chat Terminal!</p>
            <p>Choose your AI provider and start chatting.</p>
          </div>
        {:else}
          {#each activeConversation.messages as message, i (i)}
            <div class="message {message.role}">
              <div class="message-header">
                <span class="message-role">{message.role === 'assistant' ? $currentProvider : message.role}</span>
                <span class="message-time">{formatTime(message.timestamp)}</span>
              </div>
              <div class="message-content">
                {@html processContent(message.content)}
              </div>
            </div>
          {/each}
        {/if}
        
        {#if isLoading}
          <div class="loading-indicator">
            <span class="dot"></span>
            <span class="dot"></span>
            <span class="dot"></span>
          </div>
        {/if}
      </div>
      
      <div class="input-container">
        <textarea 
          bind:value={input} 
          on:keydown={handleKeydown} 
          placeholder="Type your message here... (Shift+Enter for new line)"
          rows="1"
          disabled={isLoading}
        ></textarea>
        <button class="send-btn" on:click={handleSubmit} disabled={isLoading || !input.trim()}>
          Send
        </button>
      </div>
    </div>
  </div>
</div>

<style>
  .chat-terminal {
    display: flex;
    flex-direction: column;
    color: var(--text-primary);
    border-radius: 4px;
    height: 100%;
    width: 100%;
    overflow: hidden;
    font-family: 'Fira Code', 'Cascadia Code', 'Consolas', monospace;
    background-color: var(--bg-primary);
  }
  
  .chat-terminal.standalone {
    height: calc(100vh - 60px);
    border-radius: 0;
  }
  
  .terminal-layout {
    display: flex;
    height: 100%;
    width: 100%;
  }
  
  .terminal-layout.with-sidebar .chat-content {
    width: calc(100% - 280px);
  }
  
  .terminal-layout.sidebar-collapsed .chat-content {
    width: 100%;
  }
  
  .sidebar {
    width: 280px;
    background-color: var(--bg-secondary);
    display: flex;
    flex-direction: column;
    height: 100%;
    transition: width 0.2s ease;
    border-right: 1px solid var(--border-primary);
  }
  
  .sidebar.hidden {
    width: 0;
    overflow: hidden;
  }
  
  .sidebar-header {
    padding: 15px;
    border-bottom: 1px solid var(--border-primary);
    display: flex;
    justify-content: space-between;
    align-items: center;
  }
  
  .sidebar-header h3 {
    margin: 0;
    font-size: 16px;
    color: var(--text-primary);
  }
  
  .new-chat-btn {
    background-color: var(--bg-tertiary);
    border: 1px solid var(--border-primary);
    color: var(--text-primary);
    border-radius: 4px;
    padding: 5px 10px;
    cursor: pointer;
    display: flex;
    align-items: center;
    gap: 5px;
    transition: background-color 0.2s;
  }
  
  .new-chat-btn:hover {
    background-color: var(--bg-hover);
  }
  
  .conversations-list {
    flex: 1;
    overflow-y: auto;
    padding: 10px;
  }
  
  .conversation-item {
    padding: 10px;
    border-radius: 4px;
    margin-bottom: 8px;
    cursor: pointer;
    transition: background-color 0.2s;
    border: 1px solid var(--border-secondary);
  }
  
  .conversation-item:hover {
    background-color: var(--bg-hover);
  }
  
  .conversation-item.active {
    background-color: var(--bg-active);
  }
  
  .conversation-info {
    display: flex;
    flex-direction: column;
    gap: 5px;
  }
  
  .conversation-title {
    font-weight: bold;
    color: var(--text-primary);
  }
  
  .conversation-preview {
    font-size: 12px;
    color: var(--text-secondary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  
  .conversation-date {
    font-size: 10px;
    color: var(--text-tertiary);
  }
  
  .edit-title-container {
    width: 100%;
  }
  
  .edit-title-container input {
    width: 100%;
    background-color: var(--input-bg);
    border: 1px solid var(--input-border);
    color: var(--text-primary);
    padding: 5px;
    font-size: 13px;
    border-radius: 3px;
  }
  
  .empty-state {
    text-align: center;
    padding: 30px 0;
    color: var(--text-muted);
    font-style: italic;
  }
  
  .chat-content {
    flex: 1;
    display: flex;
    flex-direction: column;
    height: 100%;
    background-color: var(--bg-primary);
  }
  
  .terminal-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 10px 15px;
    border-bottom: 1px solid var(--border-primary);
    background-color: var(--bg-secondary);
  }
  
  .header-left {
    display: flex;
    align-items: center;
    gap: 10px;
  }
  
  .toggle-sidebar-btn {
    background: none;
    border: none;
    color: var(--text-primary);
    cursor: pointer;
    font-size: 14px;
    padding: 5px;
    border-radius: 3px;
    transition: background-color 0.2s;
  }
  
  .toggle-sidebar-btn:hover {
    background-color: var(--bg-hover);
  }
  
  .terminal-title {
    font-weight: bold;
    color: var(--text-primary);
  }
  
  .terminal-controls {
    display: flex;
    gap: 5px;
  }
  
  .terminal-btn {
    background: none;
    border: none;
    color: var(--text-primary);
    cursor: pointer;
    padding: 5px 8px;
    border-radius: 3px;
    transition: background-color 0.2s;
  }
  
  .terminal-btn:hover {
    background-color: var(--bg-hover);
  }
  
  .settings-panel {
    background-color: var(--bg-secondary);
    border-bottom: 1px solid var(--border-primary);
    padding: 15px;
  }
  
  .settings-panel h3 {
    margin: 0 0 15px 0;
    color: var(--text-primary);
    font-size: 16px;
  }
  
  .provider-selector {
    display: flex;
    gap: 10px;
    margin-bottom: 15px;
  }
  
  .provider-btn {
    background-color: var(--bg-tertiary);
    border: 1px solid var(--border-primary);
    color: var(--text-primary);
    padding: 8px 12px;
    border-radius: 4px;
    cursor: pointer;
    transition: background-color 0.2s;
  }
  
  .provider-btn.active {
    background-color: var(--accent-primary);
    color: white;
  }
  
  .provider-btn:hover:not(.active) {
    background-color: var(--bg-hover);
  }
  
  .model-selector {
    display: flex;
    align-items: center;
    gap: 10px;
    margin-bottom: 15px;
  }
  
  .model-selector label {
    color: var(--text-primary);
    font-weight: bold;
  }
  
  .model-selector select {
    background-color: var(--input-bg);
    border: 1px solid var(--input-border);
    color: var(--text-primary);
    padding: 5px 8px;
    border-radius: 3px;
  }
  
  .api-key-inputs {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }
  
  .api-key-input {
    display: flex;
    flex-direction: column;
    gap: 5px;
  }
  
  .api-key-input label {
    color: var(--text-primary);
    font-size: 14px;
    font-weight: bold;
  }
  
  .api-key-input input {
    background-color: var(--input-bg);
    border: 1px solid var(--input-border);
    color: var(--text-primary);
    padding: 8px;
    border-radius: 4px;
    font-family: monospace;
  }
  
  .chat-container {
    flex: 1;
    overflow-y: auto;
    padding: 20px;
    display: flex;
    flex-direction: column;
    gap: 20px;
  }
  
  .welcome-message {
    text-align: center;
    color: var(--text-muted);
    font-style: italic;
    margin-top: 50px;
  }
  
  .message {
    border-radius: 8px;
    padding: 15px;
    max-width: 80%;
    word-wrap: break-word;
  }
  
  .message.user {
    background-color: var(--accent-primary);
    color: white;
    align-self: flex-end;
    margin-left: auto;
  }
  
  .message.assistant {
    background-color: var(--bg-secondary);
    color: var(--text-primary);
    align-self: flex-start;
    border: 1px solid var(--border-primary);
  }
  
  .message.system {
    background-color: var(--bg-tertiary);
    color: var(--text-secondary);
    align-self: center;
    max-width: 90%;
    font-style: italic;
  }
  
  .message-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 8px;
    font-size: 12px;
  }
  
  .message-role {
    font-weight: bold;
    text-transform: capitalize;
  }
  
  .message-time {
    color: var(--text-tertiary);
  }
  
  .message-content {
    line-height: 1.5;
  }
  
  .message-content :global(pre) {
    background-color: var(--bg-tertiary);
    border: 1px solid var(--border-primary);
    border-radius: 4px;
    padding: 10px;
    overflow-x: auto;
    margin: 10px 0;
  }
  
  .message-content :global(code) {
    background-color: var(--bg-tertiary);
    padding: 2px 4px;
    border-radius: 3px;
    font-family: 'Fira Code', monospace;
  }
  
  .message-content :global(blockquote) {
    border-left: 3px solid var(--accent-primary);
    margin: 10px 0;
    padding-left: 15px;
    color: var(--text-secondary);
  }
  
  .loading-indicator {
    display: flex;
    justify-content: center;
    gap: 5px;
    padding: 20px;
  }
  
  .dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background-color: var(--accent-primary);
    animation: bounce 1.4s infinite ease-in-out both;
  }
  
  .dot:nth-child(1) { animation-delay: -0.32s; }
  .dot:nth-child(2) { animation-delay: -0.16s; }
  
  @keyframes bounce {
    0%, 80%, 100% { 
      transform: scale(0);
    } 40% { 
      transform: scale(1.0);
    }
  }
  
  .input-container {
    display: flex;
    gap: 10px;
    padding: 15px;
    border-top: 1px solid var(--border-primary);
    background-color: var(--bg-secondary);
  }
  
  .input-container textarea {
    flex: 1;
    background-color: var(--input-bg);
    border: 1px solid var(--input-border);
    color: var(--text-primary);
    padding: 10px;
    border-radius: 4px;
    resize: none;
    font-family: inherit;
    min-height: 40px;
    max-height: 120px;
  }
  
  .input-container textarea:focus {
    outline: none;
    border-color: var(--accent-primary);
  }
  
  .input-container textarea:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }
  
  .send-btn {
    background-color: var(--accent-primary);
    border: none;
    color: white;
    padding: 10px 20px;
    border-radius: 4px;
    cursor: pointer;
    transition: background-color 0.2s;
  }
  
  .send-btn:hover:not(:disabled) {
    background-color: var(--accent-hover);
  }
  
  .send-btn:disabled {
    background-color: var(--bg-tertiary);
    color: var(--text-muted);
    cursor: not-allowed;
  }
</style>
