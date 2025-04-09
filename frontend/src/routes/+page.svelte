<script lang="ts">
	import { onMount } from 'svelte';
	import { organizationsApi } from '$lib/api/api'; // Use organizationsApi
	import type { Organization } from '$lib/types'; // Use Organization type
	import AlertMessage from '$lib/components/AlertMessage.svelte'; // For showing errors/success

	let organizations: Organization[] = [];
	let loading = true;
	let error = '';
	let successMessage = '';

	let newOrgName = '';
	let isSubmittingOrg = false;

	onMount(async () => {
		await loadOrganizations();
	});

	async function loadOrganizations() {
		loading = true;
		error = '';
		try {
			organizations = await organizationsApi.getOrganizations();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load organizations';
			console.error(error);
		} finally {
			loading = false;
		}
	}

	async function handleAddOrganization() {
		if (!newOrgName.trim()) {
			error = 'Organization name cannot be empty.';
			return;
		}
		isSubmittingOrg = true;
		error = '';
		successMessage = '';

		try {
			const newOrg = await organizationsApi.createOrganization({ name: newOrgName.trim() });
			organizations = [...organizations, newOrg].sort((a, b) => a.name.localeCompare(b.name)); // Add and sort
			successMessage = `Organization "${newOrg.name}" added successfully.`;
			newOrgName = ''; // Clear input
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to add organization';
			console.error(error);
		} finally {
			isSubmittingOrg = false;
		}
	}
</script>

<svelte:head>
	<title>ASM - Organizations</title>
</svelte:head>

<div class="dashboard">
	<h1>Organizations</h1>

	<section class="add-organization mb-8">
		<h2 class="text-xl font-semibold mb-3">Add New Organization</h2>
		<form on:submit|preventDefault={handleAddOrganization} class="flex items-end gap-3">
			<div class="flex-grow">
				<label for="org-name" class="block text-sm font-medium mb-1">Organization Name</label>
				<input
					type="text"
					id="org-name"
					bind:value={newOrgName}
					placeholder="Enter organization name"
					class="input w-full"
					required
					disabled={isSubmittingOrg}
				/>
			</div>
			<button type="submit" class="btn btn-primary" disabled={isSubmittingOrg}>
				{#if isSubmittingOrg}
					Adding...
				{:else}
					Add Organization
				{/if}
			</button>
		</form>
		{#if successMessage}
			<AlertMessage type="success" message={successMessage} on:dismiss={() => (successMessage = '')} />
		{/if}
		{#if error && !isSubmittingOrg}
			<AlertMessage type="error" message={error} on:dismiss={() => (error = '')} />
		{/if}
	</section>

	<section class="organizations">
		<h2 class="text-xl font-semibold mb-3">Existing Organizations</h2>
		{#if loading}
			<p>Loading organizations...</p>
		{:else if error && organizations.length === 0}
			<!-- Show error only if loading failed and no orgs are present -->
			<AlertMessage type="error" message={error} />
		{:else if organizations.length === 0}
			<p class="text-gray-500 italic">No organizations found. Add one above to get started.</p>
		{:else}
			<div class="organization-grid">
				{#each organizations as org (org.id)}
					<a href={`/organizations/${org.id}`} class="organization-card block hover:shadow-lg transition-shadow duration-200">
						<h3 class="text-lg font-medium mb-1">{org.name}</h3>
						<p class="text-sm text-gray-400">
							Created: {new Date(org.created_at).toLocaleDateString()}
						</p>
					</a>
				{/each}
			</div>
		{/if}
	</section>
</div>

<style>
	/* Use Tailwind utility classes where possible, add specific styles here */
	.dashboard {
		padding: 1rem 0;
	}

	.organization-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(250px, 1fr));
		gap: 1rem;
	}

	.organization-card {
		background-color: var(--card-bg);
		border: 1px solid var(--border);
		border-radius: 4px;
		padding: 1rem 1.5rem;
		color: var(--text);
		text-decoration: none; /* Remove underline from link */
	}
	.organization-card h3 {
		color: var(--text);
	}
	.organization-card p {
		color: var(--text-light);
	}

	/* Input styles might come from app.css or Tailwind */
	.input {
		/* Add base input styles if not globally defined */
		border: 1px solid var(--border);
		background-color: var(--input-bg);
		color: var(--text);
		padding: 0.5rem 0.75rem;
		border-radius: 4px;
	}
	.input:focus {
		outline: none;
		border-color: var(--primary);
		box-shadow: 0 0 0 2px var(--primary-focus);
	}

	/* Ensure button styles are consistent */
	/* .btn, .btn-primary might be in app.css */

	/* Add margin bottom utility class */
	.mb-8 { margin-bottom: 2rem; }
	.mb-3 { margin-bottom: 0.75rem; }
	.mb-1 { margin-bottom: 0.25rem; }
</style>
