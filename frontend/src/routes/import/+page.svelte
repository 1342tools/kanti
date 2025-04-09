<script lang="ts">
  import { importApi } from '$lib/api/api'; // Import the specific API object
  import AlertMessage from '$lib/components/AlertMessage.svelte';
  import { HttpError } from '$lib/api/api'; // Import HttpError for better error handling

  let fileInput: HTMLInputElement;
  let selectedFile: File | null = null;
  let isLoading = false;
  let successMessage: string | null = null;
  let errorMessage: string | null = null;

  // No need for createApi() anymore

  function handleFileSelect(event: Event) {
    const target = event.target as HTMLInputElement;
    if (target.files && target.files.length > 0) {
      selectedFile = target.files[0];
      successMessage = null;
      errorMessage = null;
    } else {
      selectedFile = null;
    }
  }

  async function handleUpload() {
    if (!selectedFile) {
      errorMessage = 'Please select a file first.';
      return;
    }

    if (selectedFile.type !== 'text/plain') {
        errorMessage = 'Please upload a plain text (.txt) file.';
        return;
    }

    isLoading = true;
    successMessage = null;
    errorMessage = null;

    const formData = new FormData();
    formData.append('file', selectedFile);

    try {
      // Use the new importApi function
      const response = await importApi.uploadUrls(formData);

      // postFormData returns parsed data on success
      successMessage = response.message || 'File processed successfully!';
      // Optionally clear the file input
      if (fileInput) {
            fileInput.value = '';
      }
      selectedFile = null;

    } catch (error: any) {
        console.error('Upload error:', error);
        if (error instanceof HttpError) {
            // Use the detailed error message from HttpError
            errorMessage = error.data?.detail || error.message || 'Upload failed.';
        } else {
            errorMessage = error.message || 'An unexpected error occurred during upload.';
        }
    } finally {
      isLoading = false;
    }
  }
</script>

<svelte:head>
  <title>Import URLs/Subdomains</title>
</svelte:head>

<div class="container mx-auto p-4">
  <h1 class="text-2xl font-bold mb-4">Import URLs / Subdomains</h1>

  <p class="mb-4 text-gray-600 dark:text-gray-400">
    Upload a plain text (.txt) file containing one URL or subdomain per line. The system will process the file and add any new discoveries to the database.
  </p>

  {#if successMessage}
    <AlertMessage type="success" message={successMessage} on:dismiss={() => successMessage = null} />
  {/if}
  {#if errorMessage}
    <AlertMessage type="error" message={errorMessage} on:dismiss={() => errorMessage = null} />
  {/if}

  <div class="bg-white dark:bg-gray-800 shadow-md rounded px-8 pt-6 pb-8 mb-4">
    <div class="mb-4">
      <label class="block text-gray-700 dark:text-gray-300 text-sm font-bold mb-2" for="fileInput">
        Select File (.txt)
      </label>
      <input
        bind:this={fileInput}
        class="shadow appearance-none border rounded w-full py-2 px-3 text-gray-700 dark:text-gray-300 dark:bg-gray-700 leading-tight focus:outline-none focus:shadow-outline"
        id="fileInput"
        type="file"
        accept=".txt"
        on:change={handleFileSelect}
        disabled={isLoading}
      />
      {#if selectedFile}
        <p class="text-sm text-gray-500 dark:text-gray-400 mt-2">Selected: {selectedFile.name}</p>
      {/if}
    </div>

    <div class="flex items-center justify-between">
      <button
        class="bg-blue-500 hover:bg-blue-700 text-white font-bold py-2 px-4 rounded focus:outline-none focus:shadow-outline disabled:opacity-50"
        type="button"
        on:click={handleUpload}
        disabled={!selectedFile || isLoading}
      >
        {#if isLoading}
          Processing...
        {:else}
          Upload and Process
        {/if}
      </button>
    </div>
  </div>
</div>

<style>
  /* Add any specific styles if needed, Tailwind handles most */
</style>
