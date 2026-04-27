<script lang="ts">
	import type { InstalledMcp } from '$lib/types';

	export let mcp: InstalledMcp;
	export let onManage: () => void;
	export let onDelete: () => void;
	export let onToggle: () => void;

	const runtimeColors: Record<string, string> = {
		node: 'bg-emerald-900/40 text-emerald-400 border-emerald-800/60',
		python: 'bg-blue-900/40 text-blue-400 border-blue-800/60',
		go: 'bg-cyan-900/40 text-cyan-400 border-cyan-800/60',
		binary: 'bg-orange-900/40 text-orange-400 border-orange-800/60'
	};
</script>

<div class="rounded-xl border border-white/[0.06] bg-white/[0.03] p-5 flex flex-col gap-4 hover:border-white/[0.12] transition-all duration-200 backdrop-blur-sm">
	<div class="flex items-start justify-between gap-3">
		<div class="min-w-0 flex-1">
			<h3 class="text-sm font-semibold text-white/90 truncate">{mcp.name}</h3>
			<div class="flex items-center gap-2 mt-1.5 flex-wrap">
				<span class="text-xs text-white/40">v{mcp.version}</span>
				<span class="text-xs px-1.5 py-0.5 rounded font-mono border {runtimeColors[mcp.runtime] ?? runtimeColors.node}">
					{mcp.runtime}
				</span>
			</div>
		</div>
		<button
			on:click={() => {
				if (mcp.id !== 'mach1') onToggle();
			}}
			title={mcp.id === 'mach1' ? 'Mach1 must remain active' : mcp.enabled ? 'Disable' : 'Enable'}
			class="flex-shrink-0 w-8 h-8 rounded-lg flex items-center justify-center transition-colors
				{mcp.enabled ? 'bg-emerald-900/30 text-emerald-400 hover:bg-emerald-900/50' : 'bg-white/[0.04] text-white/30 hover:bg-white/[0.08] hover:text-white/60'}
				{mcp.id === 'mach1' ? 'opacity-50 cursor-not-allowed' : ''}"
		>
			<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
				<path d="M18.36 6.64a9 9 0 1 1-12.73 0"/><line x1="12" y1="2" x2="12" y2="12"/>
			</svg>
		</button>
	</div>

	<div class="flex items-center gap-1.5">
		<span class="w-1.5 h-1.5 rounded-full {mcp.enabled ? 'bg-emerald-400 shadow-[0_0_6px_#34d399]' : 'bg-white/20'}"></span>
		<span class="text-xs {mcp.enabled ? 'text-emerald-400' : 'text-white/30'}">
			{#if mcp.id === 'mach1'}
				Alive & Routing
			{:else if mcp.enabled}
				Running
			{:else}
				Sleeping (Auto-activated by Mach1)
			{/if}
		</span>
	</div>

	<p class="text-xs text-white/40 leading-relaxed line-clamp-2 flex-1">{mcp.description}</p>

	<div class="flex items-center gap-2">
		<button
			on:click={onManage}
			class="flex-1 flex items-center justify-center gap-1.5 text-xs py-1.5 px-3 rounded-lg bg-violet-600/15 text-violet-400 border border-violet-600/25 hover:bg-violet-600/25 transition-colors font-medium"
		>
			<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
				<circle cx="12" cy="12" r="3"/><path d="M12 1v2m0 18v2M4.22 4.22l1.42 1.42m12.73 12.73 1.42 1.42M1 12h2m18 0h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42"/>
			</svg>
			Manage
		</button>
		<button
			on:click={onDelete}
			title="Uninstall"
			class="w-8 h-8 flex items-center justify-center rounded-lg text-white/30 hover:text-red-400 hover:bg-red-900/20 transition-colors"
		>
			<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
				<polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/>
			</svg>
		</button>
	</div>
</div>
