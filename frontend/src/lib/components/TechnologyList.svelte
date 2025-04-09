<script lang="ts">
    import { createEventDispatcher } from 'svelte';
    import type { Technology } from '$lib/types';
    
    export let technologies: Technology[] = [];
    export let showConfidence = false;
    
    const dispatch = createEventDispatcher<{
        technologySelected: Technology;
    }>();
    
    function handleTechnologyClick(technology: Technology) {
        dispatch('technologySelected', technology);
    }
    
    // Group technologies by category
    $: groupedTechnologies = technologies.reduce((groups, tech) => {
        const category = tech.category || 'Uncategorized';
        if (!groups[category]) {
            groups[category] = [];
        }
        groups[category].push(tech);
        return groups;
    }, {} as Record<string, Technology[]>);
    
    // Sort categories
    $: sortedCategories = Object.keys(groupedTechnologies).sort();
</script>

<div class="technology-list">
    {#if technologies.length === 0}
        <p class="empty-message">No technologies detected.</p>
    {:else}
        {#each sortedCategories as category}
            <div class="tech-category">
                <h3>{category}</h3>
                <div class="tech-items">
                    {#each groupedTechnologies[category] as tech}
                        <div class="tech-item" on:click={() => handleTechnologyClick(tech)}>
                            <span class="tech-name">{tech.name}</span>
                            {#if showConfidence && tech.confidence !== undefined}
                                <div class="confidence-bar" style="--confidence: {Math.round(tech.confidence * 100)}%">
                                    <div class="confidence-fill"></div>
                                    <span class="confidence-value">{Math.round(tech.confidence * 100)}%</span>
                                </div>
                            {/if}
                        </div>
                    {/each}
                </div>
            </div>
        {/each}
    {/if}
</div>

<style>
    .technology-list {
        width: 100%;
    }
    
    .tech-category {
        margin-bottom: 1.5rem;
    }
    
    .tech-category h3 {
        font-size: 1rem;
        color: var(--text-light); /* Use variable */
        margin-bottom: 0.5rem;
        padding-bottom: 0.25rem;
        border-bottom: 1px solid var(--border); /* Use variable */
    }
    
    .tech-items {
        display: flex;
        flex-wrap: wrap;
        gap: 0.5rem;
    }
    
    .tech-item {
        background-color: var(--card-bg); /* Use variable */
        border-radius: 4px;
        padding: 0.5rem;
        cursor: pointer;
        transition: background-color 0.2s;
        border: 1px solid var(--border); /* Add subtle border */
    }
    
    .tech-item:hover {
        /* Use a slightly different background on hover */
        background-color: color-mix(in srgb, var(--card-bg) 80%, var(--background)); 
    }
    
    .tech-name {
        font-weight: 500;
        color: var(--text); /* Use variable */
    }
    
    .confidence-bar {
        margin-top: 0.25rem;
        height: 0.375rem;
        background-color: var(--border); /* Use variable */
        border-radius: 0.25rem;
        position: relative;
        overflow: hidden;
    }
    
    .confidence-fill {
        position: absolute;
        top: 0;
        left: 0;
        height: 100%;
        width: var(--confidence);
        background-color: var(--primary); /* Use variable */
        border-radius: 0.25rem;
    }
    
    .confidence-value {
        display: block;
        font-size: 0.625rem;
        color: var(--text-light); /* Use variable */
        margin-top: 0.125rem;
    }
    
    .empty-message {
        padding: 1rem;
        color: var(--text-light); /* Use variable */
        font-style: italic;
        text-align: center;
    }
</style>
