<script lang="ts">
	import { createEventDispatcher, tick } from 'svelte'; // Import tick
	import type { ScanTemplate, ScanSectionConfig, ScanToolConfig } from '$lib/types';

	export let template: Partial<ScanTemplate> | null = null; // Pass existing template for editing

	const dispatch = createEventDispatcher();

	// --- Tool Definitions & Specific Options ---
	// Define known options for specific tools
	interface ToolDefinition {
		name: string;
		options?: Record<string, { type: 'number' | 'boolean' | 'string'; label: string; defaultValue: any }>;
	}

	const availableSubdomainTools: ToolDefinition[] = [
		{
			name: 'subfinder',
			options: {
				threads: { type: 'number', label: 'Threads', defaultValue: 10 },
				timeout: { type: 'number', label: 'Timeout (s)', defaultValue: 30 },
				maxEnumerationTime: { type: 'number', label: 'Max Enum Time (m)', defaultValue: 5 }
			}
		},
		{ name: 'crtsh' } // crtsh doesn't have specific options here
	];
	const availableUrlTools: ToolDefinition[] = [
		{
			name: 'katana',
			options: {
				maxDepth: { type: 'number', label: 'Max Depth', defaultValue: 3 },
				concurrency: { type: 'number', label: 'Concurrency', defaultValue: 10 },
				parallelism: { type: 'number', label: 'Parallelism', defaultValue: 10 },
				rateLimit: { type: 'number', label: 'Rate Limit (req/s)', defaultValue: 150 },
				timeout: { type: 'number', label: 'Timeout (s)', defaultValue: 10 }
			}
		},
		{ name: 'gau' } // Example: gau might not have specific options managed here
	];
	const availableParameterTools: ToolDefinition[] = [
		{ name: 'arjun' } // Example: arjun might not have specific options managed here
	];

	// --- Helper Functions ---

	// Parses options array from ScanToolConfig into a map for specific fields
	function parseOptionsArray(options: string[] | undefined, toolDef: ToolDefinition): Record<string, any> {
		const parsed: Record<string, any> = {};
		// Set defaults first
		if (toolDef.options) {
			for (const key in toolDef.options) {
				parsed[key] = toolDef.options[key].defaultValue;
			}
		}
		// Override with values from options array
		if (options) {
			for (const opt of options) {
				const parts = opt.split('=');
				const key = parts[0].replace(/^-+/, ''); // Remove leading dashes
				if (toolDef.options && toolDef.options[key]) {
					if (parts.length === 2) {
						const valueStr = parts[1];
						switch (toolDef.options[key].type) {
							case 'number':
								parsed[key] = parseInt(valueStr, 10) || toolDef.options[key].defaultValue;
								break;
							case 'boolean':
								parsed[key] = valueStr.toLowerCase() === 'true';
								break;
							default:
								parsed[key] = valueStr;
						}
					} else {
						// Handle boolean flags (presence implies true)
						if (toolDef.options[key].type === 'boolean') {
							parsed[key] = true;
						}
					}
				}
			}
		}
		return parsed;
	}

	// Creates the options array string[] from the specific fields map
	function createOptionsArray(specificOptions: Record<string, any>, toolDef: ToolDefinition): string[] {
		const options: string[] = [];
		if (toolDef.options) {
			for (const key in specificOptions) {
				if (toolDef.options.hasOwnProperty(key)) {
					const value = specificOptions[key];
					// Only add if not default? Or always add? Let's always add for clarity.
					// Add leading dashes back if needed by backend parser (adjust if backend parser changes)
					options.push(`${key}=${value}`);
				}
			}
		}
		return options;
	}

	// Creates default config, initializing specific options map
	function createDefaultToolConfig(toolDef: ToolDefinition): ScanToolConfig & { specificOptions: Record<string, any> } {
		const specificOptions: Record<string, any> = {};
		if (toolDef.options) {
			for (const key in toolDef.options) {
				specificOptions[key] = toolDef.options[key].defaultValue;
			}
		}
		return { enabled: false, options: [], specificOptions };
	}

	function createDefaultSectionConfig(availableTools: ToolDefinition[]): ScanSectionConfig & { tools: Record<string, ScanToolConfig & { specificOptions: Record<string, any> }> } {
		const tools: Record<string, ScanToolConfig & { specificOptions: Record<string, any> }> = {};
		for (const toolDef of availableTools) {
			tools[toolDef.name] = createDefaultToolConfig(toolDef);
		}
		return {
			enabled: false,
			tools: tools
		};
	}

	// Helper to merge existing config with defaults, parsing/creating specific options
	function deepMergeDefaults(
		defaults: ScanSectionConfig & { tools: Record<string, ScanToolConfig & { specificOptions: Record<string, any> }> },
		existing: ScanSectionConfig | undefined,
		toolDefs: ToolDefinition[]
	): ScanSectionConfig & { tools: Record<string, ScanToolConfig & { specificOptions: Record<string, any> }> } {
		if (!existing) return defaults;

		const mergedTools: Record<string, ScanToolConfig & { specificOptions: Record<string, any> }> = { ...defaults.tools };

		for (const toolDef of toolDefs) {
			const toolName = toolDef.name;
			const existingToolConfig = existing.tools[toolName];
			const defaultToolConfig = defaults.tools[toolName];

			if (mergedTools.hasOwnProperty(toolName)) {
				const specificOptions = parseOptionsArray(existingToolConfig?.options, toolDef);
				mergedTools[toolName] = {
					enabled: existingToolConfig?.enabled ?? defaultToolConfig.enabled,
					options: existingToolConfig?.options ?? [], // Keep original options array if present
					specificOptions: specificOptions // Parsed options for the form
				};
			}
		}

		return {
			enabled: existing.enabled ?? defaults.enabled,
			tools: mergedTools
		};
	}


	// --- Component State ---
	// Initialize formData with specificOptions populated
	let formData: Partial<ScanTemplate> & {
		subdomain_scan_config: ScanSectionConfig & { tools: Record<string, ScanToolConfig & { specificOptions: Record<string, any> }> };
		url_scan_config: ScanSectionConfig & { tools: Record<string, ScanToolConfig & { specificOptions: Record<string, any> }> };
		parameter_scan_config: ScanSectionConfig & { tools: Record<string, ScanToolConfig & { specificOptions: Record<string, any> }> };
	} = {
		name: template?.name ?? '',
		description: template?.description ?? '',
		subdomain_scan_config: deepMergeDefaults(createDefaultSectionConfig(availableSubdomainTools), template?.subdomain_scan_config, availableSubdomainTools),
		url_scan_config: deepMergeDefaults(createDefaultSectionConfig(availableUrlTools), template?.url_scan_config, availableUrlTools),
		parameter_scan_config: deepMergeDefaults(createDefaultSectionConfig(availableParameterTools), template?.parameter_scan_config, availableParameterTools),
		tech_detect_enabled: template?.tech_detect_enabled ?? true,
		screenshot_enabled: template?.screenshot_enabled ?? false // Initialize screenshot_enabled
	};

	// Reactive statement to update options array when specificOptions change
	$: {
		for (const toolDef of availableSubdomainTools) {
			const toolName = toolDef.name;
			if (formData.subdomain_scan_config.tools[toolName]) {
				formData.subdomain_scan_config.tools[toolName].options = createOptionsArray(formData.subdomain_scan_config.tools[toolName].specificOptions, toolDef);
			}
		}
		for (const toolDef of availableUrlTools) {
			const toolName = toolDef.name;
			if (formData.url_scan_config.tools[toolName]) {
				formData.url_scan_config.tools[toolName].options = createOptionsArray(formData.url_scan_config.tools[toolName].specificOptions, toolDef);
			}
		}
		// Add similar block for parameter_scan_config if needed
	}


	async function handleSubmit() {
		await tick(); // Ensure reactive updates complete

		// Basic validation
		if (!formData.name?.trim()) {
			alert('Template name is required.');
			return;
		}

		// Prepare data for dispatch, removing the temporary specificOptions
		const dataToSave: Omit<ScanTemplate, 'id' | 'created_at' | 'updated_at'> = {
			name: formData.name,
			description: formData.description,
			tech_detect_enabled: formData.tech_detect_enabled ?? false, // Ensure boolean type
			screenshot_enabled: formData.screenshot_enabled ?? false, // Include screenshot_enabled
			subdomain_scan_config: { enabled: formData.subdomain_scan_config.enabled, tools: {} },
			url_scan_config: { enabled: formData.url_scan_config.enabled, tools: {} },
			parameter_scan_config: { enabled: formData.parameter_scan_config.enabled, tools: {} },
		};

		for (const toolName in formData.subdomain_scan_config.tools) {
			dataToSave.subdomain_scan_config.tools[toolName] = {
				enabled: formData.subdomain_scan_config.tools[toolName].enabled,
				options: formData.subdomain_scan_config.tools[toolName].options
			};
		}
		for (const toolName in formData.url_scan_config.tools) {
			dataToSave.url_scan_config.tools[toolName] = {
				enabled: formData.url_scan_config.tools[toolName].enabled,
				options: formData.url_scan_config.tools[toolName].options
			};
		}
		for (const toolName in formData.parameter_scan_config.tools) {
			dataToSave.parameter_scan_config.tools[toolName] = {
				enabled: formData.parameter_scan_config.tools[toolName].enabled,
				options: formData.parameter_scan_config.tools[toolName].options
			};
		}


		dispatch('save', dataToSave);
	}

	function handleCancel() {
		dispatch('cancel');
	}

	// Removed the unused renderToolConfig helper function

</script>

<form on:submit|preventDefault={handleSubmit} class="space-y-6 p-4 bg-white dark:bg-gray-800 rounded shadow">
	<h2 class="text-xl font-semibold mb-4">{template ? 'Edit' : 'Create'} Scan Template</h2>

	<div>
		<label for="template-name" class="block text-sm font-medium text-gray-700 dark:text-gray-300">Template Name</label>
		<input
			type="text"
			id="template-name"
			bind:value={formData.name}
			required
			class="mt-1 block w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm dark:bg-gray-700 dark:text-white"
		/>
	</div>

	<div>
		<label for="template-description" class="block text-sm font-medium text-gray-700 dark:text-gray-300">Description</label>
		<textarea
			id="template-description"
			bind:value={formData.description}
			rows="3"
			class="mt-1 block w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm dark:bg-gray-700 dark:text-white"
		></textarea>
	</div>

	<!-- Technology Detection Toggle -->
	<div class="pt-2">
		<label class="flex items-center space-x-2">
			<input
				type="checkbox"
				bind:checked={formData.tech_detect_enabled}
				class="rounded text-indigo-600 focus:ring-indigo-500"
			/>
			<span class="text-sm font-medium text-gray-700 dark:text-gray-300">Enable Technology Detection</span>
		</label>
		<p class="text-xs text-gray-500 dark:text-gray-400 pl-6">Detect technologies used by subdomains and endpoints.</p>
	</div>

	<!-- Screenshot Toggle -->
	<div class="pt-2">
		<label class="flex items-center space-x-2">
			<input
				type="checkbox"
				bind:checked={formData.screenshot_enabled}
				class="rounded text-indigo-600 focus:ring-indigo-500"
			/>
			<span class="text-sm font-medium text-gray-700 dark:text-gray-300">Enable Screenshots</span>
		</label>
		<p class="text-xs text-gray-500 dark:text-gray-400 pl-6">Take screenshots of discovered web pages (subdomains/endpoints without file extensions or .html/.php).</p>
	</div>


	<!-- Subdomain Scan Configuration -->
	<fieldset class="border border-gray-300 dark:border-gray-600 p-4 rounded">
		<legend class="text-lg font-medium px-2 text-gray-900 dark:text-gray-100">Subdomain Scanning</legend>
		<div class="space-y-4 mt-2">
			<div>
				<label class="flex items-center space-x-2">
					<input type="checkbox" bind:checked={formData.subdomain_scan_config.enabled} class="rounded text-indigo-600 focus:ring-indigo-500" />
					<span>Enable Subdomain Scanning</span>
				</label>
			</div>
			{#if formData.subdomain_scan_config.enabled}
				<div class="pl-6 space-y-3">
					<h4 class="font-medium">Tools:</h4>
					{#each availableSubdomainTools as toolDef (toolDef.name)}
						{@const toolName = toolDef.name}
						{@const toolConfig = formData.subdomain_scan_config.tools[toolName]}
						<div class="border-l-2 border-gray-200 dark:border-gray-700 pl-3 py-1">
							<label class="flex items-center space-x-2">
								<input type="checkbox" bind:checked={toolConfig.enabled} class="rounded text-indigo-600 focus:ring-indigo-500" />
								<span>{toolName}</span>
							</label>
							{#if toolConfig.enabled && toolDef.options}
								<div class="mt-2 pl-6 grid grid-cols-1 md:grid-cols-2 gap-x-4 gap-y-2">
									{#each Object.entries(toolDef.options) as [optionKey, optionDef]}
										<div class="flex flex-col">
											<label for="{toolName}-{optionKey}-subdomain" class="text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">{optionDef.label}</label>
											{#if optionDef.type === 'number'}
												<input
													type="number"
													id="{toolName}-{optionKey}-subdomain"
													bind:value={toolConfig.specificOptions[optionKey]}
													class="block w-full px-2 py-1 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm dark:bg-gray-700 dark:text-white text-xs"
												/>
											{:else if optionDef.type === 'boolean'}
												<input
													type="checkbox"
													id="{toolName}-{optionKey}-subdomain"
													bind:checked={toolConfig.specificOptions[optionKey]}
													class="rounded text-indigo-600 focus:ring-indigo-500 h-5 w-5 mt-1"
												/>
											{:else} <!-- Assuming string -->
												<input
													type="text"
													id="{toolName}-{optionKey}-subdomain"
													bind:value={toolConfig.specificOptions[optionKey]}
													class="block w-full px-2 py-1 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm dark:bg-gray-700 dark:text-white text-xs"
												/>
											{/if}
										</div>
									{/each}
								</div>
							{/if}
						</div>
					{/each}
				</div>
			{/if}
		</div>
	</fieldset>

	<!-- URL Scan Configuration -->
	<fieldset class="border border-gray-300 dark:border-gray-600 p-4 rounded">
		<legend class="text-lg font-medium px-2 text-gray-900 dark:text-gray-100">URL Discovery</legend>
		<div class="space-y-4 mt-2">
			<div>
				<label class="flex items-center space-x-2">
					<input type="checkbox" bind:checked={formData.url_scan_config.enabled} class="rounded text-indigo-600 focus:ring-indigo-500" />
					<span>Enable URL Discovery</span>
				</label>
			</div>
			{#if formData.url_scan_config.enabled}
				<div class="pl-6 space-y-3">
					<h4 class="font-medium">Tools:</h4>
					{#each availableUrlTools as toolDef (toolDef.name)}
						{@const toolName = toolDef.name}
						{@const toolConfig = formData.url_scan_config.tools[toolName]}
						<div class="border-l-2 border-gray-200 dark:border-gray-700 pl-3 py-1">
							<label class="flex items-center space-x-2">
								<input type="checkbox" bind:checked={toolConfig.enabled} class="rounded text-indigo-600 focus:ring-indigo-500" />
								<span>{toolName}</span>
							</label>
							{#if toolConfig.enabled && toolDef.options}
								<div class="mt-2 pl-6 grid grid-cols-1 md:grid-cols-2 gap-x-4 gap-y-2">
									{#each Object.entries(toolDef.options) as [optionKey, optionDef]}
										<div class="flex flex-col">
											<label for="{toolName}-{optionKey}-url" class="text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">{optionDef.label}</label>
											{#if optionDef.type === 'number'}
												<input
													type="number"
													id="{toolName}-{optionKey}-url"
													bind:value={toolConfig.specificOptions[optionKey]}
													class="block w-full px-2 py-1 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm dark:bg-gray-700 dark:text-white text-xs"
												/>
											{:else if optionDef.type === 'boolean'}
												<input
													type="checkbox"
													id="{toolName}-{optionKey}-url"
													bind:checked={toolConfig.specificOptions[optionKey]}
													class="rounded text-indigo-600 focus:ring-indigo-500 h-5 w-5 mt-1"
												/>
											{:else} <!-- Assuming string -->
												<input
													type="text"
													id="{toolName}-{optionKey}-url"
													bind:value={toolConfig.specificOptions[optionKey]}
													class="block w-full px-2 py-1 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm dark:bg-gray-700 dark:text-white text-xs"
												/>
											{/if}
										</div>
									{/each}
								</div>
							{/if}
						</div>
					{/each}
				</div>
			{/if}
		</div>
	</fieldset>

	<!-- Parameter Scan Configuration -->
	 <fieldset class="border border-gray-300 dark:border-gray-600 p-4 rounded">
		<legend class="text-lg font-medium px-2 text-gray-900 dark:text-gray-100">Parameter Discovery</legend>
		<div class="space-y-4 mt-2">
			<div>
				<label class="flex items-center space-x-2">
					<input type="checkbox" bind:checked={formData.parameter_scan_config.enabled} class="rounded text-indigo-600 focus:ring-indigo-500" />
					<span>Enable Parameter Discovery</span>
				</label>
			</div>
			{#if formData.parameter_scan_config.enabled}
				<div class="pl-6 space-y-3">
					<h4 class="font-medium">Tools:</h4>
					{#each availableParameterTools as toolDef (toolDef.name)}
						{@const toolName = toolDef.name}
						{@const toolConfig = formData.parameter_scan_config.tools[toolName]}
						<div class="border-l-2 border-gray-200 dark:border-gray-700 pl-3 py-1">
							<label class="flex items-center space-x-2">
								<input type="checkbox" bind:checked={toolConfig.enabled} class="rounded text-indigo-600 focus:ring-indigo-500" />
								<span>{toolName}</span>
							</label>
							{#if toolConfig.enabled && toolDef.options}
								<!-- Render specific options similar to Subdomain/URL sections if arjun gets specific options -->
								<div class="mt-1 pl-6 text-xs text-gray-500 dark:text-gray-400">
									(Specific options UI not yet implemented for {toolName})
									<!-- Fallback to generic text input for now if needed -->
									<!--
									<label for="{toolName}-options-param" class="text-sm">Options:</label>
									<input
										type="text"
										id="{toolName}-options-param"
										placeholder="e.g., -t 10, -o output.txt ..."
										value={toolConfig.options?.join(', ') ?? ''}
										on:input={(e) => toolConfig.options = e.currentTarget.value.split(',').map(s => s.trim()).filter(Boolean)}
										class="mt-1 block w-full px-2 py-1 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm dark:bg-gray-700 dark:text-white text-xs"
									/>
									-->
								</div>
							{:else if toolConfig.enabled}
								 <div class="mt-1 pl-6 text-xs text-gray-500 dark:text-gray-400">
									(No specific options defined for {toolName})
								</div>
							{/if}
						</div>
					{/each}
				</div>
			{/if}
		</div>
	</fieldset>

	<!-- Action Buttons -->
	<div class="flex justify-end space-x-3 pt-4">
		<button
			type="button"
			on:click={handleCancel}
			class="bg-gray-200 hover:bg-gray-300 text-gray-800 font-bold py-2 px-4 rounded transition duration-150 ease-in-out dark:bg-gray-600 dark:hover:bg-gray-500 dark:text-white"
		>
			Cancel
		</button>
		<button
			type="submit"
			class="bg-green-600 hover:bg-green-700 text-white font-bold py-2 px-4 rounded transition duration-150 ease-in-out"
		>
			{template ? 'Save Changes' : 'Create Template'}
		</button>
	</div>
</form>
