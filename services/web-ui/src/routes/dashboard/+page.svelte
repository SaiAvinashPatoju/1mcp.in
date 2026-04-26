<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { goto } from '$app/navigation';
	import { installed, installedCount, runningCount, toggleMcp, uninstallMcp, userCount, startUserCounter, stopUserCounter } from '$lib/stores';
	import McpTile from '$lib/components/McpTile.svelte';
	import ManageModal from '$lib/components/ManageModal.svelte';

	let managingId: string | null = null;
	let pendingDeleteId: string | null = null;
	let animatedUserCount = 0;

	$: managingMcp = managingId ? $installed.find((m) => m.id === managingId) ?? null : null;
	$: pendingDeleteMcp = pendingDeleteId ? $installed.find((m) => m.id === pendingDeleteId) ?? null : null;
	$: disabledCount = $installedCount - $runningCount;

	// Animate user counter
	let countTarget = 0;
	$: countTarget = $userCount;

	let countFrame: number;
	function animateCount() {
		if (animatedUserCount < countTarget) {
			animatedUserCount += Math.ceil((countTarget - animatedUserCount) / 8);
			if (animatedUserCount > countTarget) animatedUserCount = countTarget;
		}
		countFrame = requestAnimationFrame(animateCount);
	}

	onMount(() => {
		animatedUserCount = $userCount;
		startUserCounter();
		countFrame = requestAnimationFrame(animateCount);
	});

	onDestroy(() => {
		stopUserCounter();
		if (typeof cancelAnimationFrame !== 'undefined') cancelAnimationFrame(countFrame);
	});

	function formatCount(n: number): string {
		return n.toLocaleString();
	}
</script>

<div class="p-8">
	<div class="flex items-center justify-between mb-8">
		<div>
			<h1 class="text-xl font-bold text-white/95">Dashboard</h1>
			<p class="text-sm text-white/30 mt-1">
				{$installedCount} server{$installedCount !== 1 ? 's' : ''} installed · {$runningCount} running
			</p>
		</div>
		<button on:click={() => goto('/marketplace')} class="flex items-center gap-2 text-sm px-4 py-2 rounded-lg bg-violet-600 text-white hover:bg-violet-700 transition-colors font-medium">
			+ Add Server
		</button>
	</div>

	<!-- Stats -->
	<div class="grid grid-cols-4 gap-4 mb-8">
		{#each [
			{ label: 'Registered Users', value: formatCount(animatedUserCount), icon: '👥', color: 'text-violet-400', glow: 'shadow-[0_0_20px_rgba(124,58,237,0.15)]' },
			{ label: 'Installed', value: String($installedCount), icon: '📦', color: 'text-blue-400', glow: '' },
			{ label: 'Running', value: String($runningCount), icon: '⚡', color: 'text-emerald-400', glow: '' },
			{ label: 'Disabled', value: String(disabledCount), icon: '⏸', color: 'text-white/30', glow: '' }
		] as stat}
			<div class="bg-white/[0.03] border border-white/[0.06] rounded-xl px-4 py-3 flex items-center gap-3 backdrop-blur-sm {stat.glow}">
				<span class="text-xl">{stat.icon}</span>
				<div>
					<p class="text-xl font-bold text-white/95 tabular-nums">{stat.value}</p>
					<p class="text-xs text-white/30">{stat.label}</p>
				</div>
			</div>
		{/each}
	</div>

	<!-- MCP Grid -->
	{#if $installed.length === 0}
		<div class="flex flex-col items-center justify-center py-32 gap-4 text-center">
			<span class="text-4xl opacity-20">📦</span>
			<div>
				<p class="text-sm font-medium text-white/80">No servers installed</p>
				<p class="text-xs text-white/30 mt-1">Browse the marketplace to add your first MCP server.</p>
			</div>
			<button on:click={() => goto('/marketplace')} class="text-sm px-4 py-2 rounded-lg bg-violet-600 text-white hover:bg-violet-700 transition-colors">
				Browse Marketplace
			</button>
		</div>
	{:else}
		<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
			{#each $installed as mcp (mcp.id)}
				<McpTile
					{mcp}
					onManage={() => (managingId = mcp.id)}
					onDelete={() => (pendingDeleteId = mcp.id)}
					onToggle={() => toggleMcp(mcp.id)}
				/>
			{/each}
		</div>
	{/if}

	<!-- Manage Modal -->
	{#if managingMcp}
		<ManageModal
			mcp={managingMcp}
			onClose={() => (managingId = null)}
			onSave={() => (managingId = null)}
			onToggle={() => toggleMcp(managingMcp.id)}
		/>
	{/if}

	<!-- Delete Confirmation -->
	{#if pendingDeleteMcp}
		<!-- svelte-ignore a11y-click-events-have-key-events -->
		<!-- svelte-ignore a11y-no-static-element-interactions -->
		<div class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm" on:click|self={() => (pendingDeleteId = null)}>
			<div class="w-full max-w-sm bg-[#12121a] border border-white/[0.06] rounded-2xl p-6 shadow-2xl">
				<h3 class="text-sm font-semibold text-white/90 mb-2">Uninstall Server?</h3>
				<p class="text-xs text-white/40 leading-relaxed mb-6">
					This will remove <strong class="text-white/80">{pendingDeleteMcp.name}</strong> from your router. You can reinstall from the marketplace.
				</p>
				<div class="flex gap-3">
					<button on:click={() => (pendingDeleteId = null)} class="flex-1 py-2 rounded-lg border border-white/[0.06] text-sm text-white/40 hover:text-white/80 hover:bg-white/[0.04] transition-colors">Cancel</button>
					<button on:click={() => { uninstallMcp(pendingDeleteMcp.id); pendingDeleteId = null; }} class="flex-1 py-2 rounded-lg bg-red-600 text-white text-sm font-medium hover:bg-red-700 transition-colors">Uninstall</button>
				</div>
			</div>
		</div>
	{/if}
</div>
