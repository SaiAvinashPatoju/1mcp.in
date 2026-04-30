<script lang="ts">
	import { toast, type ToastType } from '$lib/toast';

	const typeStyles: Record<ToastType, string> = {
		success: 'bg-emerald-600/90 border-emerald-500/40 text-emerald-100',
		error: 'bg-red-600/90 border-red-500/40 text-red-100',
		warning: 'bg-amber-600/90 border-amber-500/40 text-amber-100',
		info: 'bg-blue-600/90 border-blue-500/40 text-blue-100',
	};

	const typeIcons: Record<ToastType, string> = {
		success: `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/></svg>`,
		error: `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="15" y1="9" x2="9" y2="15"/><line x1="9" y1="9" x2="15" y2="15"/></svg>`,
		warning: `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>`,
		info: `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="12" y1="16" x2="12" y2="12"/><line x1="12" y1="8" x2="12.01" y2="8"/></svg>`,
	};
</script>

<div class="fixed bottom-4 right-4 z-[100] flex flex-col gap-2 pointer-events-none max-w-sm">
	{#each $toast as t (t.id)}
		<div
			class="pointer-events-auto flex items-start gap-2.5 px-4 py-3 rounded-lg border shadow-xl backdrop-blur-sm animate-slide-in {typeStyles[t.type]}"
			role="alert"
		>
			<span class="flex-shrink-0 mt-0.5">{@html typeIcons[t.type]}</span>
			<p class="text-xs leading-relaxed flex-1">{t.message}</p>
			<button
				on:click={() => toast.dismiss(t.id)}
				class="flex-shrink-0 opacity-60 hover:opacity-100 transition-opacity"
				aria-label="Dismiss"
			>
				<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
			</button>
		</div>
	{/each}
</div>

<style>
	@keyframes slide-in {
		from { transform: translateX(100%); opacity: 0; }
		to { transform: translateX(0); opacity: 1; }
	}
	.animate-slide-in { animation: slide-in 0.2s ease-out; }
</style>
