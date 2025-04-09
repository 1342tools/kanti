<script lang="ts">
    import { onMount } from 'svelte';
    import { subdomainsApi } from '$lib/api/api';
    import type { Subdomain } from '$lib/types';
    import SubdomainList from '$lib/components/SubdomainList.svelte';
    
    let subdomains: Subdomain[] = [];
    let loading = true;
    let error = '';
    let searchQuery = '';
    
    onMount(async () => {
        try {
            subdomains = await subdomainsApi.getSubdomains();
            loading = false;
        } catch (err) {
            error = err instanceof Error ? err.message : 'Failed to load subdomains';
            loading = false;
        }
    });
    
    $: filteredSubdomains = subdomains.filter(subdomain => 
        subdomain.hostname.toLowerCase().includes(searchQuery.toLowerCase())
    );
</script>

<svelte:head>
    <title>Subdomains - Attack Surface Management</title>
</svelte:head>

<div class="subdomains-page">
    <header class="page-header">
        <h1>Subdomains</h1>
        <div class="search-container">
            <input 
                type="text" 
                placeholder="Search subdomains..." 
                bind:value={searchQuery} 
                class="search-input"
            />
        </div>
    </header>
    
    {#if loading}
        <p>Loading subdomains...</p>
    {:else if error}
        <p class="error">{error}</p>
    {:else}
        <div class="stats">
            <div class="stat-item">
                <span class="stat-value">{subdomains.length}</span>
                <span class="stat-label">Total Subdomains</span>
            </div>
            <div class="stat-item">
                <span class="stat-value">{subdomains.filter(s => s.is_active).length}</span>
                <span class="stat-label">Active</span>
            </div>
            <div class="stat-item">
                <span class="stat-value">{subdomains.filter(s => !s.is_active).length}</span>
                <span class="stat-label">Inactive</span>
            </div>
        </div>
        
        <div class="list-container">
            {#if filteredSubdomains.length === 0}
                <p class="empty-message">
                    {searchQuery ? 'No subdomains match your search query.' : 'No subdomains have been discovered yet.'}
                </p>
            {:else}
                <SubdomainList subdomains={filteredSubdomains} showDomainColumn={true} />
            {/if}
        </div>
    {/if}
</div>

<style>
    .subdomains-page {
        padding: 1rem 0;
    }
    .subdomains-page h1 {
        color: var(--text); /* Use variable */
    }
    
    .page-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 2rem;
    }
    
    .search-container {
        width: 300px;
    }
    
    .search-input {
        width: 100%;
        padding: 0.5rem 1rem;
        border: 1px solid var(--border); /* Use variable */
        background-color: var(--background); /* Use variable */
        color: var(--text); /* Use variable */
        border-radius: 4px;
        font-size: 0.875rem;
    }
    .search-input::placeholder {
        color: var(--text-light); /* Use variable */
    }
    
    .subdomain-stats {
        display: flex;
        gap: 2rem;
        margin-bottom: 2rem;
        background-color: var(--card-bg); /* Use variable */
        padding: 1.5rem;
        border-radius: 8px;
        border: 1px solid var(--border); /* Add border */
    }
    
    .stat-item {
        display: flex;
        flex-direction: column;
        align-items: center;
    }
    
    .stat-value {
        font-size: 1.5rem;
        font-weight: 600;
        color: var(--primary); /* Use variable */
    }
    
    .stat-label {
        color: var(--text-light); /* Use variable */
        font-size: 0.875rem;
    }
    
    .empty-message {
        padding: 2rem;
        background-color: var(--card-bg); /* Use variable */
        border-radius: 8px;
        text-align: center;
        color: var(--text-light); /* Use variable */
        border: 1px solid var(--border); /* Add border */
    }
    
    .error {
        color: #ef4444; /* Keep error color */
    }
</style>
