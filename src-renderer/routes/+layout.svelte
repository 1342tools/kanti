<script lang="ts">
	import '../app.css';
	import Devtools from '$lib/Devtools.svelte';
	import { preferredTheme } from '$lib/preferredTheme.svelte';

	let { children } = $props();

	// Update title bar colors based on theme
	$effect(() => {
		const isDark = preferredTheme.theme === 'dark';
		const bgColor = isDark ? '#212121' : '#f0f0f0';
		const textColor = isDark ? '#ffffff' : '#1a1a1a';
		window.setTitleBarColors(bgColor, textColor);
	});
</script>

<div id='titlebar' class='shrink-0 flex'>
	<div class='px-4 select-none grow text-xs flex items-center' style='-webkit-app-region: drag;'>Kanti</div>
	{#if import.meta.env.DEV}
		<Devtools/>
	{/if}
</div>
<div class='overflow-auto h-full'>
	{@render children()}
</div>

<style>
	#titlebar {
		margin-right: env(titlebar-area-x);
		width: env(titlebar-area-width);
		height: env(titlebar-area-height);
	}
</style>
