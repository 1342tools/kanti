<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import type { Subdomain } from '$lib/types';
	import { scansApi, HttpError } from '$lib/api/api'; // Import scansApi and HttpError
	import AlertMessage from './AlertMessage.svelte'; // For showing messages

	export let subdomains: Subdomain[] = [];
	export let showDomainColumn = false;

	let scanningSubdomainId: number | null = null; // Track which subdomain is being scanned
	let scanMessage: string | null = null;
	let scanError: string | null = null;
    
    const dispatch = createEventDispatcher<{
        subdomainSelected: Subdomain;
    }>();
    
    function handleSubdomainClick(subdomain: Subdomain) {
        dispatch('subdomainSelected', subdomain);
    }
    
    // Sort subdomains by hostname
    $: sortedSubdomains = [...subdomains].sort((a, b) => a.hostname.localeCompare(b.hostname));

    async function scanSubdomain(subdomain: Subdomain) {
		scanningSubdomainId = subdomain.id;
		scanMessage = null;
		scanError = null;
		try {
			// TODO: Add ability to select a scan template
			const result = await scansApi.startScan({
				rootDomainId: subdomain.root_domain_id,
				subdomainId: subdomain.id
			});
			scanMessage = `${result.message} (Scan ID: ${result.scan_id})`;
		} catch (error) {
			console.error('Scan start error:', error);
			if (error instanceof HttpError) {
				scanError = `Failed to start scan: ${error.message} (${error.status})`;
			} else if (error instanceof Error) {
				scanError = `Failed to start scan: ${error.message}`;
			} else {
				scanError = 'An unknown error occurred while starting the scan.';
			}
		} finally {
			scanningSubdomainId = null;
			// Optionally clear messages after a delay
			setTimeout(() => {
				scanMessage = null;
				scanError = null;
			}, 5000);
		}
	}
</script>

<div class="subdomain-list">
	{#if scanMessage}
		<AlertMessage type="success" message={scanMessage} on:dismiss={() => (scanMessage = null)} />
	{/if}
	{#if scanError}
		<AlertMessage type="error" message={scanError} on:dismiss={() => (scanError = null)} />
	{/if}

    {#if subdomains.length === 0}
        <p class="empty-message">No subdomains found.</p>
    {:else}
        <table>
            <thead>
                <tr>
                    <th>Hostname</th>
                    <th>IP Address</th>
                    <th>Status</th>
                    <th>Technologies</th>
                    <th>Discovered</th>
                    <th>Actions</th>
                </tr>
            </thead>
            <tbody>
                {#each sortedSubdomains as subdomain}
                    <tr class:inactive={!subdomain.is_active}>
                        <td class="hostname">
                            <a href={`/subdomains/${subdomain.id}`}>{subdomain.hostname}</a>
                        </td>
                        <td>{subdomain.ip_address || '-'}</td>
                        <td>
                            <span class="status status-{subdomain.is_active ? 'active' : 'inactive'}">
                                {subdomain.is_active ? 'Active' : 'Inactive'}
                            </span>
                        </td>
                        <td>
                            <div class="tech-tags">
                                {#if subdomain.technologies && subdomain.technologies.length > 0}
                                    {#each subdomain.technologies.slice(0, 3) as tech}
                                        <span class="tech-tag">{tech.name}</span>
                                    {/each}
                                    {#if subdomain.technologies.length > 3}
                                        <span class="tech-tag more">+{subdomain.technologies.length - 3}</span>
                                    {/if}
                                {:else}
                                    <span class="no-tech">None detected</span>
                                {/if}
                            </div>
                        </td>
                        <td>{new Date(subdomain.discovered_at).toLocaleDateString()}</td>
                        <td>
                            <div class="actions">
                                <a href={`/subdomains/${subdomain.id}`} class="btn btn-sm">Details</a>
                                <a href={`https://${subdomain.hostname}`} target="_blank" rel="noopener noreferrer" class="btn btn-sm btn-outline">
                                    Visit
                                </a>
								<!-- Scan Button -->
								<button
									class="btn btn-sm btn-secondary"
									on:click={() => scanSubdomain(subdomain)}
									disabled={scanningSubdomainId === subdomain.id}
								>
									{#if scanningSubdomainId === subdomain.id}
										Scanning...
									{:else}
										Scan
									{/if}
								</button>
                            </div>
                        </td>
                    </tr>
                {/each}
            </tbody>
        </table>
    {/if}
</div>

<style>
    .subdomain-list {
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
    
    .hostname a {
        color: var(--text);
        text-decoration: none;
        font-weight: 500;
    }
    
    .hostname a:hover {
        text-decoration: underline;
        color: var(--primary); /* Use variable */
    }
    
    .inactive {
        opacity: 0.7;
    }
    
    .status {
        display: inline-block;
        padding: 0.25rem 0.5rem;
        border-radius: 4px;
        font-size: 0.75rem;
        font-weight: 500;
    }
    
    /* Define semantic colors - these might be okay as is, or could be themed */
    .status-active {
        background-color: #dcfce7; /* Light green */
        color: #16a34a; /* Dark green */
    }
    
    .status-inactive {
        background-color: var(--card-bg); /* Use variable */
        color: var(--text-light); /* Use variable */
    }
    
    .tech-tags {
        display: flex;
        flex-wrap: wrap;
        gap: 0.25rem;
    }
    
    .tech-tag {
        display: inline-block;
        padding: 0.125rem 0.375rem;
        /* Use a mix of primary and background */
        background-color: color-mix(in srgb, var(--primary) 15%, var(--background)); 
        color: var(--primary); /* Use variable */
        border-radius: 4px;
        font-size: 0.75rem;
    }
    
    .tech-tag.more {
        background-color: var(--card-bg); /* Use variable */
        color: var(--text-light); /* Use variable */
    }
    
    .no-tech {
        color: var(--text-light); /* Use variable */
        font-size: 0.75rem;
        font-style: italic;
    }
    
    .actions {
        display: flex;
        gap: 0.5rem;
    }
    
    /* Global .btn styles are in app.css */
    /* .btn { ... } */
    /* .btn-sm { ... } */
    /* .btn-outline { ... } */
    
    .empty-message {
        padding: 1rem;
        color: var(--text-light); /* Use variable */
        font-style: italic;
        text-align: center;
    }
</style>
