<script lang="ts">
    import { onMount } from 'svelte';
    import { domainsApi } from '$lib/api/api'; // Assuming domainsApi exists
    import type { RootDomain } from '$lib/types'; // Use RootDomain type
    import DomainList from '$lib/components/DomainList.svelte'; // Assuming DomainList component exists
    
    let domains: RootDomain[] = []; // Use RootDomain type
    let loading = true;
    let error = '';
    let searchQuery = '';
    
    onMount(async () => {
        try {
            domains = await domainsApi.getDomains(); // Use domainsApi
            loading = false;
        } catch (err) {
            error = err instanceof Error ? err.message : 'Failed to load domains';
            loading = false;
        }
    });
    
    // Filter based on domain name (using 'domain' property from RootDomain)
    $: filteredDomains = domains.filter(domain => 
        domain.domain.toLowerCase().includes(searchQuery.toLowerCase()) 
    );
</script>

<svelte:head>
    <title>Domains - Attack Surface Management</title> 
</svelte:head>

<div class="domains-page"> 
    <header class="page-header">
        <h1>Domains</h1> 
        <div class="search-container">
            <input 
                type="text" 
                placeholder="Search domains..." 
                bind:value={searchQuery} 
                class="search-input"
            />
        </div>
    </header>
    
    {#if loading}
        <p>Loading domains...</p> 
    {:else if error}
        <p class="error">{error}</p>
    {:else}
        <div class="stats"> 
            <div class="stat-item">
                <span class="stat-value">{domains.length}</span> 
                <span class="stat-label">Total Domains</span> 
            </div>
            <!-- Add more relevant stats for domains if needed -->
        </div>
        
        <div class="list-container"> 
            {#if filteredDomains.length === 0}
                <p class="empty-message">
                    {searchQuery ? 'No domains match your search query.' : 'No domains have been added yet.'} 
                </p>
            {:else}
                <DomainList domains={filteredDomains} /> 
            {/if}
        </div>
    {/if}
</div>

<style>
    /* Reuse styles, adjusting class names */
    .domains-page {
        padding: 1rem 0;
    }
    .domains-page h1 {
        color: var(--text); 
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
        border: 1px solid var(--border); 
        background-color: var(--background); 
        color: var(--text); 
        border-radius: 4px;
        font-size: 0.875rem;
    }
    .search-input::placeholder {
        color: var(--text-light); 
    }
    
    .stat-item {
        display: flex;
        flex-direction: column;
        align-items: center;
    }
    
    .stat-value {
        font-size: 1.5rem;
        font-weight: 600;
        color: var(--primary); 
    }
    
    .stat-label {
        color: var(--text-light); 
        font-size: 0.875rem;
    }

    
    .empty-message {
        padding: 2rem;
        background-color: var(--card-bg); 
        border-radius: 8px;
        text-align: center;
        color: var(--text-light); 
        border: 1px solid var(--border); 
    }
    
    .error {
        color: #ef4444; 
    }
</style>
