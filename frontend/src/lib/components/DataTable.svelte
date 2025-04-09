<script context="module" lang="ts">
	import type { SvelteComponent } from 'svelte'; // Import SvelteComponent for type safety

	// Define and export type from module context
	export type ColumnDefinition<T = any> = {
		key: string;
		label: string;
		format?: (value: any, item: T) => string;
		// Optional render function for custom cell content
		render?: (item: T) => { component: typeof SvelteComponent; props?: Record<string, any> }; // Function returning component constructor and props
		// Optional class for the column header (th)
		headerClass?: string;
		// Optional class for the column cells (td)
		cellClass?: string;
	};
</script>

<script lang="ts">
	// Regular script block for component logic
	import { createEventDispatcher } from 'svelte';
	// Removed duplicate SvelteComponent import
	// Removed import from svelte/internal

	// Props
	export let data: any[] = [];
	export let columns: ColumnDefinition[] = []; // Use the imported type
	export let pagination = true;
	export let pageSize = 10;

	const dispatch = createEventDispatcher<{
		rowClick: { item: any; index: number };
	}>();
    
    // Pagination state
    let currentPage = 1;
    
    // Computed values
    $: totalPages = pagination ? Math.ceil(data.length / pageSize) : 1;
    $: paginatedData = pagination 
        ? data.slice((currentPage - 1) * pageSize, currentPage * pageSize) 
        : data;
    
    function nextPage() {
        if (currentPage < totalPages) {
            currentPage++;
        }
    }
    
    function prevPage() {
        if (currentPage > 1) {
            currentPage--;
        }
    }
    
    function goToPage(page: number) {
        if (page >= 1 && page <= totalPages) {
            currentPage = page;
        }
    }
    
    function handleRowClick(item: any, index: number) {
        dispatch('rowClick', { item, index });
    }
    
    // Format cell value
	function formatCellValue(item: any, column: ColumnDefinition): string {
		const value = item[column.key];

		// Use format function if provided (pass the whole item for context)
		if (column.format) {
			return column.format(value, item);
		}

		if (value === null || value === undefined) {
            return '-';
        }
        
        if (value instanceof Date) {
            return value.toLocaleString();
        }
        
        return String(value);
    }
</script>

<div class="data-table-container">
    {#if data.length === 0}
        <p class="empty-message">No data available</p>
    {:else}
        <table class="data-table">
			<thead>
				<tr>
					{#each columns as column}
						<th class="{column.headerClass || ''}">{column.label}</th> <!-- Corrected class directive -->
					{/each}
				</tr>
			</thead>
			<tbody>
				{#each paginatedData as item, index}
					<!-- Removed row click handler for now to avoid conflicts with button clicks -->
					<!-- <tr on:click={() => handleRowClick(item, index)}> -->
					<tr>
						{#each columns as column}
							<td class="{column.cellClass || ''}"> <!-- Corrected class directive -->
								{#if column.render}
									<!-- Render custom component if 'render' function is provided -->
									{@const rendered = column.render(item)}
									<svelte:component this={rendered.component} {...(rendered.props || {})} />
								{:else}
									<!-- Otherwise, display formatted value -->
									{formatCellValue(item, column)}
								{/if}
							</td>
						{/each}
					</tr>
				{/each}
			</tbody>
        </table>
        
        {#if pagination && totalPages > 1}
            <div class="pagination">
                <button class="pagination-btn" disabled={currentPage === 1} on:click={prevPage}>
                    Previous
                </button>
                
                <div class="page-numbers">
                    {#if totalPages <= 7}
                        {#each Array(totalPages) as _, i}
                            <button 
                                class="page-number" 
                                class:active={currentPage === i + 1}
                                on:click={() => goToPage(i + 1)}
                            >
                                {i + 1}
                            </button>
                        {/each}
                    {:else}
                        <!-- First page -->
                        <button 
                            class="page-number" 
                            class:active={currentPage === 1}
                            on:click={() => goToPage(1)}
                        >
                            1
                        </button>
                        
                        <!-- Ellipsis or pages before current -->
                        {#if currentPage > 3}
                            <span class="ellipsis">...</span>
                        {:else}
                            <button 
                                class="page-number" 
                                class:active={currentPage === 2}
                                on:click={() => goToPage(2)}
                            >
                                2
                            </button>
                        {/if}
                        
                        <!-- Pages around current -->
                        {#each Array(3) as _, i}
                            {#if currentPage - 1 + i > 1 && currentPage - 1 + i < totalPages}
                                <button 
                                    class="page-number" 
                                    class:active={currentPage === currentPage - 1 + i}
                                    on:click={() => goToPage(currentPage - 1 + i)}
                                >
                                    {currentPage - 1 + i}
                                </button>
                            {/if}
                        {/each}
                        
                        <!-- Ellipsis or pages after current -->
                        {#if currentPage < totalPages - 2}
                            <span class="ellipsis">...</span>
                        {:else}
                            <button 
                                class="page-number" 
                                class:active={currentPage === totalPages - 1}
                                on:click={() => goToPage(totalPages - 1)}
                            >
                                {totalPages - 1}
                            </button>
                        {/if}
                        
                        <!-- Last page -->
                        <button 
                            class="page-number" 
                            class:active={currentPage === totalPages}
                            on:click={() => goToPage(totalPages)}
                        >
                            {totalPages}
                        </button>
                    {/if}
                </div>
                
                <button class="pagination-btn" disabled={currentPage === totalPages} on:click={nextPage}>
                    Next
                </button>
            </div>
        {/if}
    {/if}
</div>

<style>
    .data-table-container {
        width: 100%;
        overflow-x: auto;
    }
    
    .data-table {
        width: 100%;
        border-collapse: collapse;
        font-size: 0.875rem;
    }
    
    .data-table th, .data-table td {
        padding: 0.75rem;
        text-align: left;
        border-bottom: 1px solid var(--border); /* Use variable */
    }
    
    .data-table th {
        font-weight: 600;
        color: var(--text-light); /* Use variable */
        background-color: var(--card-bg); /* Use variable */
    }
    
    .data-table tbody tr {
        cursor: pointer;
        transition: background-color 0.2s;
    }
    
    .data-table tbody tr:hover {
        background-color: var(--card-bg); /* Use variable */
    }
    
    .pagination {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-top: 1rem;
        padding: 0.5rem 0;
    }
    
    .pagination-btn {
        padding: 0.375rem 0.75rem;
        background-color: var(--card-bg); /* Use variable */
        border: 1px solid var(--border); /* Use variable */
        border-radius: 4px;
        color: var(--text-light); /* Use variable */
        font-size: 0.875rem;
        cursor: pointer;
    }
    
    .pagination-btn:disabled {
        opacity: 0.5;
        cursor: not-allowed;
    }
    
    .page-numbers {
        display: flex;
        gap: 0.25rem;
    }
    
    .page-number {
        width: 2rem;
        height: 2rem;
        display: flex;
        align-items: center;
        justify-content: center;
        background: none;
        border: 1px solid var(--border); /* Use variable */
        border-radius: 4px;
        color: var(--text-light); /* Use variable */
        font-size: 0.875rem;
        cursor: pointer;
    }
    
    .page-number.active {
        background-color: var(--primary); /* Use variable */
        color: white; /* Keep white */
        border-color: var(--primary); /* Use variable */
    }
    
    .ellipsis {
        width: 2rem;
        height: 2rem;
        display: flex;
        align-items: center;
        justify-content: center;
        color: var(--text-light); /* Use variable */
    }
    
    .empty-message {
        padding: 2rem;
        text-align: center;
        color: var(--text-light); /* Use variable */
        font-style: italic;
        background-color: var(--card-bg); /* Use variable */
        border-radius: 4px;
    }
</style>
