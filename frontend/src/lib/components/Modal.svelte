<script lang="ts">
    import { createEventDispatcher, onMount } from 'svelte';
    import { fade, scale } from 'svelte/transition';
    
    export let show = false;
    export let title = '';
    export let width = '500px';
    export let closeOnEscape = true;
    export let closeOnOutsideClick = true;
    
    const dispatch = createEventDispatcher<{
        close: void;
    }>();
    
    function close() {
        dispatch('close');
    }
    
    function handleKeydown(event: KeyboardEvent) {
        if (closeOnEscape && event.key === 'Escape' && show) {
            close();
        }
    }
    
    function handleOutsideClick(event: MouseEvent) {
        if (closeOnOutsideClick && event.target === event.currentTarget && show) {
            close();
        }
    }
    
    onMount(() => {
        document.addEventListener('keydown', handleKeydown);
        
        return () => {
            document.removeEventListener('keydown', handleKeydown);
        };
    });
</script>

{#if show}
	<!-- Use Tailwind classes for backdrop styling and positioning -->
	<div
		class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-[1000]"
		on:click={handleOutsideClick}
		transition:fade={{ duration: 200 }}
	>
		<!-- Use Tailwind classes for modal container styling -->
		<div
			class="bg-[var(--background)] rounded-lg shadow-lg w-full max-h-[90vh] flex flex-col"
			style="max-width: {width}"
			transition:scale={{ duration: 200, start: 0.95 }}
		>
			<!-- Use Tailwind classes for header -->
			<div class="p-4 border-b border-[var(--border)] flex items-center justify-between">
				<h2 class="text-xl font-semibold text-[var(--text)]">{title}</h2>
				<!-- Use Tailwind classes for close button -->
				<button
					class="bg-transparent border-none text-2xl cursor-pointer text-[var(--text-light)] p-0 flex items-center justify-center w-8 h-8 rounded-full hover:bg-[var(--card-bg)]"
					on:click={close}>×</button
				>
			</div>
			<!-- Use Tailwind classes for content area -->
			<div class="p-4 overflow-y-auto flex-1 text-[var(--text)]">
				<slot />
			</div>
			<!-- Use Tailwind classes for footer -->
			<div class="p-4 border-t border-[var(--border)] flex justify-end gap-2">
				<slot name="footer">
					<!-- Apply btn and potentially theme-specific styles if needed, or rely on global btn styles -->
					<button
						class="btn bg-[var(--card-bg)] text-[var(--text-light)] border border-[var(--border)] hover:brightness-95 dark:hover:brightness-125"
						on:click={close}>Close</button
					>
				</slot>
			</div>
		</div>
	</div>
{/if}

<!-- Removed the <style> block entirely -->
