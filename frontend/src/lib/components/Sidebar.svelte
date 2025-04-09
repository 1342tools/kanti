<script lang="ts">
	import { onMount } from 'svelte';
	import { organizationsApi } from '$lib/api/api'; // Use organizationsApi
	import type { Organization } from '$lib/types'; // Use Organization type
	import { page } from '$app/stores';

	let organizations: Organization[] = []; // Change to organizations
	let loading = true;

	onMount(async () => {
		try {
			organizations = await organizationsApi.getOrganizations(); // Fetch organizations
		} catch (error) {
			console.error('Failed to load organizations:', error); // Update error message
        } finally {
			loading = false;
		}
	});

	// Update active state logic for organizations
	$: activeOrganizationId = $page.url.pathname.startsWith('/organizations/')
		? parseInt($page.url.pathname.split('/')[2])
		: null;
</script>

<div class="sidebar">
    <div class="sidebar-section">
		<h3>Navigation</h3> <!-- Changed title -->
		<ul class="action-list">
			<li>
				<a href="/" class="action-item" class:active={$page.url.pathname === '/'}>
					<span>Organizations</span> <!-- Changed text -->
				</a>
			</li>
            <li>
				<a href="/domains" class="action-item" class:active={$page.url.pathname.startsWith('/domains')}>
					<span>All Domains</span>
				</a>
			</li>
			<li>
                <a href="/subdomains" class="action-item">
                    <span>All Subdomains</span>
                </a>
            </li>
            <li>
                <a href="/endpoints" class="action-item">
                    <span>All Endpoints</span>
                </a>
            </li>
            <li>
                <a href="/technologies" class="action-item">
                    <span>All Technologies</span>
                </a>
            </li>
			<li>
                <a href="/scan-templates" class="action-item">
                    <span>Enumeration Templates</span>
                </a>
            </li>
            <li>
                <a href="/settings" class="action-item" class:active={$page.url.pathname === '/settings'}>
                    <span>Settings</span>
                </a>
            </li>
		</ul>
	</div>

	<!-- Changed section to Organizations -->
	<div class="sidebar-section">
		<h3>Organizations</h3>
		{#if loading}
			<p class="loading">Loading organizations...</p>
		{:else if organizations.length === 0}
			<p class="empty">No organizations added yet</p>
		{:else}
			<ul class="organization-list">
				{#each organizations as org (org.id)}
					<li class:active={activeOrganizationId === org.id}>
						<a href={`/organizations/${org.id}`} class="organization-item">
							<span>{org.name}</span>
						</a>
					</li>
				{/each}
			</ul>
		{/if}
	</div>
</div>

<style>
    .sidebar {
        padding: 1rem 0;
    }
    
    .sidebar-section {
        margin-bottom: 2rem;
    }
    
    .sidebar-section h3 {
        font-size: 0.875rem;
        text-transform: uppercase;
        color: var(--text-light); /* Use variable */
        margin-bottom: 0.75rem;
		padding-bottom: 0.25rem;
		border-bottom: 1px solid var(--border); /* Use variable */
	}

	.action-list, .organization-list { /* Renamed class */
		list-style: none;
		padding: 0;
		margin: 0;
	}

	.action-item, .organization-item { /* Renamed class */
		display: block;
		padding: 0.5rem 0.75rem;
		color: var(--text); /* Use variable */
        text-decoration: none;
        border-radius: 4px;
        margin-bottom: 0.25rem;
        font-size: 0.875rem;
		transition: background-color 0.2s, color 0.2s;
	}

	.action-item:hover, .organization-item:hover { /* Renamed class */
		background-color: var(--card-bg); /* Use variable */
		color: var(--primary); /* Use variable */
	}

	/* Updated active class selector */
	li.active .action-item,
	li.active .organization-item,
	a.action-item.active {
		/* Use a mix of primary and background for active state */
		background-color: color-mix(in srgb, var(--primary) 10%, var(--background));
		color: var(--primary); /* Use variable */
        font-weight: 500;
    }
    
    .loading, .empty {
        color: var(--text-light); /* Use variable */
        font-size: 0.875rem;
        font-style: italic;
        padding: 0.5rem 0;
    }
</style>
