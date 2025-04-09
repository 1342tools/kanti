<script lang="ts">
    import { createEventDispatcher } from 'svelte';
    import { domainsApi, HttpError } from '$lib/api/api'; // Import HttpError
    import type { RootDomain } from '$lib/types';
    
    // Prop to receive the organization ID from the parent component
    export let organizationId: number; 

    const dispatch = createEventDispatcher<{
        domainAdded: RootDomain;
    }>();
    
    let domain = '';
    let loading = false;
    let error = '';
    let success = '';
    
    async function handleSubmit() {
        if (!domain) {
            error = 'Please enter a domain';
            return;
        }
        
        loading = true;
        error = '';
        success = '';
        
        try {
            // Pass organizationId when creating the domain
            const newDomain = await domainsApi.createDomain({ domain, organization_id: organizationId }); 
            domain = ''; // Clear input on success
            success = `Domain ${newDomain.domain} added successfully`;
            dispatch('domainAdded', newDomain);
            // Optionally clear success message after a delay
            setTimeout(() => success = '', 3000); 
        } catch (err) {
            if (err instanceof HttpError && err.status === 409) {
                // Specific message for duplicate domain conflict
                error = err.message; // Use the message from the backend (e.g., "Domain 'x' already exists...")
            } else if (err instanceof Error) {
                 // Handle other errors (network, server errors, etc.)
                error = err.message;
            } else {
                // Fallback for unknown errors
                error = 'An unexpected error occurred while adding the domain.';
            }
        } finally {
            loading = false;
        }
    }
</script>

<form on:submit|preventDefault={handleSubmit} class="domain-form">
    <div class="form-group">
        <label for="domain">Domain Name</label>
        <input 
            type="text" 
            id="domain" 
            bind:value={domain} 
            placeholder="example.com" 
            disabled={loading}
        />
    </div>
    
    <button type="submit" class="btn" disabled={loading}>
        {loading ? 'Adding...' : 'Add Domain'}
    </button>
    
    {#if error}
        <p class="error">{error}</p>
    {/if}
    
    {#if success}
        <p class="success">{success}</p>
    {/if}
</form>

<style>
    .domain-form {
        background-color: var(--card-bg); /* Use variable */
        padding: 1.5rem;
        border-radius: 8px;
        max-width: 500px;
        border: 1px solid var(--border); /* Add border */
    }
    
    .form-group {
        margin-bottom: 1rem;
    }
    
    label {
        display: block;
        margin-bottom: 0.5rem;
        font-weight: 500;
        color: var(--text); /* Use variable */
    }
    
    input {
        width: 100%;
        padding: 0.75rem;
        border: 1px solid var(--border); /* Use variable */
        background-color: var(--background); /* Use variable */
        color: var(--text); /* Use variable */
        border-radius: 4px;
        font-size: 1rem;
    }
    input::placeholder {
        color: var(--text-light); /* Use variable */
    }
    input:disabled {
        background-color: var(--card-bg); /* Use variable */
        opacity: 0.7;
    }
    
    /* Global .btn styles are in app.css */
    /* .btn { ... } */
    
    .btn:disabled {
        background-color: var(--text-light); /* Use variable for disabled state */
        opacity: 0.6;
        cursor: not-allowed;
    }
    
    .error {
        color: #ef4444; /* Keep error color */
        margin-top: 1rem;
    }
    
    .success {
        color: #22c55e; /* Keep success color */
        margin-top: 1rem;
    }
</style>
