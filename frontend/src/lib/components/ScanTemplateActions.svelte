<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import type { ScanTemplate } from '$lib/types';

	export let item: ScanTemplate; // The template data for the current row

	const dispatch = createEventDispatcher<{
		viewEdit: ScanTemplate;
		delete: ScanTemplate;
	}>();

	function handleViewEditClick(event: MouseEvent) {
		event.stopPropagation(); // Prevent triggering row click if it exists
		dispatch('viewEdit', item);
	}

	function handleDeleteClick(event: MouseEvent) {
		event.stopPropagation(); // Prevent triggering row click if it exists
		dispatch('delete', item);
	}
</script>

<div class="flex space-x-2">
	<button
		on:click={handleViewEditClick}
		class="text-indigo-600 hover:text-indigo-900 dark:text-indigo-400 dark:hover:text-indigo-200 text-sm px-2 py-1 rounded hover:bg-indigo-100 dark:hover:bg-gray-700 transition-colors"
		aria-label="View or Edit {item.name}"
	>
		View/Edit
	</button>
	<button
		on:click={handleDeleteClick}
		class="text-red-600 hover:text-red-900 dark:text-red-400 dark:hover:text-red-200 text-sm px-2 py-1 rounded hover:bg-red-100 dark:hover:bg-gray-700 transition-colors"
		aria-label="Delete {item.name}"
	>
		Delete
	</button>
</div>
