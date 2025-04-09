<script lang="ts">
    import type { RootDomain } from '$lib/types'; // Use RootDomain type
    
    export let domains: RootDomain[] = []; // Changed prop name and type
    
    // Sort domains by name
    $: sortedDomains = [...domains].sort((a, b) => a.domain.localeCompare(b.domain));
</script>

<div class="domain-list"> <!-- Renamed class -->
    {#if domains.length === 0}
        <p class="empty-message">No domains found.</p>
    {:else}
        <table>
            <thead>
                <tr>
                    <th>Domain</th>
                    <th>Subdomains</th>
                    <th>Endpoints</th>
                    <th>Created</th>
                    <th>Actions</th>
                </tr>
            </thead>
            <tbody>
                {#each sortedDomains as domain} <!-- Changed loop variable -->
                    <tr>
                        <td class="domain-name"> <!-- Renamed class -->
                            <a href={`/domains/${domain.id}`}>{domain.domain}</a> <!-- Updated link and property -->
                        </td>
                        <td>{domain.total_subdomains}</td> <!-- Display total subdomains -->
                        <td>{domain.total_endpoints}</td> <!-- Display total endpoints -->
                        <td>{new Date(domain.created_at).toLocaleDateString()}</td> <!-- Display creation date -->
                        <td>
                            <div class="actions">
                                <a href={`/domains/${domain.id}`} class="btn btn-sm">Details</a>
                                <!-- Add other relevant actions if needed -->
                            </div>
                        </td>
                    </tr>
                {/each}
            </tbody>
        </table>
    {/if}
</div>

<style>
    .domain-list { /* Renamed class - JS comments are fine in <style> */
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
        border-bottom: 1px solid var(--border); 
    }
    
    th {
        font-weight: 600;
        color: var(--text-light); 
        background-color: var(--card-bg); 
    }
    
    .domain-name a { /* Renamed class - JS comments are fine in <style> */
        color: var(--text);
        text-decoration: none;
        font-weight: 500;
    }
    
    .domain-name a:hover { /* Renamed class - JS comments are fine in <style> */
        text-decoration: underline;
        color: var(--primary); 
    }
    
    .actions {
        display: flex;
        gap: 0.5rem;
    }
    
    /* Global .btn styles are in app.css */
    
    .empty-message {
        padding: 1rem;
        color: var(--text-light); 
        font-style: italic;
        text-align: center;
    }
</style>
