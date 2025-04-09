// frontend/src/routes/organizations/[id]/+page.ts
import { error } from '@sveltejs/kit';
import { organizationsApi } from '$lib/api/api';
import type { PageLoad } from './$types'; // Import PageLoad type
import type { OrganizationDetail } from '$lib/types';

export const load: PageLoad = async ({ params, fetch }) => {
	// `fetch` is passed by SvelteKit for server-side requests during SSR or client-side fetch
	// We use our custom api wrapper which uses the global fetch

	const orgId = parseInt(params.id, 10);
	if (isNaN(orgId)) {
		throw error(400, 'Invalid Organization ID'); // Use SvelteKit's error helper
	}

	try {
		const organization = await organizationsApi.getOrganization(orgId);
		return {
			organization: organization as OrganizationDetail // Cast or validate if necessary
		};
	} catch (err) {
		console.error(`Failed to load organization ${orgId}:`, err);
		const errorMessage = err instanceof Error ? err.message : 'Organization not found or failed to load';
		// Throw a SvelteKit error to render the nearest +error.svelte page
		// Adjust status code based on the actual error if possible (e.g., 404 for not found)
		if (errorMessage.toLowerCase().includes('not found')) {
			throw error(404, `Organization with ID ${orgId} not found`);
		} else {
			throw error(500, `Failed to load organization: ${errorMessage}`);
		}
		// Alternatively, return an error prop for the page to handle:
		// return {
		//  organization: null,
		//  error: `Failed to load organization: ${errorMessage}`
		// };
	}
};
