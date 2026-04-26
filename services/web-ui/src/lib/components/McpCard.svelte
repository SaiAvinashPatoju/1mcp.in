<script lang="ts">
	import type { MarketplaceMcp } from '$lib/types';
	import StarRating from './StarRating.svelte';

	export let mcp: MarketplaceMcp;
	export let onInstall: () => void;
	export let onUninstall: () => void;

	const verificationConfig = {
		verified: { label: 'Verified', textClass: 'text-emerald-400', bgClass: 'bg-emerald-900/20 border-emerald-800/50' },
		unverified: { label: 'Community', textClass: 'text-yellow-400', bgClass: 'bg-yellow-900/20 border-yellow-800/50' },
		pending: { label: 'Pending', textClass: 'text-orange-400', bgClass: 'bg-orange-900/20 border-orange-800/50' }
	} as const;

	const runtimeBadge: Record<string, string> = {
		node: 'bg-emerald-900/30 text-emerald-400',
		python: 'bg-blue-900/30 text-blue-400',
		go: 'bg-cyan-900/30 text-cyan-400',
		binary: 'bg-orange-900/30 text-orange-400'
	};

	function fmt(n: number): string {
		if (n >= 1000) return `${(n / 1000).toFixed(n >= 10000 ? 0 : 1)}k`;
		return String(n);
	}

	$: vc = verificationConfig[mcp.verificationStatus];
</script>

<div class="rounded-xl border border-white/[0.06] bg-white/[0.03] p-5 flex flex-col gap-4 hover:border-white/[0.12] transition-all duration-200 backdrop-blur-sm">
	<div class="flex items-start justify-between gap-3">
		<div class="min-w-0 flex-1">
			<div class="flex items-center gap-2 flex-wrap">
				<h3 class="text-sm font-semibold text-white/90">{mcp.name}</h3>
				<span class="flex items-center gap-1 text-xs px-1.5 py-0.5 rounded border {vc.bgClass} {vc.textClass}">
					{vc.label}
				</span>
			</div>
			<p class="text-xs text-white/30 mt-0.5">by {mcp.author}</p>
		</div>
		<span class="text-xs px-2 py-0.5 rounded font-mono flex-shrink-0 {runtimeBadge[mcp.runtime] ?? runtimeBadge.node}">
			{mcp.runtime}
		</span>
	</div>

	<p class="text-xs text-white/40 leading-relaxed line-clamp-2 flex-1">{mcp.shortDescription}</p>

	<div class="flex flex-wrap gap-1.5">
		{#each mcp.tags.slice(0, 4) as tag}
			<span class="text-xs px-2 py-0.5 rounded bg-white/[0.04] text-white/40 border border-white/[0.06]">#{tag}</span>
		{/each}
	</div>

	<div class="flex items-center gap-3 text-xs text-white/40">
		<div class="flex items-center gap-1.5">
			<StarRating rating={mcp.rating} size={11} />
			<span class="text-white/80 font-medium">{mcp.rating > 0 ? mcp.rating.toFixed(1) : '—'}</span>
			{#if mcp.reviewCount > 0}
				<span>({fmt(mcp.reviewCount)})</span>
			{/if}
		</div>
		<div class="flex items-center gap-1 ml-auto">
			<svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
			{fmt(mcp.downloads)}
		</div>
		<span>v{mcp.version}</span>
	</div>

	{#if mcp.installed}
		<button on:click={onUninstall} class="w-full text-xs py-1.5 px-3 rounded-lg border border-white/[0.06] text-white/40 hover:text-red-400 hover:border-red-900/60 hover:bg-red-900/10 transition-colors">
			Uninstall
		</button>
	{:else}
		<button on:click={onInstall} class="w-full text-xs py-1.5 px-3 rounded-lg bg-violet-600 text-white hover:bg-violet-700 transition-colors font-medium">
			Install
		</button>
	{/if}
</div>
