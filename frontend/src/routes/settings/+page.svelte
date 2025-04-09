<script lang="ts">
  import { onMount } from 'svelte';
  import { settingsApi } from '$lib/api/api'; // Use the new settingsApi
  import AlertMessage from '$lib/components/AlertMessage.svelte';

  let settings: { [key: string]: string } = {};
  let isLoading = true;
  let error: string | null = null;
  let successMessage: string | null = null;

  // Define the keys for the settings we want to manage
  // Add more keys as needed for different subfinder sources
  const settingKeys = [
    'SHODAN_API_KEY',
    'CENSYS_API_ID',
    'CENSYS_API_SECRET',
    'BINARYEDGE_API_KEY',
    'VIRUSTOTAL_API_KEY',
    'SECURITYTRAILS_API_KEY',
    'CHAOS_API_KEY',
    'GITHUB_TOKEN', // Subfinder uses GITHUB_TOKEN
    'PASSIVETOTAL_USERNAME', // PassiveTotal needs username and key
    'PASSIVETOTAL_API_KEY',
    'ZOOMEYE_API_KEY',
    'FOFA_EMAIL', // Fofa needs email and key
    'FOFA_API_KEY',
    'HUNTER_API_KEY',
    'QUAKE_API_KEY',
    'NETLAS_API_KEY',
    'INTELX_API_KEY', // IntelX needs key and host (host usually defaults)
    'LEAKIX_API_KEY',
    // Add other relevant keys here (Check subfinder docs for latest)
  ];

  onMount(async () => {
    try {
      // Initialize settings object with empty strings
      settingKeys.forEach(key => {
        settings[key] = '';
      });
      // Fetch existing settings
      const fetchedSettings = await settingsApi.getSettings(); // Use settingsApi
      if (fetchedSettings) {
         // Only update keys that are returned from the API
         for (const key in fetchedSettings) {
            if (settings.hasOwnProperty(key)) {
               settings[key] = fetchedSettings[key];
            }
         }
      }
    } catch (err) {
      error = `Failed to load settings: ${err instanceof Error ? err.message : String(err)}`;
      console.error(err);
    } finally {
      isLoading = false;
    }
  });

  async function saveSettings() {
    error = null;
    successMessage = null;
    isLoading = true;
    try {
      // Filter out empty values before sending? Or send all? Sending all for now.
      const settingsToSave = { ...settings };
      await settingsApi.saveSettings(settingsToSave); // Use settingsApi
      successMessage = 'Settings saved successfully!';
    } catch (err) {
      // Assuming HttpError is exported from api.ts or handle appropriately
      // import { HttpError } from '$lib/api/api'; // Might need this import
      error = `Failed to save settings: ${err instanceof Error ? err.message : String(err)}`;
      console.error(err);
    } finally {
      isLoading = false;
      // Clear success message after a few seconds
      if (successMessage) {
        setTimeout(() => {
          successMessage = null;
        }, 3000);
      }
    }
  }
</script>

<svelte:head>
  <title>Settings</title>
</svelte:head>

<div class="container mx-auto p-4">
  <h1 class="text-2xl font-bold mb-4">API Key Settings</h1>

  {#if error}
    <AlertMessage type="error" message={error} />
  {/if}
  {#if successMessage}
    <AlertMessage type="success" message={successMessage} />
  {/if}

  {#if isLoading && !Object.keys(settings).length}
    <p>Loading settings...</p>
  {:else}
    <form on:submit|preventDefault={saveSettings} class="space-y-4">
      {#each settingKeys as key (key)}
        <div>
          <label for={key} class="block text-sm font-medium text-gray-700 dark:text-gray-300">
            {key.replace(/_/g, ' ').replace(/\b\w/g, l => l.toUpperCase())}
          </label>
          <input
            type="password"
            id={key}
            name={key}
            bind:value={settings[key]}
            class="mt-1 block w-full px-3 py-2 bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm"
            placeholder="Enter API Key"
          />
        </div>
      {/each}

      <div>
        <button
          type="submit"
          class="inline-flex justify-center py-2 px-4 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 disabled:opacity-50"
          disabled={isLoading}
        >
          {isLoading ? 'Saving...' : 'Save Settings'}
        </button>
      </div>
    </form>
  {/if}
</div>

<style>
  /* Add any specific styles if needed */
</style>
