<script lang="ts">
    import { createEventDispatcher } from 'svelte';
    import type { Endpoint } from '$lib/types';
    
    export let endpoints: Endpoint[] = [];
    export let showSubdomainColumn = false;
    
    const dispatch = createEventDispatcher<{
        endpointSelected: Endpoint;
    }>();
    
    function handleEndpointClick(endpoint: Endpoint) {
        dispatch('endpointSelected', endpoint);
    }
    
    // Helper function to get status color
    function getStatusColor(statusCode: number | null): string {
        if (!statusCode) return 'gray';
        if (statusCode < 300) return 'green';
        if (statusCode < 400) return 'blue';
        if (statusCode < 500) return 'orange';
        return 'red';
    }
    
    // Sort endpoints by path
    $: sortedEndpoints = [...endpoints].sort((a, b) => {
        // First sort by method priority (GET, POST, PUT, DELETE, others)
        const methodPriority: { [key: string]: number } = {
            'GET': 1,
            'POST': 2,
            'PUT': 3,
            'DELETE': 4
        };
        
        const aPriority = methodPriority[a.method] || 5;
        const bPriority = methodPriority[b.method] || 5;
        
        if (aPriority !== bPriority) {
            return aPriority - bPriority;
        }
        
        // Then sort by path
        return a.path.localeCompare(b.path);
    });
</script>

<div class="endpoint-list">
    {#if endpoints.length === 0}
        <p class="empty-message">No endpoints found.</p>
    {:else}
        <table>
            <thead>
                <tr>
                    <th>Method</th>
                    <th>Path</th>
                    <th>Status</th>
                    <th>Content Type</th>
                    {#if showSubdomainColumn}
                        <th>Subdomain</th>
                    {/if}
                    <th>Discovered</th>
                    <th>Actions</th>
                </tr>
            </thead>
            <tbody>
                {#each sortedEndpoints as endpoint}
                    <tr>
                        <td>
                            <span class="method method-{endpoint.method.toLowerCase()}">
                                {endpoint.method}
                            </span>
                        </td>
                        <td class="path">
                            <a href={`/endpoints/${endpoint.id}`}>{endpoint.path}</a>
                        </td>
                        <td>
                            {#if endpoint.status_code}
                                <span class="status status-{getStatusColor(endpoint.status_code)}">
                                    {endpoint.status_code}
                                </span>
                            {:else}
                                <span class="status status-unknown">Unknown</span>
                            {/if}
                        </td>
                        <td>
                            <span class="content-type">
                                {endpoint.content_type || 'Unknown'}
                            </span>
                        </td>
                        {#if showSubdomainColumn}
                            <td>
                                <!-- This would need to be populated with the actual subdomain data -->
                                <a href={`/subdomains/${endpoint.subdomain_id}`}>View Subdomain</a>
                            </td>
                        {/if}
                        <td>{new Date(endpoint.discovered_at).toLocaleDateString()}</td>
                        <td>
                            <div class="actions">
                                <a href={`/endpoints/${endpoint.id}`} class="btn btn-sm">Details</a>
                            </div>
                        </td>
                    </tr>
                {/each}
            </tbody>
        </table>
    {/if}
</div>

<style>
    .endpoint-list {
        width: 100%;
        overflow-x: auto;
    }
    
    table {
        width: 100%;
        border-collapse: collapse;
        font-size: 0.875rem;
    }
    
    th, td {
        padding: 0.75rem;
        text-align: left;
        border-bottom: 1px solid var(--border); /* Use variable */
    }
    
    th {
        font-weight: 600;
        color: var(--text-light); /* Use variable */
        background-color: var(--card-bg); /* Use variable */
    }
    
    .method {
        display: inline-block;
        padding: 0.25rem 0.5rem;
        border-radius: 4px;
        font-size: 0.75rem;
        font-weight: 600;
        text-transform: uppercase;
    }
    
    /* Semantic colors - might not need direct theme mapping */
    .method-get { background-color: #dbeafe; color: #2563eb; }
    .method-post { background-color: #dcfce7; color: #16a34a; }
    .method-put { background-color: #fef3c7; color: #d97706; }
    .method-delete { background-color: #fee2e2; color: #dc2626; }
    
    .path a {
        color: var(--primary); /* Use variable */
        text-decoration: none;
        font-weight: 500;
    }
    
    .path a:hover {
        text-decoration: underline;
    }
    
    .status {
        display: inline-block;
        padding: 0.25rem 0.5rem;
        border-radius: 4px;
        font-size: 0.75rem;
        font-weight: 500;
    }
    
    /* Semantic colors - might not need direct theme mapping */
    .status-green { background-color: #dcfce7; color: #16a34a; }
    .status-blue { background-color: #dbeafe; color: #2563eb; }
    .status-orange { background-color: #fef3c7; color: #d97706; }
    .status-red { background-color: #fee2e2; color: #dc2626; }
    
    .status-gray, .status-unknown {
        background-color: var(--card-bg); /* Use variable */
        color: var(--text-light); /* Use variable */
    }
    
    .content-type {
        font-size: 0.75rem;
        color: var(--text-light); /* Use variable */
    }
    
    .actions {
        display: flex;
        gap: 0.5rem;
    }
    
    /* .btn styles are handled globally in app.css */
    /* .btn { ... } */
    /* .btn-sm { ... } */
    
    .empty-message {
        padding: 1rem;
        color: var(--text-light); /* Use variable */
        font-style: italic;
        text-align: center;
    }
</style>
