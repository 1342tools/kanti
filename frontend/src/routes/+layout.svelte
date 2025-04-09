<script lang="ts">
    import '../app.css';
    import '../app.css';
    import Header from '$lib/components/Header.svelte';
    import Sidebar from '$lib/components/Sidebar.svelte';
    import { theme } from '$lib/stores/theme';
    import { browser } from '$app/environment'; // Import browser check

    let sidebarVisible = false;
    const sidebarWidth = 250; // Define sidebar width as a constant
    const activationThreshold = 100; // Pixels from left edge to activate

    function handleMouseMove(event: MouseEvent) {
        if (!browser) return; // Ensure code only runs in the browser

        if (event.clientX < activationThreshold) {
            sidebarVisible = true;
        } else if (event.clientX >= sidebarWidth) {
            // Hide only if mouse moves off the sidebar area
            sidebarVisible = false;
        }
        // If clientX is between activationThreshold and sidebarWidth, state remains unchanged
        // allowing the mouse to move over the opened sidebar without closing it.
    }
</script>

<svelte:window on:mousemove={handleMouseMove} />

<div class="app" data-theme={$theme}>
    <Header />
    
    <main>
        <div class="container">
            <div class="sidebar-area"> 
                <div class="sidebar {sidebarVisible ? 'visible' : ''}">
                    <Sidebar />
                </div>
            </div>
            <div class="content" style="margin-left: {sidebarVisible ? sidebarWidth + 'px' : '0'}; padding: 1.5rem;">
                <slot />
            </div>
        </div>
    </main>
    
    <footer>
        <div class="container">
            <p>Attack Surface Management Platform &copy; {new Date().getFullYear()}</p>
        </div>
    </footer>
</div>

<style>
    .app {
        display: flex;
        flex-direction: column;
        min-height: 100vh;   
    }
    
    main {
        flex: 1;
    }
    
    .container {
        /* max-width: 1200px; */ /* Removed to allow full width */
        margin: 0 auto;  /* Removed centering margin */
        /* padding: 0 1rem; */ /* Removed padding */
        padding: 0 1rem; /* Explicitly set padding to 0 */
        display: flex;
        width: 100%; /* Ensure container takes full width */
        box-sizing: border-box; /* Include padding in width calculation */
        /* display: flex; */ /* No longer needed as sidebar is fixed */
    }

    .sidebar-area {
        position: relative; /* Context for potential future absolute elements inside */
        z-index: 1000; /* Ensure it's above default content flow */
    }
    
    .sidebar {
        position: fixed;
        top: 60px; /* Adjust based on actual Header height */
        left: 0;
        bottom: 0;
        width: 250px; /* Use the constant if preferred, but direct value is fine here */
        background-color: var(--card-bg);
        transform: translateX(-100%);
        transition: transform 0.3s ease-in-out;
        padding: 1rem; /* Internal padding */
        box-sizing: border-box;
        overflow-y: auto; /* Allow scrolling if content overflows */
    }

    .sidebar.visible {
        transform: translateX(0);
    }
    
    .content {
        flex: 1; /* Still useful if container becomes flex again */
        transition: margin-left 0.3s ease-in-out;
        /* padding is now added inline based on sidebar state */
    }
    
    footer {
        padding: 1rem 0;
        background-color: var(--card-bg); /* Use variable */
        margin-top: 2rem;
    }
</style>
