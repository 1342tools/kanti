<script lang="ts">
    import { onMount } from 'svelte';
    import { page } from '$app/stores';
    import { endpointsApi, API_BASE_URL } from '$lib/api/api'; // Import API_BASE_URL
    import type { Endpoint, Parameter, RequestResponse } from '$lib/types';
    import TechnologyList from '$lib/components/TechnologyList.svelte';
    
    const endpointId = parseInt($page.params.id);
    
    let endpoint: Endpoint | null = null;
    let parameters: Parameter[] = [];
    let requestResponses: RequestResponse[] = [];
    let loading = true;
    let error = '';
    
    // For request/response display
    let selectedRequestResponse: RequestResponse | null = null;
    let activeTab = 'request';
    
    onMount(async () => {
        try {
            await loadData();
            loading = false;
            
            if (requestResponses.length > 0) {
                selectedRequestResponse = requestResponses[0];
            }
        } catch (err) {
            error = err instanceof Error ? err.message : 'Failed to load endpoint data';
            loading = false;
        }
    });
    
    async function loadData() {
        // Load endpoint details
        const endpointData = await endpointsApi.getEndpoint(endpointId);
        endpoint = endpointData;
        
        // Load parameters and request/responses in parallel
        const [paramsData, reqRespData] = await Promise.all([
            endpointsApi.getEndpointParameters(endpointId),
            endpointsApi.getEndpointRequestResponses(endpointId)
        ]);
        
        parameters = paramsData;
        requestResponses = reqRespData;
    }
    
    function selectRequestResponse(reqResp: RequestResponse) {
        selectedRequestResponse = reqResp;
    }
    
    function switchTab(tab: 'request' | 'response') {
        activeTab = tab;
    }
    
    // Helper function to format JSON for display
    function formatJson(jsonString: string | null): string {
        if (!jsonString) return '';
        
        try {
            const parsed = JSON.parse(jsonString);
            return JSON.stringify(parsed, null, 2);
        } catch (e) {
            return jsonString;
        }
    }
    
    // Helper function to get status color
    function getStatusColor(statusCode: number | null): string {
        if (!statusCode) return 'gray';
        if (statusCode < 300) return 'green';
        if (statusCode < 400) return 'blue';
        if (statusCode < 500) return 'orange';
        return 'red';
    }
</script>

<svelte:head>
    <title>{endpoint ? `${endpoint.method} ${endpoint.path}` : 'Endpoint'} - Attack Surface Management</title>
</svelte:head>

<div class="endpoint-details">
    {#if loading}
        <p>Loading endpoint data...</p>
    {:else if error}
        <p class="error">{error}</p>
    {:else if endpoint}
        <header class="endpoint-header">
            <!-- Wrapper for main content -->
            <div class="header-content-wrapper"> 
                <div class="endpoint-title">
                    <span class="method method-{endpoint.method.toLowerCase()}">{endpoint.method}</span>
                    <h1>{endpoint.path}</h1>
                </div>
                
                <div class="endpoint-meta">
                    <div class="meta-item">
                        <span class="meta-label">Status:</span>
                    {#if endpoint.status_code}
                        <span class="status status-{getStatusColor(endpoint.status_code)}">
                            {endpoint.status_code}
                        </span>
                    {:else}
                        <span class="status status-unknown">Unknown</span>
                    {/if}
                </div>
                <div class="meta-item">
                    <span class="meta-label">Content Type:</span>
                    <span class="meta-value">{endpoint.content_type || 'Unknown'}</span>
                </div>
                <div class="meta-item">
                    <span class="meta-label">Discovered:</span>
                    <span class="meta-value">{new Date(endpoint.discovered_at).toLocaleString()}</span>
                </div>
            </div>
            
            <div class="endpoint-actions">
                    <a href={`/subdomains/${endpoint.subdomain_id}`} class="btn btn-outline">
                        Back to Subdomain
                    </a>
                </div>
            </div> 
            <!-- End Wrapper -->

            <!-- Screenshot Display (Remains in header, but outside wrapper) -->
            {#if endpoint.latest_screenshot_path}
               <div class="screenshot-container-header"> 
                   <!-- Construct the image URL -->
                   {#if endpoint.latest_screenshot_path}
                       {@const relativePath = endpoint.latest_screenshot_path.replace(/^(\.\/)?data\/screenshots\//, '')} 
                       {@const imageUrl = `${API_BASE_URL}/screenshots/${relativePath}`}
                       <a href={imageUrl} target="_blank" rel="noopener noreferrer" title="View full screenshot">
                           <img src={imageUrl} alt="Screenshot of {endpoint.path}" class="screenshot-image-header" />
                       </a>
                   {/if}
               </div>
            {/if}
        </header>
        
        <div class="endpoint-content">
            <div class="endpoint-sidebar">
                <section class="parameters-section">
                    <h2>Parameters ({parameters.length})</h2>
                    
                    {#if parameters.length === 0}
                        <p class="no-data">No parameters discovered for this endpoint.</p>
                    {:else}
                        <table class="parameters-table">
                            <thead>
                                <tr>
                                    <th>Name</th>
                                    <th>Type</th>
                                </tr>
                            </thead>
                            <tbody>
                                {#each parameters as param}
                                    <tr>
                                        <td>{param.name}</td>
                                        <td>
                                            <span class="param-type param-type-{param.param_type}">
                                                {param.param_type}
                                            </span>
                                        </td>
                                    </tr>
                                {/each}
                            </tbody>
                        </table>
                    {/if}
                </section>
                
                <section class="technologies-section">
                    <h2>Technologies</h2>
                    
                    {#if endpoint.technologies && endpoint.technologies.length > 0}
                        <TechnologyList technologies={endpoint.technologies} showConfidence={true} />
                    {:else}
                        <p class="no-data">No technologies detected for this endpoint.</p>
                    {/if}
                </section>
            </div>
            
            <div class="endpoint-main">
                <section class="request-response-section">
                    <h2>Request/Response Samples</h2>
                    
                    {#if requestResponses.length === 0}
                        <p class="no-data">No request/response samples available for this endpoint.</p>
                    {:else}
                        <div class="req-resp-container">
                            <div class="req-resp-list">
                                {#each requestResponses as reqResp}
                                    <div 
                                        class="req-resp-item" 
                                        class:active={selectedRequestResponse && selectedRequestResponse.id === reqResp.id}
                                        on:click={() => selectRequestResponse(reqResp)}
                                    >
                                        <span class="req-resp-date">
                                            {new Date(reqResp.captured_at).toLocaleString()}
                                        </span>
                                    </div>
                                {/each}
                            </div>
                            
                            {#if selectedRequestResponse}
                                <div class="req-resp-detail">
                                    <div class="req-resp-tabs">
                                        <button 
                                            class="tab-btn" 
                                            class:active={activeTab === 'request'}
                                            on:click={() => switchTab('request')}
                                        >
                                            Request
                                        </button>
                                        <button 
                                            class="tab-btn" 
                                            class:active={activeTab === 'response'}
                                            on:click={() => switchTab('response')}
                                        >
                                            Response
                                        </button>
                                    </div>
                                    
                                    <div class="req-resp-content">
                                        {#if activeTab === 'request'}
                                            <div class="content-section">
                                                <h3>Headers</h3>
                                                <pre class="code-block">{formatJson(selectedRequestResponse.request_headers)}</pre>
                                            </div>
                                            
                                            {#if selectedRequestResponse.request_body}
                                                <div class="content-section">
                                                    <h3>Body</h3>
                                                    <pre class="code-block">{formatJson(selectedRequestResponse.request_body)}</pre>
                                                </div>
                                            {/if}
                                        {:else}
                                            <div class="content-section">
                                                <h3>Headers</h3>
                                                <pre class="code-block">{formatJson(selectedRequestResponse.response_headers)}</pre>
                                            </div>
                                            
                                            {#if selectedRequestResponse.response_body}
                                                <div class="content-section">
                                                    <h3>Body</h3>
                                                    <pre class="code-block">{formatJson(selectedRequestResponse.response_body)}</pre>
                                                </div>
                                            {/if}
                                        {/if}
                                    </div>
                                </div>
                            {/if}
                        </div>
                    {/if}
                </section>
            </div>
        </div>
    {:else}
        <p>Endpoint not found</p>
    {/if}
</div>

<style>
    .endpoint-details {
        padding: 1rem 0;
    }
    
    .endpoint-header {
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
    
    .endpoint-title {
        display: flex;
        align-items: center;
        gap: 0.75rem;
    }
    .endpoint-title h1 {
        color: var(--text); /* Use variable */
    }
    
    /* Semantic colors - keep these */
    .method { display: inline-block; padding: 0.25rem 0.5rem; border-radius: 4px; font-size: 0.75rem; font-weight: 600; text-transform: uppercase; }
    .method-get { background-color: #dbeafe; color: #2563eb; }
    .method-post { background-color: #dcfce7; color: #16a34a; }
    .method-put { background-color: #fef3c7; color: #d97706; }
    .method-delete { background-color: #fee2e2; color: #dc2626; }
    
    .endpoint-meta {
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
    .status-green { background-color: #dcfce7; color: #16a34a; }
    .status-blue { background-color: #dbeafe; color: #2563eb; }
    .status-orange { background-color: #fef3c7; color: #d97706; }
    .status-red { background-color: #fee2e2; color: #dc2626; }
    
    .status-gray, .status-unknown {
        background-color: var(--card-bg); /* Use variable */
        color: var(--text-light); /* Use variable */
    }
    
    .endpoint-actions {
        margin-top: 1.5rem;
    }
    
    /* Global .btn styles are in app.css */
    /* .btn { ... } */
    /* .btn-outline { ... } */
    
    .endpoint-content {
        display: flex;
        gap: 2rem;
    }
    
    .endpoint-sidebar {
        width: 300px;
        flex-shrink: 0;
    }
    
    .endpoint-main {
        flex: 1;
    }
    
    section {
        margin-bottom: 2rem;
    }
    
    h2 {
        margin-bottom: 1rem;
        font-size: 1.25rem;
        color: var(--text); /* Use variable */
    }
    
    .parameters-table {
        width: 100%;
        border-collapse: collapse;
        font-size: 0.875rem;
    }
    
    .parameters-table th, .parameters-table td {
        padding: 0.5rem;
        text-align: left;
        border-bottom: 1px solid var(--border); /* Use variable */
        color: var(--text); /* Use variable */
    }
    .parameters-table th {
        color: var(--text-light); /* Use variable */
    }
    
    /* Semantic colors - keep these */
    .param-type { display: inline-block; padding: 0.125rem 0.375rem; border-radius: 4px; font-size: 0.75rem; }
    .param-type-query { background-color: #dbeafe; color: #2563eb; }
    .param-type-body { background-color: #dcfce7; color: #16a34a; }
    .param-type-cookie { background-color: #fef3c7; color: #d97706; }
    .param-type-header { background-color: #f3e8ff; color: #7e22ce; }
    
    .req-resp-container {
        display: flex;
        border: 1px solid var(--border); /* Use variable */
        border-radius: 8px;
        overflow: hidden;
        background-color: var(--background); /* Use variable */
    }
    
    .req-resp-list {
        width: 200px;
        border-right: 1px solid var(--border); /* Use variable */
        background-color: var(--card-bg); /* Use variable */
        overflow-y: auto;
        max-height: 500px; /* Add max height for scroll */
    }
    
    .req-resp-item {
        padding: 0.75rem;
        border-bottom: 1px solid var(--border); /* Use variable */
        cursor: pointer;
        transition: background-color 0.2s;
    }
    
    .req-resp-item:hover {
        background-color: color-mix(in srgb, var(--card-bg) 80%, var(--background)); /* Use mix */
    }
    
    .req-resp-item.active {
        background-color: color-mix(in srgb, var(--primary) 10%, var(--background)); /* Use mix */
        border-left: 3px solid var(--primary); /* Use variable */
        padding-left: calc(0.75rem - 3px); /* Adjust padding */
    }
    .req-resp-item.active .req-resp-date {
        color: var(--primary); /* Use variable */
    }
    
    .req-resp-date {
        font-size: 0.75rem;
        color: var(--text-light); /* Use variable */
    }
    
    .req-resp-detail {
        flex: 1;
    }
    
    .req-resp-tabs {
        display: flex;
        border-bottom: 1px solid var(--border); /* Use variable */
        background-color: var(--card-bg); /* Use variable */
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
    
    .req-resp-content {
        padding: 1rem;
        max-height: 440px; /* Adjust based on list height */
        overflow-y: auto;
    }
    
    .content-section {
        margin-bottom: 1rem;
    }
    
    .content-section h3 {
        font-size: 0.875rem;
        color: var(--text-light); /* Use variable */
        margin-bottom: 0.5rem;
    }
    
    .code-block {
        background-color: var(--card-bg); /* Use variable */
        color: var(--text); /* Use variable */
        padding: 0.75rem;
        border-radius: 4px;
        font-family: monospace;
        font-size: 0.875rem;
        overflow-x: auto;
        white-space: pre-wrap;
        border: 1px solid var(--border); /* Add border */
    }
    
    .no-data {
        padding: 1rem;
        background-color: var(--card-bg); /* Use variable */
        border-radius: 4px;
        color: var(--text-light); /* Use variable */
        font-style: italic;
        border: 1px solid var(--border); /* Add border */
    }
    
    .error {
        color: #ef4444; /* Keep error color */
    }
    
    @media (max-width: 768px) {
        .endpoint-content {
            flex-direction: column;
        }
        
        .endpoint-sidebar {
            width: 100%;
        }
    }

    /* Styles for screenshot thumbnail in header */
    .screenshot-container-header {
       flex-shrink: 0; /* Prevent screenshot from shrinking */
       /* margin-top: 1rem; Removed as flex alignment handles spacing */
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
