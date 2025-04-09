<script lang="ts">
    import { page } from '$app/stores';
    import { theme } from '$lib/stores/theme';
    import type { Theme } from '$lib/types';
    
    function toggleTheme(): void {
        theme.update((current: Theme) => current === 'light' ? 'dark' : 'light');
    }
</script>

<header class="header">
    <div class="container">
        <div class="logo">
            <a href="/">
                <span class="logo-text">ASM Platform</span>
            </a>
        </div>
        
        <nav class="main-nav">
            <ul>
                <li class:active={$page.url.pathname === '/'}>
                    <a href="/">Organizations</a>
                </li>
                 <li class:active={$page.url.pathname.startsWith('/domains')}>
                    <a href="/domains">Domains</a>
                </li>
                <li class:active={$page.url.pathname.startsWith('/subdomains')}>
                    <a href="/subdomains">Subdomains</a>
                </li>
                <li class:active={$page.url.pathname.startsWith('/endpoints')}>
                    <a href="/endpoints">Endpoints</a>
                </li>
                <li class:active={$page.url.pathname.startsWith('/technologies')}>
                    <a href="/technologies">Technologies</a>
                </li>
                <li>
                    <button on:click={toggleTheme} class="theme-toggle">
                        {#if $theme === 'light'}
                            <span class="dark-icon">🌙</span>
                        {:else}
                            <span class="light-icon">☀️</span>
                        {/if}
                    </button>
                </li>
            </ul>
        </nav>
    </div>
</header>

<style>
    .header {
        background-color: var(--background); /* Use variable */
        box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1); /* Keep shadow for now, might need theme-specific shadow later */
        position: sticky;
        top: 0;
        z-index: 100;
    }
    
    .container {
        
        margin: 0 auto;
        padding: 0;
        display: flex;
        align-items: center;
        justify-content: space-between;
        height: 64px;
    }
    
    .logo a {
        display: flex;
        align-items: center;
        text-decoration: none;
    }
    
    .logo-text {
        font-size: 1.25rem;
        font-weight: 700;
        color: var(--primary); /* Use variable */
    }
    
    .main-nav ul {
        display: flex;
        list-style: none;
        margin: 0;
        padding: 0;
    }
    
    .main-nav li {
        margin-left: 1.5rem;
    }
    
    .main-nav a {
        color: var(--text-light); /* Use variable */
        text-decoration: none;
        font-weight: 500;
        padding: 0.5rem 0;
        display: inline-block;
        position: relative;
    }
    
    .main-nav a:hover {
        color: var(--primary); /* Use variable */
    }
    
    .main-nav li.active a {
        color: var(--primary); /* Use variable */
    }
    
    .main-nav li.active a::after {
        content: '';
        position: absolute;
        bottom: 0;
        left: 0;
        width: 100%;
        height: 2px;
        background-color: var(--primary); /* Use variable */
        border-radius: 1px;
    }

    .theme-toggle {
        background: none;
        border: none;
        padding: 0;
        font-size: 1.2rem;
        cursor: pointer;
        color: var(--text);
        transition: color 0.3s ease;
    }

    .theme-toggle:hover {
        color: var(--primary);
    }
</style>
