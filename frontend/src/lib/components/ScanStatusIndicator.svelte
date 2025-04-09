<script lang="ts">
    // Removed onMount and onDestroy imports
    import type { Scan } from '$lib/types';
    
    export let scan: Scan;
    
    // Removed all WebSocket related variables
    // Removed WebSocket functions and lifecycle hooks

    // Reactive block to parse results_summary
    let parsedSummary: { subdomains_found?: number; endpoints_found?: number } | null = null;
    $: {
        if (scan.results_summary && typeof scan.results_summary === 'string') {
            try {
                parsedSummary = JSON.parse(scan.results_summary);
            } catch (e) {
                console.error("Failed to parse scan results summary:", e);
                parsedSummary = null; // Reset on error
            }
        } else {
            parsedSummary = null; // Reset if summary is null or not a string
        }
    }
</script>

<div class="scan-status scan-status-{scan.status}">
    <div class="scan-info">
        <span class="scan-type">{scan.scan_type} Scan</span>
        <span class="scan-id">ID: {scan.id}</span>
        <span class="scan-time">
            Started: {new Date(scan.started_at).toLocaleTimeString()}
        </span>
    </div>
    
    {#if scan.status === 'running'}
        <!-- Removed progress bar and dynamic status message -->
        <div class="status-label">
            Scan is currently running...
            <!-- Removed dynamic stats display -->
        </div>
    {:else if scan.status === 'completed'}
        <div class="status-label completed">
            Scan completed at {scan.completed_at ? new Date(scan.completed_at).toLocaleTimeString() : 'Unknown'}
            <!-- Use parsedSummary -->
            {#if parsedSummary}
                <div class="stats">
                    {#if parsedSummary.subdomains_found && parsedSummary.subdomains_found > 0}
                        <span>Subdomains found: {parsedSummary.subdomains_found}</span>
                    {/if}
                    {#if parsedSummary.endpoints_found && parsedSummary.endpoints_found > 0}
                        <span>Endpoints found: {parsedSummary.endpoints_found}</span>
                    {/if}
                </div>
            {/if}
        </div>
    {:else if scan.status === 'failed'}
         <div class="status-label failed">
            Scan failed {scan.completed_at ? `at ${new Date(scan.completed_at).toLocaleTimeString()}` : ''}
        </div>
    {:else} <!-- Handle other potential statuses like 'pending' -->
         <div class="status-label">
            Scan status: {scan.status}
        </div>
    {/if}
</div>

<style>
    /* Define status colors as variables for easier management */
    :root {
        --status-running-bg: #fef3c7;
        --status-running-border: #fcd34d;
        --status-running-text: #b45309;
        --status-completed-bg: #dcfce7;
        --status-completed-border: #86efac;
        --status-completed-text: #16a34a;
        --status-failed-bg: #fee2e2;
        --status-failed-border: #fca5a5;
        --status-failed-text: #dc2626;
        --progress-bar-bg: #fbbf24;
    }

    [data-theme="dark"] {
        /* Define dark theme status colors */
        --status-running-bg: #422006;
        --status-running-border: #d97706;
        --status-running-text: #fcd34d;
        --status-completed-bg: #064e3b;
        --status-completed-border: #10b981;
        --status-completed-text: #a7f3d0;
        --status-failed-bg: #7f1d1d;
        --status-failed-border: #ef4444;
        --status-failed-text: #fca5a5;
        --progress-bar-bg: #f59e0b;
    }

    .scan-status {
        margin: 1rem 0;
        padding: 1rem;
        border-radius: 8px;
        background-color: var(--card-bg); /* Use variable */
        border: 1px solid var(--border); /* Use variable */
        color: var(--text); /* Use variable */
    }
    
    .scan-status-running {
        border-color: var(--status-running-border);
        background-color: var(--status-running-bg);
        color: var(--status-running-text);
    }
    
    .scan-status-completed {
        border-color: var(--status-completed-border);
        background-color: var(--status-completed-bg);
        color: var(--status-completed-text);
    }
    
    .scan-status-failed {
        border-color: var(--status-failed-border);
        background-color: var(--status-failed-bg);
        color: var(--status-failed-text);
    }
    
    .scan-info {
        display: flex;
        flex-wrap: wrap;
        gap: 1rem;
        margin-bottom: 0.5rem;
    }
    
    .scan-type {
        font-weight: 600;
        /* Inherit color from parent .scan-status-* */
        color: inherit; 
    }
    
    .scan-id, .scan-time {
        /* Use text-light for less emphasis */
        color: var(--text-light); 
        font-size: 0.875rem;
    }
    
    .progress-container {
        height: 0.5rem;
        /* Use border color for background */
        background-color: var(--border); 
        border-radius: 0.25rem;
        margin: 0.5rem 0;
        position: relative;
        overflow: hidden;
    }
    
    .progress-bar {
        height: 100%;
        background-color: var(--progress-bar-bg); /* Use variable */
        border-radius: 0.25rem;
        transition: width 0.5s ease-in-out;
    }
    
    .progress-label {
        position: absolute;
        right: 0.5rem;
        top: 0;
        font-size: 0.625rem;
        /* Use text-light for less emphasis */
        color: var(--text-light); 
    }
    
    .status-label {
        font-size: 0.875rem;
        /* Inherit color from parent .scan-status-* */
        color: inherit; 
        margin-top: 0.5rem;
    }
    
    /* Remove specific color overrides as they are handled by parent */
    /* .status-label.completed { ... } */
    /* .status-label.failed { ... } */
    
    .stats {
        display: flex;
        gap: 1rem;
        margin-top: 0.5rem;
        font-size: 0.75rem;
    }
    
    .stats span {
        /* Use card-bg with opacity for subtle background */
        background-color: color-mix(in srgb, var(--card-bg) 70%, transparent); 
        padding: 0.25rem 0.5rem;
        border-radius: 4px;
        /* Inherit color from parent .scan-status-* */
        color: inherit; 
    }
</style>
