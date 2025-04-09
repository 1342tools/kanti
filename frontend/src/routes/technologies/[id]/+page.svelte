<script lang="ts">
    import { onMount } from 'svelte';
    import { page } from '$app/stores';
    import { technologiesApi } from '$lib/api/api';
    import type { Technology, RootDomain, Subdomain, Endpoint } from '$lib/types';
    import SubdomainList from '$lib/components/SubdomainList.svelte';
    import EndpointList from '$lib/components/EndpointList.svelte';
    
    const technologyId = parseInt($page.params.id);
    
    let technology: Technology | null = null;
    let domains: RootDomain[] = [];
    let subdomains: Subdomain[] = [];
    let endpoints: Endpoint[] = [];
    let loading = true;
    let error = '';
    
    // Tab management
    let activeTab = 'subdomains';
    
    onMount(async () => {
        try {
            await loadData();
            loading = false;
        } catch (err) {
            error = err instanceof Error ? err.message : 'Failed to load technology data';
            loading = false;
        }
    });
    
    async function loadData() {
        // Load technology
        technology = await technologiesApi.getTechnology(technologyId);
        
        // Load domains, subdomains, and endpoints in parallel
        [domains, subdomains, endpoints] = await Promise.all([
            technologiesApi.getDomainsWithTechnology(technologyId),
            technologiesApi.getSubdomainsWithTechnology(technologyId),
            technologiesApi.getEndpointsWithTechnology(technologyId)
        ]);
    }
    
    function switchTab(tab: string) {
        activeTab = tab;
    }
</script>

<svelte:head>
    <title>{technology ? technology.name : 'Technology'} - Attack Surface Management</title>
</svelte:head>

<div class="technology-details">
    {#if loading}
        <p>Loading technology data...</p>
    {:else if error}
        <p class="error">{error}</p>
    {:else if technology}
        <header class="technology-header">
            <h1>{technology.name}</h1>
            {#if technology.category}
                <div class="technology-category">
                    <span class="category-label">Category:</span>
                    <span class="category-value">{technology.category}</span>
                </div>
            {/if}
            
            <div class="technology-stats">
                <div class="stat-item">
                    <span class="stat-value">{domains.length}</span>
                    <span class="stat-label">Domains</span>
                </div>
                <div class="stat-item">
                    <span class="stat-value">{subdomains.length}</span>
                    <span class="stat-label">Subdomains</span>
                </div>
                <div class="stat-item">
                    <span class="stat-value">{endpoints.length}</span>
                    <span class="stat-label">Endpoints</span>
                </div>
            </div>
        </header>
        
        <div class="tab-navigation">
            <button 
                class="tab-btn" 
                class:active={activeTab === 'subdomains'}
                on:click={() => switchTab('subdomains')}
            >
                Subdomains ({subdomains.length})
            </button>
            <button 
                class="tab-btn" 
                class:active={activeTab === 'endpoints'}
                on:click={() => switchTab('endpoints')}
            >
                Endpoints ({endpoints.length})
            </button>
            <button 
                class="tab-btn" 
                class:active={activeTab === 'domains'}
                on:click={() => switchTab('domains')}
            >
                Domains ({domains.length})
            </button>
        </div>
        
        <div class="tab-content">
            {#if activeTab === 'subdomains'}
                {#if subdomains.length === 0}
                    <p class="empty-message">No subdomains found with this technology.</p>
                {:else}
                    <SubdomainList subdomains={subdomains} showDomainColumn={true} />
                {/if}
            {:else if activeTab === 'endpoints'}
                {#if endpoints.length === 0}
                    <p class="empty-message">No endpoints found with this technology.</p>
                {:else}
                    <EndpointList endpoints={endpoints} showSubdomainColumn={true} />
                {/if}
            {:else}
                {#if domains.length === 0}
                    <p class="empty-message">No domains found with this technology.</p>
                {:else}
                    <table class="domains-table">
                        <thead>
                            <tr>
                                <th>Domain</th>
                                <th>Added</th>
                                <th>Last Scanned</th>
                                <th>Actions</th>
                            </tr>
                        </thead>
                        <tbody>
                            {#each domains as domain}
                                <tr>
                                    <td>{domain.domain}</td>
                                    <td>{new Date(domain.created_at).toLocaleDateString()}</td>
                                    <td>{domain.last_scanned_at ? new Date(domain.last_scanned_at).toLocaleDateString() : 'Never'}</td>
                                    <td>
                                        <a href={`/domains/${domain.id}`} class="btn btn-sm">View</a>
                                    </td>
                                </tr>
                            {/each}
                        </tbody>
                    </table>
                {/if}
            {/if}
        </div>
    {:else}
        <p>Technology not found</p>
    {/if}
</div>

<style>
    .technology-details {
        padding: 1rem 0;
    }
    .technology-details h1 {
        color: var(--text); /* Use variable */
    }
    
    .technology-header {
        margin-bottom: 2rem;
        padding-bottom: 1rem;
        border-bottom: 1px solid var(--border); /* Use variable */
    }
    
    .technology-category {
        margin: 0.5rem 0 1rem;
    }
    
    .category-label {
        color: var(--text-light); /* Use variable */
        font-weight: 500;
        margin-right: 0.5rem;
    }
    
    .category-value {
        display: inline-block;
        padding: 0.25rem 0.5rem;
        background-color: var(--card-bg); /* Use variable */
        border: 1px solid var(--border); /* Add border */
        border-radius: 4px;
        color: var(--text); /* Use variable */
        font-size: 0.875rem;
    }
    
    .technology-stats {
        display: flex;
        gap: 2rem;
        margin-top: 1.5rem;
    }
    
    .stat-item {
        display: flex;
        flex-direction: column;
        align-items: center;
    }
    
    .stat-value {
        font-size: 2rem;
        font-weight: 600;
        color: var(--primary); /* Use variable */
    }
    
    .stat-label {
        color: var(--text-light); /* Use variable */
        font-size: 0.875rem;
    }
    
    .tab-navigation {
        display: flex;
        border-bottom: 1px solid var(--border); /* Use variable */
        margin-bottom: 1.5rem;
    }
    
    .tab-btn {
        padding: 0.75rem 1rem;
        background: none;
        border: none;
        cursor: pointer;
        font-size: 0.875rem;
        color: var(--text-light); /* Use variable */
        border-bottom: 2px solid transparent; /* Add transparent border */
        margin-bottom: -1px; /* Overlap border */
    }
    
    .tab-btn.active {
        color: var(--primary); /* Use variable */
        font-weight: 500;
        border-bottom: 2px solid var(--primary); /* Use variable */
    }
    
    .domains-table {
        width: 100%;
        border-collapse: collapse;
        font-size: 0.875rem;
    }
    
    .domains-table th, .domains-table td {
        padding: 0.75rem;
        text-align: left;
        border-bottom: 1px solid var(--border); /* Use variable */
        color: var(--text); /* Use variable */
    }
    
    .domains-table th {
        font-weight: 600;
        color: var(--text-light); /* Use variable */
        background-color: var(--card-bg); /* Use variable */
    }
    
    /* Global .btn styles are in app.css */
    /* .btn { ... } */
    /* .btn-sm { ... } */
    
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
