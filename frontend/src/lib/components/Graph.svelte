<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { Network } from 'vis-network'; // Import vis-network (changed path)
  import type { Node, Edge, Options, Data } from 'vis-network'; // (changed path)
  import { graphApi } from '../api/api';

  // Interfaces for API data (matching backend response)
  interface ApiNode {
    id: string;
    label: string;
    type: string;
    size: number;
    color: string;
    x: number;
    y: number;
  }

  interface ApiLink {
    from: string;
    to: string;
  }

  interface ApiGraphData {
    nodes: ApiNode[];
    links: ApiLink[];
  }

  let loading = true;
  let error: string | null = null;
  let graphContainer: HTMLDivElement; // Container for vis-network
  let networkInstance: Network | null = null;

  // Define colors locally for the legend (could also fetch from API if needed)
  const colors: { [key: string]: string } = {
    domain: '#ff6b6b',
    subdomain: '#48dbfb',
    endpoint: '#1dd1a1',
    parameter: '#f368e0'
  };

  function initializeGraph(apiData: ApiGraphData): void {
    if (!graphContainer) return;

    // Map API data to vis-network format
    const nodes: Node[] = apiData.nodes.map(node => ({
      id: node.id,
      label: node.label,
      x: node.x,
      y: node.y,
      size: node.size,
      color: node.color,
      // Prevent physics from overriding calculated positions
      fixed: { x: true, y: true }, 
      // Store type for potential future use (e.g., filtering, different interactions)
      group: node.type 
    }));

    const edges: Edge[] = apiData.links.map(link => ({
      from: link.from,
      to: link.to,
      arrows: { to: { enabled: false } }, // No arrows for simplicity
      color: { color: 'var(--border)', opacity: 0.6 } // Use CSS variable
    }));

    const data: Data = { nodes, edges };

    // Vis-network options
    const options: Options = {
      // Disable physics engine as layout is pre-calculated
      physics: false, 
      interaction: {
        // Enable zoom and drag
        dragNodes: true, 
        dragView: true,
        zoomView: true,
        tooltipDelay: 200, // Delay before showing tooltip
        hover: true // Enable hover events for tooltips
      },
      nodes: {
        shape: 'dot', // Use simple dots
        font: {
          color: 'var(--text)', // Use CSS variable for label color
          size: 12 // Adjust font size as needed
        },
        borderWidth: 1,
         borderWidthSelected: 2,
         color: {
             border: 'var(--background)', // Use CSS variable for border
             highlight: {
                 border: 'var(--primary)' // Use primary color for highlight border
             },
             hover: {
                 border: 'var(--primary)' // Use primary color for hover border
             }
         }
      },
      edges: {
        width: 1,
        hoverWidth: 0.2, // Slight increase on hover
        smooth: false // Straight lines are faster
      },
      // Define groups based on type for potential specific styling (optional)
      // groups: { ... } 
    };

    // Destroy previous instance if exists
    if (networkInstance) {
      networkInstance.destroy();
    }

    // Create the network
    networkInstance = new Network(graphContainer, data, options);

    // Add tooltip functionality
    networkInstance.on('showPopup', (params: string) => { // Added type annotation
        const nodeId = params;
        const node = nodes.find(n => n.id === nodeId);
        if (node && graphContainer) {
            const popup = document.getElementById('graph-tooltip');
            if (popup) {
                popup.innerHTML = node.label || ''; // Show node label in tooltip
                popup.style.display = 'block';
                // Position tooltip - requires getting mouse coords, vis-network doesn't provide directly in event
                // For simplicity, we might need a different approach or use vis-network's title attribute
            }
        }
    });

     networkInstance.on('hidePopup', () => {
         const popup = document.getElementById('graph-tooltip');
         if (popup) {
             popup.style.display = 'none';
         }
     });

     // Alternative simpler tooltip using node title attribute
     nodes.forEach(node => {
         node.title = node.label; // Assign label to title for default browser tooltip
     });
     networkInstance.setData({ nodes, edges }); // Update data with titles

  }

  onMount(async () => {
    try {
      const apiData = await graphApi.getGraphData();
      if (!apiData || !apiData.nodes || !apiData.links) {
          throw new Error("Invalid data received from API");
      }
      initializeGraph(apiData);
      error = null;
    } catch (err) {
      error = err instanceof Error ? err.message : 'Failed to load graph data';
      console.error('Error loading or initializing graph:', err);
    } finally {
      loading = false;
    }
  });

  onDestroy(() => {
    // Clean up network instance when component is destroyed
    if (networkInstance) {
      networkInstance.destroy();
      networkInstance = null;
    }
  });
</script>

<div class="graph-container">
  {#if loading}
    <div class="loading-overlay">
      <div class="loading-spinner"></div>
      <div class="loading-text">Loading graph data...</div>
    </div>
  {:else if error}
    <div class="error-overlay">
      <div class="error-text">{error}</div>
      <button on:click={() => window.location.reload()}>Retry</button>
    </div>
  {/if}

  <!-- Container for vis-network -->
  <div bind:this={graphContainer} class="vis-network-container"></div>

  <!-- Simple tooltip element (alternative to title attribute) -->
  <!-- <div id="graph-tooltip" class="tooltip" style="display: none;"></div> -->
  
  <div class="legend">
    <div class="legend-item">
      <span class="legend-dot" style="background-color: {colors.domain};"></span>
      <span>Domain</span>
    </div>
    <div class="legend-item">
      <span class="legend-dot" style="background-color: {colors.subdomain};"></span>
      <span>Subdomain</span>
    </div>
    <div class="legend-item">
      <span class="legend-dot" style="background-color: {colors.endpoint};"></span>
      <span>Endpoint</span>
    </div>
    <div class="legend-item">
      <span class="legend-dot" style="background-color: {colors.parameter};"></span>
      <span>Parameter</span>
    </div>
  </div>
</div>

<style>
  .graph-container {
    position: relative;
    width: 100%;
    height: calc(100vh - 100px); /* Adjust height as needed */
    background-color: var(--card-bg);
    border-radius: 8px;
    overflow: hidden;
  }

  .vis-network-container {
    width: 100%;
    height: 100%;
  }

  /* Make sure vis-network canvas is visible */
  :global(.vis-network canvas) {
      display: block;
      width: 100%;
      height: 100%;
  }


  .loading-overlay,
  .error-overlay {
    position: absolute;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    background-color: color-mix(in srgb, var(--background) 80%, transparent); 
    z-index: 100;
  }

  .loading-spinner {
    border: 4px solid color-mix(in srgb, var(--border) 50%, transparent); 
    border-radius: 50%;
    border-top: 4px solid var(--primary); 
    width: 40px;
    height: 40px;
    animation: spin 1s linear infinite;
    margin-bottom: 10px;
  }

  @keyframes spin {
    0% { transform: rotate(0deg); }
    100% { transform: rotate(360deg); }
  }

  .loading-text,
  .error-text {
    margin-bottom: 15px;
    font-size: 16px;
    color: var(--text);
  }

  .error-overlay button {
    padding: 8px 16px;
    background-color: var(--primary);
    color: white; 
    border: none;
    border-radius: 4px;
    cursor: pointer;
  }

  .error-overlay button:hover {
    background-color: var(--primary-hover);
  }
  
  /* Tooltip styling (if using custom tooltip element) */
  /* .tooltip {
    position: absolute;
    background-color: color-mix(in srgb, var(--card-bg) 90%, black); 
    color: var(--text); 
    padding: 5px 10px;
    border-radius: 4px;
    font-size: 12px;
    pointer-events: none;
    z-index: 10;
    white-space: nowrap;
  } */
  
  .legend {
    position: absolute;
    bottom: 20px;
    right: 20px;
    background-color: color-mix(in srgb, var(--card-bg) 90%, transparent); 
    padding: 10px;
    border-radius: 4px;
    border: 1px solid var(--border); 
    z-index: 5; /* Ensure legend is above graph but below overlays */
  }
  
  .legend-item {
    display: flex;
    align-items: center;
    margin-bottom: 5px;
    font-size: 12px; /* Smaller font for legend */
    color: var(--text-muted); /* Use muted text color */
  }
  
  .legend-dot {
    width: 10px; /* Smaller dots */
    height: 10px;
    border-radius: 50%;
    margin-right: 6px;
  }
</style>
