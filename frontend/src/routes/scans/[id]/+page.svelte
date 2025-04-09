<script lang="ts">
	// Removed onMount import
	import ScanStatusIndicator from '$lib/components/ScanStatusIndicator.svelte';
	import AlertMessage from '$lib/components/AlertMessage.svelte';
	import type { PageData } from './$types';
	import type { Scan, DiscoveredSubdomain, DiscoveredEndpoint } from '$lib/types'; // Import new types
	import { tick } from 'svelte'; // Import tick for DOM updates before copy

	export let data: PageData; // Corrected data prop type

	let scan: Scan = data.scan; // Use Scan type
	// Removed messages variable
	// Removed socket variable
	let error: string | null = null; // Initialize error to null

	// Reactive declarations for derived data
	$: discoveredSubdomains = scan?.discovered_subdomains || [];
	$: discoveredEndpoints = scan?.discovered_endpoints || [];

	$: urls = discoveredEndpoints
		.map((ep) => {
			// Attempt to construct a full URL. Default to https.
			// Might need more robust logic if http vs https matters significantly here.
			if (ep.subdomain_hostname && ep.path) {
				// Ensure path starts with /
				const path = ep.path.startsWith('/') ? ep.path : `/${ep.path}`;
				// Basic check for http/https in hostname, default to https
				const scheme = ep.subdomain_hostname.startsWith('http') ? '' : 'https://';
				return `${scheme}${ep.subdomain_hostname}${path}`;
			}
			return null; // Cannot construct URL
		})
		.filter((url): url is string => url !== null); // Type guard

    // Deduplicate and sort URLs
    $: uniqueUrls = [...new Set(urls)].sort();

    // Sort subdomains alphabetically by hostname
    $: sortedSubdomains = [...discoveredSubdomains].sort((a, b) => a.hostname.localeCompare(b.hostname));


	let copySubdomainSuccess = false;
	let copyUrlSuccess = false;

	async function copyToClipboard(text: string, type: 'subdomain' | 'url') {
		if (!text) return; // Don't copy if empty
		try {
			await navigator.clipboard.writeText(text);
			if (type === 'subdomain') copySubdomainSuccess = true;
			if (type === 'url') copyUrlSuccess = true;

			// Reset message after a delay
			setTimeout(() => {
				if (type === 'subdomain') copySubdomainSuccess = false;
				if (type === 'url') copyUrlSuccess = false;
			}, 2000);
		} catch (err) {
			console.error('Failed to copy:', err);
			// Optionally show an error message to the user
            error = `Failed to copy ${type}s to clipboard.`;
            setTimeout(() => error = null, 3000);
		}
	}

	async function copySubdomains() {
		const textToCopy = sortedSubdomains.map((s) => s.hostname).join('\n');
		await copyToClipboard(textToCopy, 'subdomain');
	}

	async function copyUrls() {
		const textToCopy = uniqueUrls.join('\n'); // Use uniqueUrls here
		await copyToClipboard(textToCopy, 'url');
	}

	// Removed onMount hook and all WebSocket connection logic
</script>

<div class="scan-container">
	<h1>Scan Details</h1>

	{#if error}
		<AlertMessage type="error" message={error} on:dismiss={() => error = null} />
	{/if}

	<div class="scan-header">
		<!-- ScanStatusIndicator is already themed -->
		<ScanStatusIndicator {scan} />
		<div class="scan-meta">
			<div>ID: {scan.id}</div>
			<div>Type: {scan.scan_type}</div>
			<div>Started: {new Date(scan.started_at).toLocaleString()}</div>
			{#if scan.completed_at}
				<div>Completed: {new Date(scan.completed_at).toLocaleString()}</div>
			{/if}
		</div>
	</div>

    <!-- Discovered Subdomains Section -->
    <section class="discovered-items">
        <div class="section-header">
            <h2>Discovered Subdomains ({sortedSubdomains.length})</h2>
            {#if sortedSubdomains.length > 0}
                <button class="btn btn-secondary btn-sm" on:click={copySubdomains} title="Copy subdomain list">
                    {#if copySubdomainSuccess} Copied! {:else} Copy List {/if}
                </button>
            {/if}
        </div>
        {#if sortedSubdomains.length > 0}
            <ul class="item-list">
                {#each sortedSubdomains as sub (sub.id)}
                    <li>{sub.hostname} {#if sub.ip_address}({sub.ip_address}){/if}</li>
                {/each}
            </ul>
        {:else if scan.status === 'completed'}
            <p>No subdomains were discovered in this scan.</p>
        {:else if scan.status === 'running'}
             <p>Scan in progress...</p>
        {:else}
             <p>Scan did not complete successfully or data is unavailable.</p>
        {/if}
    </section>

    <!-- Discovered URLs Section -->
    <section class="discovered-items">
        <div class="section-header">
            <h2>Discovered URLs ({uniqueUrls.length})</h2> <!-- Use uniqueUrls length -->
             {#if uniqueUrls.length > 0} <!-- Use uniqueUrls length -->
                <button class="btn btn-secondary btn-sm" on:click={copyUrls} title="Copy URL list">
                     {#if copyUrlSuccess} Copied! {:else} Copy List {/if}
                </button>
            {/if}
        </div>
        {#if uniqueUrls.length > 0} <!-- Use uniqueUrls length -->
            <ul class="item-list url-list">
                {#each uniqueUrls as url (url)} <!-- Iterate over uniqueUrls -->
                    <li><a href={url} target="_blank" rel="noopener noreferrer">{url}</a></li>
                {/each}
            </ul>
        {:else if scan.status === 'completed'}
            <p>No URLs were discovered in this scan.</p>
        {:else if scan.status === 'running'}
             <p>Scan in progress...</p>
        {:else}
             <p>Scan did not complete successfully or data is unavailable.</p>
        {/if}
    </section>

	<!-- Removed Progress Updates section -->

	{#if scan.results_summary}
		<div class="scan-results">
			<h2>Raw Results Summary</h2>
			<pre>{scan.results_summary}</pre> <!-- Display raw JSON string -->
		</div>
	{/if}
</div>

<style>
	.scan-container {
		padding: 1rem;
		max-width: 1200px;
		margin: 0 auto;
        display: flex;
        flex-direction: column;
        gap: 2rem; /* Add gap between sections */
	}
	.scan-container h1,
	.scan-container h2 {
		color: var(--text); /* Use variable */
        margin-bottom: 0.5rem; /* Consistent bottom margin for headers */
	}
    .scan-container h1 {
        margin-bottom: 1rem; /* More margin for main title */
    }

	.scan-header {
		display: flex;
		align-items: center;
		gap: 2rem;
		/* margin-bottom: 2rem; */ /* Removed, using container gap */
        padding-bottom: 1rem;
        border-bottom: 1px solid var(--border);
	}

	.scan-meta {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
		color: var(--text-light); /* Use variable */
        font-size: 0.9em;
	}

	.messages {
		margin-top: 0.5rem; /* Reduced top margin */
		border: 1px solid var(--border); /* Use variable */
		border-radius: 4px;
		max-height: 300px; /* Slightly reduced height */
		overflow-y: auto;
		background-color: var(--background); /* Use variable */
	}

	.message {
		padding: 0.5rem 1rem; /* Slightly reduced padding */
		border-bottom: 1px solid var(--border); /* Use variable */
		color: var(--text); /* Use variable */
        font-size: 0.9em;
	}
	.message:last-child {
		border-bottom: none;
	}

	.message.status {
		background-color: var(--card-bg); /* Use variable */
        font-weight: 500;
	}

	.message.progress {
		background-color: color-mix(in srgb, var(--primary) 5%, var(--background));
	}

	.timestamp {
		color: var(--text-light); /* Use variable */
		margin-right: 1rem;
		font-size: 0.9em;
	}

	.scan-results pre {
		background-color: var(--card-bg); /* Use variable */
		color: var(--text); /* Use variable */
		padding: 1rem;
		border-radius: 4px;
		overflow-x: auto;
		border: 1px solid var(--border); /* Add border */
        max-height: 300px;
        font-size: 0.85em;
	}

    /* Styles for new sections */
    .discovered-items {
        background-color: var(--card-bg);
        padding: 1rem 1.5rem;
        border-radius: 6px;
        border: 1px solid var(--border);
    }

    .section-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 1rem;
    }

    .section-header h2 {
        margin-bottom: 0; /* Remove bottom margin from h2 inside header */
    }

    .item-list {
        list-style: none;
        padding: 0;
        margin: 0;
        max-height: 300px;
        overflow-y: auto;
        border: 1px solid var(--border-light);
        border-radius: 4px;
        background-color: var(--background);
    }

    .item-list li {
        padding: 0.5rem 1rem;
        border-bottom: 1px solid var(--border-light);
        font-family: monospace;
        font-size: 0.9em;
        color: var(--text);
        word-break: break-all; /* Prevent long strings from overflowing */
    }

    .item-list li:last-child {
        border-bottom: none;
    }

    .url-list li a {
        color: var(--primary);
        text-decoration: none;
    }
    .url-list li a:hover {
        text-decoration: underline;
    }

    .btn-sm {
        padding: 0.25rem 0.75rem;
        font-size: 0.8rem;
        line-height: 1.2;
    }

    .btn-secondary {
        background-color: var(--secondary-button-bg);
        color: var(--secondary-button-text);
        border: 1px solid var(--secondary-button-border);
    }
    .btn-secondary:hover {
         background-color: var(--secondary-button-hover-bg);
    }

</style>
