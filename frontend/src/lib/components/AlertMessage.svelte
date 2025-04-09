<script lang="ts">
    import { createEventDispatcher, onMount } from 'svelte';
    import { fade } from 'svelte/transition';
    
    export let type: 'success' | 'error' | 'info' | 'warning' = 'info';
    export let message = '';
    export let dismissible = true;
    export let autoDismiss = false;
    export let autoDismissTimeout = 5000;
    export let show = true;
    
    const dispatch = createEventDispatcher<{
        dismiss: void;
    }>();
    
    let timeoutId: number;
    
    function dismiss() {
        show = false;
        dispatch('dismiss');
    }
    
    onMount(() => {
        if (autoDismiss && show) {
            timeoutId = window.setTimeout(dismiss, autoDismissTimeout);
        }
        
        return () => {
            if (timeoutId) {
                clearTimeout(timeoutId);
            }
        };
    });
    
    // Icon based on alert type
    $: icon = type === 'success' ? '✓' :
              type === 'error' ? '✕' :
              type === 'warning' ? '⚠' : 'ℹ';
</script>

{#if show}
    <div class="alert alert-{type}" transition:fade={{ duration: 200 }}>
        <div class="alert-icon">{icon}</div>
        <div class="alert-content">{message}</div>
        {#if dismissible}
            <button class="alert-dismiss" on:click={dismiss}>×</button>
        {/if}
    </div>
{/if}

<style>
    .alert {
        display: flex;
        align-items: center;
        padding: 1rem;
        border-radius: 4px;
        margin-bottom: 1rem;
    }
    
    .alert-success {
        background-color: #dcfce7;
        color: #16a34a;
        border-left: 4px solid #16a34a;
    }
    
    .alert-error {
        background-color: #fee2e2;
        color: #dc2626;
        border-left: 4px solid #dc2626;
    }
    
    .alert-info {
        background-color: #dbeafe;
        color: #2563eb;
        border-left: 4px solid #2563eb;
    }
    
    .alert-warning {
        background-color: #fef3c7;
        color: #d97706;
        border-left: 4px solid #d97706;
    }
    
    .alert-icon {
        font-size: 1.25rem;
        margin-right: 0.75rem;
    }
    
    .alert-content {
        flex: 1;
    }
    
    .alert-dismiss {
        background: none;
        border: none;
        font-size: 1.25rem;
        cursor: pointer;
        color: inherit;
        opacity: 0.7;
        padding: 0;
        display: flex;
        align-items: center;
        justify-content: center;
        width: 1.5rem;
        height: 1.5rem;
    }
    
    .alert-dismiss:hover {
        opacity: 1;
    }
</style>