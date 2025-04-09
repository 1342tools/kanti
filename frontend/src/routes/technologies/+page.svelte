<script lang="ts">
    import { onMount } from 'svelte';
    import { technologiesApi } from '$lib/api/api';
    import type { Technology } from '$lib/types';
    
    let technologies: Technology[] = [];
    let loading = true;
    let error = '';
    
    // Group technologies by category
    let groupedTechnologies: Record<string, Technology[]> = {};
    
    onMount(async () => {
        try {
            technologies = await technologiesApi.getTechnologies();
            groupTechnologies();
            loading = false;
        } catch (err) {
            error = err instanceof Error ? err.message : 'Failed to load technologies';
            loading = false;
        }
    });
    
    function groupTechnologies() {
        groupedTechnologies = technologies.reduce((groups, tech) => {
            const category = tech.category || 'Uncategorized';
            if (!groups[category]) {
                groups[category] = [];
            }
            groups[category].push(tech);
            return groups;
        }, {} as Record<string, Technology[]>);
    }
</script>

<svelte:head>
    <title>Technologies - Attack Surface Management</title>
</svelte:head>

<div class="technologies-page">
    <h1>Technologies</h1>
    
    {#if loading}
        <p>Loading technologies...</p>
    {:else if error}
        <p class="error">{error}</p>
    {:else if Object.keys(groupedTechnologies).length === 0}
        <p class="empty-message">No technologies detected yet.</p>
    {:else}
        <div class="technologies-grid">
            {#each Object.keys(groupedTechnologies).sort() as category}
                <div class="category-card">
                    <h2>{category}</h2>
                    <div class="tech-list">
                        {#each groupedTechnologies[category].sort((a, b) => a.name.localeCompare(b.name)) as tech}
                            <a href={`/technologies/${tech.id}`} class="tech-item">
                                <span class="tech-name">{tech.name}</span>
                            </a>
                        {/each}
                    </div>
                </div>
            {/each}
        </div>
    {/if}
</div>

<style>
    .technologies-page {
        padding: 1rem 0;
    }
    
    h1, h2 {
        color: var(--text); /* Use variable */
    }
    
    h1 {
        margin-bottom: 2rem;
    }
    
    .technologies-grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
        gap: 1.5rem;
    }
    
    .category-card {
        background-color: var(--card-bg); /* Use variable */
        border: 1px solid var(--border); /* Add border */
        border-radius: 8px;
        padding: 1.5rem;
        box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05); /* Keep subtle shadow */
    }
    
    h2 {
        font-size: 1.25rem;
        color: var(--text-light); /* Use variable */
        margin-bottom: 1rem;
        padding-bottom: 0.5rem;
        border-bottom: 1px solid var(--border); /* Use variable */
    }
    
    .tech-list {
        display: flex;
        flex-wrap: wrap;
        gap: 0.5rem;
    }
    
    .tech-item {
        background-color: var(--background); /* Use variable */
        border: 1px solid var(--border); /* Add border */
        padding: 0.5rem 0.75rem;
        border-radius: 4px;
        color: var(--text); /* Use variable */
        text-decoration: none;
        font-size: 0.875rem;
        transition: background-color 0.2s, color 0.2s;
    }
    
    .tech-item:hover {
        background-color: var(--primary); /* Use variable */
        color: white; /* Keep white text on primary hover */
        border-color: var(--primary); /* Use variable */
    }
    
    .empty-message {
        padding: 2rem;
        background-color: var(--card-bg); /* Use variable */
        border-radius: 8px;
        text-align: center;
        color: var(--text-light); /* Use variable */
        border: 1px solid var(--border); /* Add border */
    }
    
    .error {
        color: #ef4444; /* Keep error color */
    }
</style>
