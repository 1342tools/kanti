<script lang="ts">
    import { onMount } from 'svelte';
    import { endpointsApi } from '$lib/api/api';
    import type { Endpoint } from '$lib/types';
    import EndpointList from '$lib/components/EndpointList.svelte';
    
    let endpoints: Endpoint[] = [];
    let loading = true;
    let error = '';
    let searchQuery = '';
    
    // Filter options
    let methodFilter = 'all';
    let statusFilter = 'all';
    
    onMount(async () => {
        try {
            endpoints = await endpointsApi.getEndpoints();
            loading = false;
        } catch (err) {
            error = err instanceof Error ? err.message : 'Failed to load endpoints';
            loading = false;
        }
    });
    
    $: filteredEndpoints = endpoints.filter(endpoint => {
        // Apply search filter
        const matchesSearch = endpoint.path.toLowerCase().includes(searchQuery.toLowerCase());
        
        // Apply method filter
        const matchesMethod = methodFilter === 'all' || endpoint.method === methodFilter;
        
        // Apply status filter
        let matchesStatus = true;
        if (statusFilter === '2xx') {
            matchesStatus = endpoint.status_code !== null && endpoint.status_code >= 200 && endpoint.status_code < 300;
        } else if (statusFilter === '3xx') {
            matchesStatus = endpoint.status_code !== null && endpoint.status_code >= 300 && endpoint.status_code < 400;
        } else if (statusFilter === '4xx') {
            matchesStatus = endpoint.status_code !== null && endpoint.status_code >= 400 && endpoint.status_code < 500;
        } else if (statusFilter === '5xx') {
            matchesStatus = endpoint.status_code !== null && endpoint.status_code >= 500;
        }
        
        return matchesSearch && matchesMethod && matchesStatus;
    });
    
    // Count methods for stats
    $: methodCounts = endpoints.reduce((counts, endpoint) => {
        counts[endpoint.method] = (counts[endpoint.method] || 0) + 1;
        return counts;
    }, {} as Record<string, number>);
</script>

<svelte:head>
    <title>Endpoints - Attack Surface Management</title>
</svelte:head>

<div class="endpoints-page">
    <header class="page-header">
        <h1>Endpoints</h1>
        <div class="search-container">
            <input 
                type="text" 
                placeholder="Search endpoints..." 
                bind:value={searchQuery} 
                class="search-input"
            />
        </div>
    </header>
    
    {#if loading}
        <p>Loading endpoints...</p>
    {:else if error}
        <p class="error">{error}</p>
    {:else}
        <div class="filter-bar">
            <div class="filter-group">
                <label for="method-filter">Method:</label>
                <select id="method-filter" bind:value={methodFilter} class="filter-select">
                    <option value="all">All Methods</option>
                    <option value="GET">GET</option>
                    <option value="POST">POST</option>
                    <option value="PUT">PUT</option>
                    <option value="DELETE">DELETE</option>
                </select>
            </div>
            
            <div class="filter-group">
                <label for="status-filter">Status:</label>
                <select id="status-filter" bind:value={statusFilter} class="filter-select">
                    <option value="all">All Status Codes</option>
                    <option value="2xx">2xx (Success)</option>
                    <option value="3xx">3xx (Redirection)</option>
                    <option value="4xx">4xx (Client Error)</option>
                    <option value="5xx">5xx (Server Error)</option>
                </select>
            </div>
        </div>
        
        <div class="stats">
            <div class="stat-item">
                <span class="stat-value">{endpoints.length}</span>
                <span class="stat-label">Total Endpoints</span>
            </div>
            {#each Object.entries(methodCounts) as [method, count]}
                <div class="stat-item">
                    <span class="stat-value method-{method.toLowerCase()}">{count}</span>
                    <span class="stat-label">{method}</span>
                </div>
            {/each}
        </div>
        
        <div class="list-container">
            {#if filteredEndpoints.length === 0}
                <p class="empty-message">
                    {searchQuery || methodFilter !== 'all' || statusFilter !== 'all' 
                        ? 'No endpoints match your filters.' 
                        : 'No endpoints have been discovered yet.'}
                </p>
            {:else}
                <EndpointList endpoints={filteredEndpoints} showSubdomainColumn={true} />
            {/if}
        </div>
    {/if}
</div>

<style>
    .endpoints-page {
        padding: 1rem 0;
    }
    
    .page-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 1.5rem;
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
    
    .filter-bar {
        display: flex;
        gap: 1rem;
        margin-bottom: 1.5rem;
        background-color: var(--card-bg); /* Use variable */
        padding: 1rem;
        border-radius: 4px;
        border: 1px solid var(--border); /* Add border */
    }
    
    .filter-group {
        display: flex;
        align-items: center;
        gap: 0.5rem;
        color: var(--text-light); /* Use variable */
    }
    
    .filter-select {
        padding: 0.375rem 0.75rem;
        border: 1px solid var(--border); /* Use variable */
        background-color: var(--background); /* Use variable */
        color: var(--text); /* Use variable */
        border-radius: 4px;
        font-size: 0.875rem;
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
    
    /* Semantic colors - keep these as they are meaningful */
    .stat-value.method-get { color: #2563eb; }
    .stat-value.method-post { color: #16a34a; }
    .stat-value.method-put { color: #d97706; }
    .stat-value.method-delete { color: #dc2626; }
    
    .stat-label {
        color: var(--text-light); /* Use variable */
        font-size: 0.875rem;
    }
    
    
    .empty-message {
        padding: 2rem;
        background-color: var(--card-bg); /* Use variable */
        border-radius: 4px;
        text-align: center;
        color: var(--text-light); /* Use variable */
        border: 1px solid var(--border); /* Add border */
    }
    
    .error {
        color: #ef4444; /* Keep error color */
    }
</style>
