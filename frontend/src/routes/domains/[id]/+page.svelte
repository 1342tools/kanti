<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation'; // Import goto for programmatic navigation
	import { domainsApi, subdomainsApi, scansApi, scanTemplatesApi } from '$lib/api/api'; // Import scanTemplatesApi
	import type { RootDomain, Subdomain, Scan, ScanTemplate } from '$lib/types'; // Import ScanTemplate type
	// SubdomainList removed
	import ScanStatusIndicator from '$lib/components/ScanStatusIndicator.svelte';
	import Modal from '$lib/components/Modal.svelte'; // Import Modal
	import AlertMessage from '$lib/components/AlertMessage.svelte'; // Import AlertMessage

	const domainId = parseInt($page.params.id);

	let domain: RootDomain | null = null;
    let subdomains: Subdomain[] = [];
    let scans: Scan[] = [];
	let loading = true;
	let error: string | null = null; // General page error
	let templateError: string | null = null; // Specific error for modal

	let activeScan: Scan | null = null;
	let scanInterval: number | undefined = undefined;

	// State for template selection modal (aligned with org page)
	let showTemplateModal = false;
	let availableTemplates: ScanTemplate[] = [];
	let selectedTemplateId: number | null = null; // Use null for no selection
	let isLoadingTemplates = false;

	onMount(() => {
        const init = async () => {
            try {
                await loadData();
                loading = false;
                
                // Check for active scans every 5 seconds
                scanInterval = setInterval(checkActiveScan, 5000) as unknown as number;
            } catch (err) {
                error = err instanceof Error ? err.message : 'Failed to load domain data';
                loading = false;
            }
        };
        
        init(); // Call the async function
        
        // Return the cleanup function directly
        return () => {
            if (scanInterval !== undefined) {
                clearInterval(scanInterval);
            }
		};
	});

	async function loadData() {
		loading = true; // Ensure loading state is set
		error = null;
		try {
			// Load domain, subdomains, and scans in parallel
			const [domainData, subdomainsData, scansData] = await Promise.all([
				domainsApi.getDomain(domainId),
				subdomainsApi.getSubdomains(domainId),
				scansApi.getScans(domainId)
			]);

			domain = domainData;
        subdomains = subdomainsData;
        scans = scansData;
        
        // Check if there's an active scan
        activeScan = scans.find(scan => scan.status === 'running') || null;
		} catch (err) { // Catch block for loadData
			error = err instanceof Error ? err.message : 'Failed to load initial data';
		} finally { // Finally block for loadData
			loading = false;
		}
	}

	async function checkActiveScan() {
		if (!activeScan) return; // Exit if no active scan

		try { // Add try block for checkActiveScan
			const updatedScan = await scansApi.getScan(activeScan.id);

			if (updatedScan.status !== 'running') {
				// Scan completed or failed, reload data and clear interval if needed
				await loadData(); // This will reset activeScan if it's no longer running
				// Consider clearing interval here if loadData confirms no active scan,
				// but loadData already resets activeScan, so interval check should handle it.
			} else {
				// Update the local activeScan state if still running
				activeScan = updatedScan;
			}
		} catch (err) { // Add catch block for checkActiveScan
			console.error('Failed to check active scan status:', err);
			// Optionally set an error message, but avoid overwriting main page errors frequently
			// error = err instanceof Error ? err.message : 'Failed to check scan status';
			// Consider stopping checks if it fails repeatedly
			if (scanInterval !== undefined) {
				// clearInterval(scanInterval);
				// scanInterval = undefined;
				// error = "Failed to monitor scan status. Please refresh.";
			}
		}
	}


	async function openScanModal() { // Renamed from openTemplateModal
		selectedTemplateId = null; // Reset selection
		templateError = null; // Reset modal error
		showTemplateModal = true;
		isLoadingTemplates = true;
		try {
			availableTemplates = await scanTemplatesApi.getScanTemplates();
		} catch (err) {
			templateError = `Failed to load scan templates: ${err instanceof Error ? err.message : 'Unknown error'}`;
			availableTemplates = []; // Ensure it's empty on error
		} finally {
			isLoadingTemplates = false;
		}
	}

	function closeModal() { // Renamed from closeTemplateModal
		showTemplateModal = false;
		selectedTemplateId = null;
		templateError = null;
	}

	// Renamed from startScanWithTemplate and adapted logic
	async function handleScanWithTemplate() {
		if (selectedTemplateId === null) {
			templateError = 'Please select a scan template.'; // Use templateError
			return;
		}

		const templateToUse = selectedTemplateId; // Capture before closing modal
		closeModal(); // Close modal immediately
		error = null; // Clear general page error before starting scan

		// Optionally show a global loading indicator here

		try {
			// Use the correct scansApi.startScan
			await scansApi.startScan({
				rootDomainId: domainId,
				scanTemplateId: templateToUse
			});
			// Refresh data after starting scan
			await loadData();
		} catch (err) {
			// Display error on the main page
			error = `Failed to start scan: ${err instanceof Error ? err.message : 'Unknown error'}`;
		} finally {
			// Hide global loading indicator if used
		}
	}
</script>

<svelte:head>
    <title>{domain ? domain.domain : 'Domain'} - Attack Surface Management</title>
</svelte:head>

<div class="domain-details">
    {#if loading}
        <p>Loading domain data...</p>
    {:else if error}
        <p class="error">{error}</p>
    {:else if domain}
        <header class="domain-header">
            <h1>{domain.domain}</h1>
            <div class="domain-meta">
                <p>Added: {new Date(domain.created_at).toLocaleDateString()}</p>
                <p>Last Scan: {domain.last_scanned_at ? new Date(domain.last_scanned_at).toLocaleString() : 'Never'}</p>
            </div>

			<!-- Stats Section -->
			<div class="stats grid grid-cols-2 gap-4 my-4 p-4 bg-gray-800 rounded border border-gray-700">
				<div class="stat text-center">
					<div class="stat-value text-2xl font-bold">{domain.total_subdomains ?? 0}</div>
					<div class="stat-title text-sm text-gray-400">Subdomains</div>
				</div>
				<div class="stat text-center">
					<div class="stat-value text-2xl font-bold">{domain.total_endpoints ?? 0}</div>
					<div class="stat-title text-sm text-gray-400">Endpoints</div>
				</div>
			</div>
            
			<div class="domain-actions">
				<!-- Update button to open the modal using renamed function -->
				<button class="btn btn-primary" on:click={openScanModal} disabled={!!activeScan}>
					{activeScan ? 'Scan in Progress' : 'Start New Scan...'}
				</button>
			</div>

			{#if activeScan}
                <ScanStatusIndicator scan={activeScan} />
            {/if}
        </header>
        
        <section class="subdomains-section">
            <h2>Subdomains ({subdomains.length})</h2>
            
            {#if subdomains.length === 0}
                <p>No subdomains discovered yet. Run a scan to discover subdomains.</p>
            {:else}
                <!-- Start: Enhanced Grid Subdomain List Implementation -->
                <div class="subdomain-grid grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4">
                    {#each subdomains.sort((a, b) => a.hostname.localeCompare(b.hostname)) as subdomain (subdomain.id)}
                        <!-- Container is a div with click handler -->
                        <div 
                            class="subdomain-item-container flex flex-col p-3 bg-gray-800 rounded border border-gray-700 h-full hover:border-blue-600 transition-colors duration-150 cursor-pointer"
                            on:click={() => goto(`/subdomains/${subdomain.id}`)}
                            role="link"
                            tabindex="0"
                            aria-label={`View details for ${subdomain.hostname}`}
                            on:keydown={(e) => { if (e.key === 'Enter' || e.key === ' ') goto(`/subdomains/${subdomain.id}`); }}
                        >
                            <!-- Content -->
                            <div class="flex flex-col flex-grow"> <!-- Removed pointer-events-none -->
                                <!-- Top Section: Hostname & IP -->
                                <div class="mb-2">
                                    <!-- Hostname is no longer a link itself -->
                                    <span class="font-semibold break-all text-white">{subdomain.hostname}</span>
                                    <p class="text-xs text-gray-400 mt-1">{subdomain.ip_address || 'No IP'}</p>
                                </div>

                                <!-- Middle Section: Tech Tags -->
                                <div class="tech-tags flex flex-wrap gap-1 mb-3">
                                {#if subdomain.technologies && subdomain.technologies.length > 0}
                                    {#each subdomain.technologies.slice(0, 3) as tech}
                                        <span class="tech-tag text-xs px-1.5 py-0.5 rounded bg-gray-700 text-gray-300">{tech.name}</span>
                                    {/each}
                                    {#if subdomain.technologies.length > 3}
                                        <span class="tech-tag text-xs px-1.5 py-0.5 rounded bg-gray-600 text-gray-400">+{subdomain.technologies.length - 3}</span>
                                    {/if}
                                {:else}
                                    <!-- Optionally show placeholder or nothing -->
                                    <!-- <span class="text-xs text-gray-500 italic">No tech detected</span> -->
                                {/if}
                                </div>

                                <!-- Spacer to push bottom content down -->
                                <div class="flex-grow"></div>

                                <!-- Bottom Section: Status & Visit Button -->
                                <div class="flex justify-between items-center pt-2 border-t border-gray-700/50">
                                    <span class="status text-xs font-medium px-2 py-0.5 rounded {subdomain.is_active ? 'bg-green-900/50 text-green-300' : 'bg-red-900/50 text-red-300'}">
                                        {subdomain.is_active ? 'Active' : 'Inactive'}
                                    </span>
                                    <!-- Visit button needs relative positioning and higher z-index to be clickable over the main link -->
                                    <a
                                        href={`https://${subdomain.hostname}`}
                                        target="_blank"
                                        rel="noopener noreferrer"
                                        class="btn btn-outline btn-xs"
                                        title="Visit subdomain in new tab"
                                        on:click|stopPropagation
                                    >
                                        Visit
                                    </a>
                                </div>
                            </div>
                        </div> <!-- Close the main div container -->
                    {/each}
                </div>
                <!-- End: Enhanced Grid Subdomain List Implementation -->
            {/if}
        </section>

        <section class="scans-section">
            <h2>Scan History</h2>
            
            {#if scans.length === 0}
                <p>No scans have been run yet.</p>
            {:else}
                <table class="scans-table">
                    <thead>
                        <tr>
                            <th>ID</th>
                            <th>Type</th>
                            <th>Started</th>
                            <th>Completed</th>
                            <th>Status</th>
                        </tr>
                    </thead>
                    <tbody>
                        {#each scans as scan}
                            <tr>
                                <td><a href="/scans/{scan.id}" class="scan-link">{scan.id}</a></td>
                                <td>{scan.scan_type}</td>
                                <td>{new Date(scan.started_at).toLocaleString()}</td>
                                <td>{scan.completed_at ? new Date(scan.completed_at).toLocaleString() : '-'}</td>
                                <td>
                                    <span class="status status-{scan.status}">
                                        {scan.status}
                                    </span>
                                </td>
                            </tr>
                        {/each}
                    </tbody>
                </table>
            {/if}
        </section>
    {:else}
		<p>Domain not found</p>
	{/if}
</div>

<!-- Scan Template Selection Modal (aligned with organization page) -->
<Modal bind:show={showTemplateModal} on:close={closeModal}>
	<!-- Adjusted title for domain context -->
	<h2 class="text-xl font-semibold mb-4">Select Scan Template for {domain?.domain || 'this domain'}</h2>

	{#if isLoadingTemplates}
		<p>Loading templates...</p>
	{:else if templateError}
		<!-- Use templateError here -->
		<AlertMessage type="error" message={templateError} />
	{:else if availableTemplates.length === 0}
		<!-- Adjusted message slightly -->
		<p>No scan templates found. You can <a href="/scan-templates" class="link">create them</a> in the Scan Templates section.</p>
		<div class="mt-4 text-right">
			<button class="btn btn-secondary" on:click={closeModal}>Cancel</button>
		</div>
	{:else}
		<!-- Radio button structure from org page -->
		<div class="template-list space-y-2 mb-4 max-h-60 overflow-y-auto">
			{#each availableTemplates as template (template.id)}
				<label class="flex items-center p-2 rounded hover:bg-gray-700 cursor-pointer border border-transparent has-[:checked]:border-blue-500 has-[:checked]:bg-gray-700">
					<input type="radio" name="scan-template" value={template.id} bind:group={selectedTemplateId} class="mr-2 radio radio-primary radio-sm" />
					<span>{template.name}</span>
				</label>
			{/each}
		</div>
		<!-- Actions structure from org page -->
		<div class="modal-actions text-right space-x-2">
			<button class="btn btn-secondary" on:click={closeModal}>Cancel</button>
			<button class="btn btn-primary" on:click={handleScanWithTemplate} disabled={selectedTemplateId === null}>
				Start Scan
			</button>
		</div>
	{/if}
</Modal>


<style>
	/* Add specific styles if needed, rely on Tailwind/global styles */
	/* Styles from org page modal might be needed if not globally defined */
	.radio-primary { /* Example if needed */
		/* Define styles or ensure DaisyUI/Tailwind provides them */
	}
	.radio-sm { /* Example if needed */
		/* Define styles or ensure DaisyUI/Tailwind provides them */
	}

	.domain-details {
		padding: 1rem 0;
    }
    
    .domain-header {
        margin-bottom: 2rem;
        padding-bottom: 1rem;
        border-bottom: 1px solid #eee;
    }
    
    .domain-meta {
        display: flex;
        gap: 2rem;
        margin: 1rem 0;
        color: #666;
    }
    
    .domain-actions {
        margin: 1.5rem 0;
    }
    

    
    .btn:disabled {
        background-color: #a0aec0;
        cursor: not-allowed;
    }
    
    section {
        margin-bottom: 2rem;
    }
    
    .scans-table {
        width: 100%;
        border-collapse: collapse;
        margin-top: 1rem;
    }
    
    .scans-table th, .scans-table td {
        padding: 0.75rem;
        text-align: left;
        border-bottom: 1px solid #eee;
    }
    
    .status {
        display: inline-block;
        padding: 0.25rem 0.5rem;
        border-radius: 4px;
        font-size: 0.875rem;
    }
    
    .status-running {
        background-color: #fef3c7;
        color: #d97706;
    }
    
    .status-completed {
        background-color: #dcfce7;
        color: #16a34a;
    }
    
    .status-failed {
        background-color: #fee2e2;
        color: #dc2626;
    }
    
    .error {
        color: #ef4444;
    }


    .subdomain-item-container { 
    background-color: var(--card-bg); 
    border-radius: 4px;
    border: 1px solid var(--border); 
}
</style>
