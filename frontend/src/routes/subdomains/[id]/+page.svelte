<script lang="ts">
    import { onMount } from 'svelte';
    import { page } from '$app/stores';
    import { get } from 'svelte/store';
    import { subdomainsApi } from '$lib/api/api';
    import type { Subdomain, Endpoint } from '$lib/types';
    import EndpointList from '$lib/components/EndpointList.svelte';
    import TechnologyList from '$lib/components/TechnologyList.svelte';
    import { API_BASE_URL } from '$lib/api/api'; // Import base URL

    const subdomainId = parseInt(get(page).params.id); // Use get() for initial value
    
    let subdomain: Subdomain | null = null;
    let endpoints: Endpoint[] = [];
    let loading = true;
    let error = '';
    let selectedExtension: string | null = null; // State for selected tab
    
    onMount(async () => {
        try {
            await loadData();
            loading = false;
        } catch (err) {
            error = err instanceof Error ? err.message : 'Failed to load subdomain data';
            loading = false;
        }
    });
    
    async function loadData() {
        // Load subdomain and endpoints in parallel
        const [subdomainData, endpointsData] = await Promise.all([
            subdomainsApi.getSubdomain(subdomainId),
            subdomainsApi.getSubdomainEndpoints(subdomainId)
        ]);
        
        subdomain = subdomainData;
        endpoints = endpointsData;
    }

    // Helper function to extract file extension from a path
    function getFileExtension(path: string): string | null {
        const lastDotIndex = path.lastIndexOf('.');
        const lastSlashIndex = path.lastIndexOf('/');

        // Ensure the dot is after the last slash and not the first character
        if (lastDotIndex > lastSlashIndex && lastDotIndex > 0) {
            // Basic check for common web extensions, adjust as needed
            // Exclude paths ending in just '.' or '..'
            if (path.endsWith('/.') || path.endsWith('/..')) return null;

            const ext = path.substring(lastDotIndex + 1).toLowerCase();
            // Allow extensions up to 6 chars, alphanumeric
            if (ext.length > 0 && ext.length <= 6 && /^[a-z0-9]+$/.test(ext)) {
                 // Check if it looks like a version number segment (e.g., v1.2) - crude check
                 if (lastDotIndex > 0 && path[lastDotIndex - 1].match(/[a-z]/i) && ext.match(/^\d+$/)) {
                     // Likely part of a versioned path segment, not a file extension
                     return null;
                 }
                 return `.${ext}`;
            }
        }
        return null; // No valid extension found
    }

    // Reactive statement to group endpoints by extension
    $: groupedEndpoints = endpoints.reduce((acc, endpoint) => {
        // Use URL constructor for more robust path parsing
        let pathname = '/'; // Default for root or invalid paths
        try {
            // Assume http protocol if none is present for parsing
            const url = new URL(endpoint.path.startsWith('http') ? endpoint.path : `http://dummy${endpoint.path}`);
            pathname = url.pathname;
        } catch (e) {
            // If URL parsing fails, fall back to the original path
            pathname = endpoint.path;
        }

        const extension = getFileExtension(pathname) || 'No Extension';
        if (!acc[extension]) {
            acc[extension] = [];
        }
        acc[extension].push(endpoint);
        return acc;
    }, {} as Record<string, Endpoint[]>);

    // Sort extensions for consistent display order
    $: sortedExtensions = Object.keys(groupedEndpoints).sort((a, b) => {
        if (a === 'No Extension') return 1; // Put 'No Extension' last
        if (b === 'No Extension') return -1;
        // Sort by extension name, ignoring the leading dot
        return a.substring(1).localeCompare(b.substring(1));
    });

    // Set the initial selected extension once data is loaded and grouped
    $: {
        if (!loading && !error && sortedExtensions.length > 0 && selectedExtension === null) {
            selectedExtension = sortedExtensions[0]; // Default to the first extension
        } else if (!loading && !error && sortedExtensions.length === 0) {
             selectedExtension = null; // Reset if no endpoints
         }
     }
 
     // Helper function to get display label for extension tabs
     function getExtensionLabel(extension: string): string {
         const labelMap: Record<string, string> = {
             '.css': 'CSS',
             '.js': 'Javascript',
             '.php': 'PHP',
             '.html': 'HTML',
             'No Extension': 'Paths' 
         };
         return labelMap[extension] || extension; // Return mapped label or the extension itself
     }
 </script>
 
 <svelte:head>
    <title>{subdomain ? subdomain.hostname : 'Subdomain'} - Attack Surface Management</title>
</svelte:head>

<div class="subdomain-details">
    {#if loading}
        <p>Loading subdomain data...</p>
    {:else if error}
        <p class="error">{error}</p>
    {:else if subdomain}
        <header class="subdomain-header">
            <div class="header-content-wrapper">
                <h1>{subdomain.hostname}</h1>
                <div class="subdomain-meta">
                    <div class="meta-item">
                        <span class="meta-label">IP Address:</span>
                    <span class="meta-value">{subdomain.ip_address || 'Unknown'}</span>
                </div>
                <div class="meta-item">
                    <span class="meta-label">Status:</span>
                    <span class="status status-{subdomain.is_active ? 'active' : 'inactive'}">
                        {subdomain.is_active ? 'Active' : 'Inactive'}
                    </span>
                </div>
                <div class="meta-item">
                    <span class="meta-label">Discovered:</span>
                    <span class="meta-value">{new Date(subdomain.discovered_at).toLocaleString()}</span>
                </div>
            </div>
            
            <div class="subdomain-actions">
                <a href={`/domains/${subdomain.root_domain_id}`} class="btn btn-outline">
                    Back to Domain
                </a>
                <a href={`https://${subdomain.hostname}`} target="_blank" rel="noopener noreferrer" class="btn">
                        Visit Subdomain
                    </a>
                </div>
            </div>

             <!-- Screenshot Display (Remains in header, but outside wrapper) -->
            {#if subdomain.latest_screenshot_path}
               <div class="screenshot-container-header">
                   <!-- Construct the image URL -->
                   {#if subdomain.latest_screenshot_path}
                       {@const relativePath = subdomain.latest_screenshot_path.replace(/^(\.\/)?data\/screenshots\//, '')} 
                       {@const imageUrl = `${API_BASE_URL}/screenshots/${relativePath}`}
                       <a href={imageUrl} target="_blank" rel="noopener noreferrer" title="View full screenshot">
                           <img src={imageUrl} alt="Screenshot of {subdomain.hostname}" class="screenshot-image-header" />
                       </a>
                   {/if}
               </div>
            {/if}
        </header>
        
        <section class="technologies-section">
            <h2>Detected Technologies</h2>
            {#if subdomain.technologies && subdomain.technologies.length > 0}
                <TechnologyList technologies={subdomain.technologies} showConfidence={true} />
            {:else}
                <p class="no-data">No technologies detected for this subdomain.</p>
            {/if}
        </section>

        {#if endpoints.length === 0}
             <section class="endpoints-section">
                 <h2>Endpoints (0)</h2>
                 <p class="no-data">No endpoints discovered for this subdomain.</p>
             </section>
        {:else}
            <section class="endpoints-section">
                <h2>Endpoints ({endpoints.length})</h2>
                
                <!-- Tabs for selecting extension -->
                <div class="extension-tabs">
                    {#each sortedExtensions as extension}
                        <button 
                            class="tab-button" 
                            class:active={selectedExtension === extension}
                            on:click={() => selectedExtension = extension}
                        >
                            {getExtensionLabel(extension)} 
                            ({groupedEndpoints[extension].length})
                        </button>
                    {/each}
                </div>

                <!-- Display only the selected endpoint list -->
                {#if selectedExtension && groupedEndpoints[selectedExtension]}
                    <div class="list-container">
                         <EndpointList endpoints={groupedEndpoints[selectedExtension]} />
                    </div>
                {:else if selectedExtension}
                     <p class="no-data">No endpoints found for this selection.</p> 
                {/if}
            </section>
        {/if}
    {:else}
        <p>Subdomain not found</p>
    {/if}
</div>

<style>
    .subdomain-details {
        padding: 1rem 0;
    }
    .subdomain-details h1, .subdomain-details h2 {
        color: var(--text); /* Use variable */
    }
    
    .subdomain-header {
        display: flex; /* Use flexbox */
        justify-content: space-between; /* Space out content and screenshot */
        align-items: flex-start; /* Align items to the top */
        gap: 2rem; /* Add gap between content and screenshot */
        margin-bottom: 2rem;
        padding-bottom: 1rem;
        border-bottom: 1px solid var(--border); /* Use variable */
    }

    .header-content-wrapper {
        flex-grow: 1; /* Allow content to take available space */
    }
    
    .subdomain-meta {
        display: flex;
        flex-wrap: wrap;
        gap: 1.5rem;
        margin: 1rem 0;
    }
    
    .meta-item {
        display: flex;
        align-items: center;
        gap: 0.5rem;
    }
    
    .meta-label {
        color: var(--text-light); /* Use variable */
        font-weight: 500;
    }
    .meta-value {
        color: var(--text); /* Use variable */
    }
    
    /* Semantic colors - keep these */
    .status { display: inline-block; padding: 0.25rem 0.5rem; border-radius: 4px; font-size: 0.75rem; font-weight: 500; }
    .status-active { background-color: #dcfce7; color: #16a34a; }
    
    .status-inactive {
        background-color: var(--card-bg); /* Use variable */
        color: var(--text-light); /* Use variable */
    }
    
    .subdomain-actions {
        display: flex;
        gap: 1rem;
        margin-top: 1.5rem;
    }
    
    /* Global .btn styles are in app.css */
    /* .btn { ... } */
    /* .btn-outline { ... } */
    
    section {
        margin-bottom: 2rem;
    }
    
    h2 {
        margin-bottom: 1rem;
        font-size: 1.25rem;
    }

    .extension-tabs {
        display: flex;
        flex-wrap: wrap;
        gap: 0.5rem;
        margin-bottom: 1.5rem;
        padding-bottom: 0.75rem;
        border-bottom: 2px solid var(--border);
    }

    .tab-button {
        padding: 0.5rem 1rem;
        border: none;
        background-color: transparent;
        color: var(--text-light);
        cursor: pointer;
        font-size: 0.875rem;
        font-weight: 500;
        border-bottom: 2px solid transparent;
        margin-bottom: -2px; /* Align bottom border with container border */
        transition: color 0.2s ease, border-color 0.2s ease;
    }

    .tab-button:hover {
        color: var(--primary);
    }

    .tab-button.active {
        color: var(--primary);
        border-bottom-color: var(--primary);
        font-weight: 600;
    }
    
    .no-data {
        padding: 1rem;
        background-color: var(--card-bg); /* Use variable */
        border-radius: 4px;
        color: var(--text-light); /* Use variable */
        font-style: italic;
        border: 1px solid var(--border); /* Add border */
        margin-top: 1rem; /* Add margin if it follows tabs */
    }
    
    .error {
        color: #ef4444; /* Keep error color */
    }

    /* Styles for screenshot thumbnail in header */
    .screenshot-container-header {
       flex-shrink: 0; /* Prevent screenshot from shrinking */
    }
    .screenshot-image-header {
        max-width: 250px; /* Smaller max width */
        max-height: 150px; /* Smaller max height */
        height: auto;
        border: 1px solid var(--border);
        border-radius: 4px;
        box-shadow: 0 1px 3px rgba(0,0,0,0.1);
        cursor: pointer; 
        object-fit: contain; 
        background-color: var(--card-bg); 
        display: block; /* Ensure it behaves like a block */
    }
</style>
