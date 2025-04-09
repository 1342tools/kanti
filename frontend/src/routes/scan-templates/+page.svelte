<script lang="ts">
	import { onMount } from 'svelte';
	import type { SvelteComponent } from 'svelte'; // Import SvelteComponent for casting
	import { scanTemplatesApi } from '$lib/api/api';
	import type { ScanTemplate } from '$lib/types';
	import DataTable from '$lib/components/DataTable.svelte';
	import type { ColumnDefinition } from '$lib/components/DataTable.svelte'; // Import ColumnDefinition type
	import ScanTemplateActions from '$lib/components/ScanTemplateActions.svelte'; // Import actions component
	import AlertMessage from '$lib/components/AlertMessage.svelte';
	import Modal from '$lib/components/Modal.svelte'; // Import Modal
	import ScanTemplateForm from '$lib/components/ScanTemplateForm.svelte'; // Import Form

	let templates: ScanTemplate[] = [];
	let isLoading = true;
	let error: string | null = null;
	let showModal = false;
	let editingTemplate: ScanTemplate | null = null; // Null for create, object for edit
	let modalError: string | null = null; // Error specific to the modal form

	// Define table columns for DataTable component using ColumnDefinition type
	const columns: ColumnDefinition<ScanTemplate>[] = [
		{ key: 'name', label: 'Name' },
		{ key: 'description', label: 'Description' },
		{ key: 'created_at', label: 'Created At', format: (value) => value ? new Date(value).toLocaleString() : '-' },
		{ key: 'updated_at', label: 'Updated At', format: (value) => value ? new Date(value).toLocaleString() : '-' },
		{
			key: '__actions', // Special key, not from data
			label: 'Actions',
			cellClass: 'text-right', // Align actions to the right
			// Return component constructor (cast) and props object
			render: (item) => ({
				component: ScanTemplateActions as typeof SvelteComponent, // Cast to satisfy TS
				props: { item: item } // Pass the current item as a prop
			})
		}
	];

	async function fetchTemplates() {
		isLoading = true;
		error = null;
		try {
			templates = await scanTemplatesApi.getScanTemplates();
		} catch (err: any) {
			error = err.message || 'Failed to load scan templates.';
			templates = []; // Clear templates on error
		} finally {
			isLoading = false;
		}
	}

	function handleCreate() {
		editingTemplate = null; // Ensure we are in create mode
		modalError = null;
		showModal = true;
	}

	function handleViewEdit(template: ScanTemplate) {
		editingTemplate = { ...template }; // Clone template data for editing
		modalError = null;
		showModal = true;
	}

	// This function will be triggered by the 'delete' event from ScanTemplateActions
	async function handleDeleteRequest(event: CustomEvent<ScanTemplate>) {
		const templateToDelete = event.detail;
		if (!confirm(`Are you sure you want to delete the template "${templateToDelete.name}"?`)) {
			return;
		}
		await deleteTemplate(templateToDelete);
	}

	// Extracted delete logic
	async function deleteTemplate(template: ScanTemplate) {
		isLoading = true; // Optional: show loading state during delete
		error = null;
		try {
			await scanTemplatesApi.deleteScanTemplate(template.id);
			await fetchTemplates(); // Refresh list
		} catch (err: any) {
			error = `Failed to delete template "${template.name}": ${err.message}`;
		} finally {
			isLoading = false; // Ensure loading state is reset
		}
	}

	// This function will be triggered by the 'viewEdit' event from ScanTemplateActions
	function handleViewEditRequest(event: CustomEvent<ScanTemplate>) {
		const templateToEdit = event.detail;
		handleViewEdit(templateToEdit); // Call function to open modal
	}

	// Handle form submission from ScanTemplateForm
	async function handleSave(event: CustomEvent<Partial<ScanTemplate>>) {
		const formData = event.detail;
		modalError = null;
		try {
			if (editingTemplate?.id) {
				// Update existing template
				await scanTemplatesApi.updateScanTemplate(editingTemplate.id, formData);
			} else {
				// Create new template (ensure required fields are present - form should handle this)
				await scanTemplatesApi.createScanTemplate(formData as Omit<ScanTemplate, 'id' | 'created_at' | 'updated_at'>);
			}
			showModal = false; // Close modal on success
			await fetchTemplates(); // Refresh the list
		} catch (err: any) {
			modalError = err.message || 'Failed to save template.';
		}
	}

	function handleCancel() {
		showModal = false;
		editingTemplate = null;
		modalError = null;
	}

	onMount(() => {
		fetchTemplates();
	});
</script>

<svelte:head>
	<title>Scan Templates</title>
</svelte:head>

<div class="container mx-auto px-4 py-8">
	<div class="flex justify-between items-center mb-6">
		<h1 class="text-3xl font-bold">Scan Templates</h1>
		<button
			on:click={handleCreate}
			class="bg-blue-600 hover:bg-blue-700 text-white font-bold py-2 px-4 rounded transition duration-150 ease-in-out"
		>
			Create Template
		</button>
	</div>

	{#if isLoading && !showModal} // Don't show main loading if modal is open
		<p>Loading templates...</p>
	{/if}

	{#if error}
		<AlertMessage type="error" message={error} on:dismiss={() => (error = null)} />
	{/if}

	{#if !isLoading && templates.length === 0}
		<p>No scan templates found. Create one to get started!</p>
	{:else if templates.length > 0}
		<!-- Listen for custom events from the rendered ScanTemplateActions component -->
		<DataTable
			{columns}
			data={templates}
			on:viewEdit={handleViewEditRequest}
			on:delete={handleDeleteRequest}
		/>
	{/if}

	<!-- Modal for Create/Edit -->
	{#if showModal}
	<ScanTemplateForm template={editingTemplate} on:save={handleSave} on:cancel={handleCancel} />
	{/if}

</div>
