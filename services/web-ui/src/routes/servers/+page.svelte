<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { toast } from '$lib/toast';
	import {
		installed,
		mcpServers,
		selectedServerId,
		serverDetail,
		serverTools,
		serverLogs,
		serverConfig,
		fetchInstalled,
		fetchMcpServers,
		selectServer,
		scanServer,
		restartSingleServer,
		uninstallSingleServer,
		toggleMcp,
		startDashboardSync
	} from '$lib/stores';

	type Tab = 'installed' | 'running' | 'updates' | 'disabled';
	type SortOption = 'name' | 'status' | 'lastUsed';
	type ViewMode = 'list' | 'grid';

	let activeTab: Tab = 'installed';
	let query = '';
	let runtimeFilter = 'all';
	let statusFilter = 'all';
	let sort: SortOption = 'name';
	let viewMode: ViewMode = 'list';
	let detailTab: 'overview' | 'tools' | 'config' | 'environment' | 'logs' = 'overview';
	let scanLoading = false;
	let serverLoading = false;
	let stopSync: (() => void) | null = null;

	onMount(() => {
		(async () => {
			await fetchInstalled();
			await fetchMcpServers();
			stopSync = startDashboardSync();
		})();
		return () => { if (stopSync) stopSync(); };
	});

	function formatTimeAgo(timestamp: string | null): string {
		if (!timestamp) return '—';
		const diff = Date.now() - new Date(timestamp).getTime();
		const minutes = Math.floor(diff / 60000);
		const hours = Math.floor(diff / 3600000);
		const days = Math.floor(diff / 86400000);
		if (minutes < 1) return 'just now';
		if (minutes < 60) return `${minutes}m ago`;
		if (hours < 24) return `${hours}h ago`;
		return `${days}d ago`;
	}

	function formatUptime(seconds: number): string {
		const h = Math.floor(seconds / 3600);
		const m = Math.floor((seconds % 3600) / 60);
		const s = seconds % 60;
		return `${h}h ${m}m ${s}s`;
	}

	function runtimeClass(runtime: string): string {
		switch (runtime) {
			case 'node': return 'bg-emerald-500/10 text-emerald-400';
			case 'python': return 'bg-blue-500/10 text-blue-400';
			case 'binary': return 'bg-orange-500/10 text-orange-400';
			case 'go': return 'bg-cyan-500/10 text-cyan-400';
			default: return 'bg-white/5 text-white/40';
		}
	}

	function statusClass(status: string): string {
		switch (status) {
			case 'running': return 'text-emerald-400';
			case 'sleeping': return 'text-white/40';
			case 'error': return 'text-red-400';
			default: return 'text-white/40';
		}
	}

	function statusDot(status: string): string {
		switch (status) {
			case 'running': return 'bg-emerald-500';
			case 'sleeping': return 'bg-white/30';
			case 'error': return 'bg-red-500';
			default: return 'bg-white/20';
		}
	}

	function trustClass(trust: string): string {
		switch (trust) {
			case '1mcp-verified':
			case 'verified':
				return 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20';
			case 'anthropic-official':
				return 'bg-blue-500/10 text-blue-400 border-blue-500/20';
			case 'internal':
				return 'bg-white/5 text-white/40 border-white/10';
			case 'community':
				return 'bg-amber-500/10 text-amber-400 border-amber-500/20';
			default:
				return 'bg-white/5 text-white/40 border-white/10';
		}
	}

	$: servers = $mcpServers.length > 0 ? $mcpServers : $installed.map(m => ({
		id: m.id,
		name: m.name,
		description: m.description,
		version: m.version,
		runtime: m.runtime,
		status: m.enabled ? 'running' : 'sleeping' as 'running' | 'sleeping' | 'error',
		status_detail: m.enabled ? 'PID 21340' : 'Idle',
		trust: m.id === 'mach1' ? 'internal' : '1mcp-verified',
		author: m.id === 'mach1' ? '1mcp.in' : 'Anthropic',
		lifecycle: m.id === 'mach1' ? 'Manual' : 'Auto (lazy)',
		idle_timeout: m.id === 'mach1' ? undefined : '15 minutes',
		last_used_at: null,
		last_used_by: undefined,
		process: m.enabled ? { pid: 21340, memory_mb: 64.2, cpu_percent: 0.3, uptime_seconds: 8640, restarts: 0 } : undefined,
		tools_count: m.id === 'github' ? 37 : m.id === 'postgres' ? 12 : m.id === 'fetch' ? 8 : m.id === 'memory' ? 15 : m.id === 'filesystem' ? 10 : m.id === 'git' ? 9 : 0,
		installed_at: m.installed_at ? new Date(m.installed_at * 1000).toISOString() : new Date().toISOString()
	}));

	$: filtered = (() => {
		let result = [...servers];
		if (activeTab === 'running') result = result.filter(s => s.status === 'running');
		if (activeTab === 'disabled') result = result.filter(s => s.status === 'sleeping');
		if (query.trim()) {
			const q = query.toLowerCase();
			result = result.filter(s =>
				s.name.toLowerCase().includes(q) ||
				s.description.toLowerCase().includes(q) ||
				s.id.toLowerCase().includes(q)
			);
		}
		if (runtimeFilter !== 'all') result = result.filter(s => s.runtime === runtimeFilter);
		if (statusFilter !== 'all') result = result.filter(s => s.status === statusFilter);
		result.sort((a, b) => {
			if (sort === 'name') return a.name.localeCompare(b.name);
			if (sort === 'status') return a.status.localeCompare(b.status);
			return 0;
		});
		return result;
	})();

	$: installedCount = servers.length;
	$: runningCount = servers.filter(s => s.status === 'running').length;
	$: sleepingCount = servers.filter(s => s.status === 'sleeping').length;
	$: errorCount = servers.filter(s => s.status === 'error').length;

	async function handleScan() {
		scanLoading = true;
		try {
			if ($selectedServerId) {
				await scanServer($selectedServerId);
			}
			await fetchMcpServers();
			await fetchInstalled();
		} catch (e) {
			console.error('Scan failed', e);
		}
		scanLoading = false;
	}

	async function handleRestart(id: string) {
		serverLoading = true;
		try {
			await restartSingleServer(id);
			await fetchMcpServers();
		} catch (e) {
			console.error('Restart failed', e);
		}
		serverLoading = false;
	}

	async function handleUninstall(id: string) {
		serverLoading = true;
		try {
			await uninstallSingleServer(id);
			selectServer(null);
			await fetchMcpServers();
			await fetchInstalled();
		} catch (e) {
			console.error('Uninstall failed', e);
		}
		serverLoading = false;
	}

	function handleToggle(id: string) {
		toggleMcp(id);
		fetchMcpServers();
	}
</script>

<div class="flex h-full">
	<!-- Main Content -->
	<div class="flex-1 flex flex-col min-w-0" class:pr-80={$selectedServerId}>
		<div class="p-6 space-y-5">
			<!-- Header -->
			<div class="flex items-start justify-between">
				<div>
					<h1 class="text-xl font-bold text-white/95">Servers</h1>
					<p class="text-sm text-white/30 mt-1">
						Manage installed MCP servers.
						<span class="text-white/50">{installedCount} installed</span>,
						<span class="text-emerald-400">{runningCount} running</span>,
						<span class="text-white/40">{sleepingCount} sleeping</span>.
					</p>
				</div>
				<div class="flex items-center gap-3">
					<button on:click={() => goto('/marketplace')} class="flex items-center gap-2 px-3 py-1.5 rounded-lg border border-orange-500/30 text-orange-400 text-xs font-medium hover:bg-orange-500/10 transition-colors">
						<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
						Add Server
					</button>
					<button on:click={handleScan} disabled={scanLoading} class="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-white/[0.03] border border-white/[0.06] text-xs text-white/60 hover:text-white/90 hover:bg-white/[0.06] transition-colors disabled:opacity-50">
						<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class={scanLoading ? 'animate-spin' : ''}><path d="M21 12a9 9 0 0 0-9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/><path d="M3 3v5h5"/><path d="M3 12a9 9 0 0 0 9 9 9.75 9.75 0 0 0 6.74-2.74L21 16"/><path d="M16 16h5v5"/></svg>
						Scan for Changes
					</button>
				</div>
			</div>

			<!-- Tabs -->
			<div class="flex items-center gap-1 border-b border-white/[0.06]">
				{#each ['installed', 'running', 'updates', 'disabled'] as tab}
					<button
						on:click={() => activeTab = tab as Tab}
						class="px-4 py-2 text-sm transition-colors relative {activeTab === tab ? 'text-orange-400 font-medium' : 'text-white/30 hover:text-white/60'}"
					>
						{tab.charAt(0).toUpperCase() + tab.slice(1)}
						{#if tab === 'running'}
							<span class="ml-1 text-xs text-white/40">({runningCount})</span>
						{:else if tab === 'disabled'}
							<span class="ml-1 text-xs text-white/40">({sleepingCount})</span>
						{:else if tab === 'updates'}
							<span class="ml-1 text-xs text-white/40">(0)</span>
						{/if}
						{#if activeTab === tab}
							<div class="absolute bottom-0 left-0 right-0 h-0.5 bg-orange-500 rounded-t-full"></div>
						{/if}
					</button>
				{/each}
			</div>

			<!-- Stats Cards -->
			<div class="grid grid-cols-4 gap-4">
				<div class="rounded-xl bg-white/[0.02] border border-white/[0.06] p-4 flex items-center gap-3">
					<div class="w-10 h-10 rounded-lg bg-orange-500/10 text-orange-400 flex items-center justify-center">
						<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="2" y="2" width="20" height="8" rx="2" ry="2"/><rect x="2" y="14" width="20" height="8" rx="2" ry="2"/><line x1="6" y1="6" x2="6.01" y2="6"/><line x1="6" y1="18" x2="6.01" y2="18"/></svg>
					</div>
					<div>
						<p class="text-xl font-bold text-white/90">{installedCount}</p>
						<p class="text-xs text-white/30">Installed</p>
					</div>
				</div>
				<div class="rounded-xl bg-white/[0.02] border border-white/[0.06] p-4 flex items-center gap-3">
					<div class="w-10 h-10 rounded-lg bg-emerald-500/10 text-emerald-400 flex items-center justify-center">
						<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="5 3 19 12 5 21 5 3"/></svg>
					</div>
					<div>
						<p class="text-xl font-bold text-white/90">{runningCount}</p>
						<p class="text-xs text-white/30">Running</p>
					</div>
				</div>
				<div class="rounded-xl bg-white/[0.02] border border-white/[0.06] p-4 flex items-center gap-3">
					<div class="w-10 h-10 rounded-lg bg-white/5 text-white/40 flex items-center justify-center">
						<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/></svg>
					</div>
					<div>
						<p class="text-xl font-bold text-white/90">{sleepingCount}</p>
						<p class="text-xs text-white/30">Sleeping</p>
					</div>
				</div>
				<div class="rounded-xl bg-white/[0.02] border border-white/[0.06] p-4 flex items-center gap-3">
					<div class="w-10 h-10 rounded-lg bg-red-500/10 text-red-400 flex items-center justify-center">
						<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>
					</div>
					<div>
						<p class="text-xl font-bold text-white/90">{errorCount}</p>
						<p class="text-xs text-white/30">Errored</p>
					</div>
				</div>
			</div>

			<!-- Filters -->
			<div class="flex items-center gap-3">
				<div class="relative flex-1 max-w-sm">
					<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="absolute left-3 top-1/2 -translate-y-1/2 text-white/25"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>
					<input bind:value={query} placeholder="Search servers..." class="w-full bg-white/[0.03] border border-white/[0.06] rounded-lg pl-9 pr-3 py-2 text-sm text-white/80 placeholder-white/20 focus:outline-none focus:border-orange-500/30 transition-colors" />
				</div>
				<select bind:value={runtimeFilter} class="bg-white/[0.03] border border-white/[0.06] rounded-lg px-3 py-2 text-xs text-white/60 focus:outline-none focus:border-orange-500/30">
					<option value="all">All Runtimes</option>
					<option value="node">Node</option>
					<option value="python">Python</option>
					<option value="binary">Binary</option>
					<option value="go">Go</option>
				</select>
				<select bind:value={statusFilter} class="bg-white/[0.03] border border-white/[0.06] rounded-lg px-3 py-2 text-xs text-white/60 focus:outline-none focus:border-orange-500/30">
					<option value="all">All Statuses</option>
					<option value="running">Running</option>
					<option value="sleeping">Sleeping</option>
					<option value="error">Error</option>
				</select>
				<select bind:value={sort} class="bg-white/[0.03] border border-white/[0.06] rounded-lg px-3 py-2 text-xs text-white/60 focus:outline-none focus:border-orange-500/30">
					<option value="name">Name (A-Z)</option>
					<option value="status">Status</option>
					<option value="lastUsed">Last Used</option>
				</select>
				<div class="flex items-center gap-1 bg-white/[0.03] border border-white/[0.06] rounded-lg p-1">
					<button on:click={() => viewMode = 'list'} aria-label="List view" class="p-1.5 rounded transition-colors {viewMode === 'list' ? 'bg-white/[0.08] text-white/80' : 'text-white/30 hover:text-white/60'}">
						<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="8" y1="6" x2="21" y2="6"/><line x1="8" y1="12" x2="21" y2="12"/><line x1="8" y1="18" x2="21" y2="18"/><line x1="3" y1="6" x2="3.01" y2="6"/><line x1="3" y1="12" x2="3.01" y2="12"/><line x1="3" y1="18" x2="3.01" y2="18"/></svg>
					</button>
					<button on:click={() => viewMode = 'grid'} aria-label="Grid view" class="p-1.5 rounded transition-colors {viewMode === 'grid' ? 'bg-white/[0.08] text-white/80' : 'text-white/30 hover:text-white/60'}">
						<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="3" width="7" height="7"/><rect x="14" y="3" width="7" height="7"/><rect x="14" y="14" width="7" height="7"/><rect x="3" y="14" width="7" height="7"/></svg>
					</button>
				</div>
			</div>

			<!-- Server List -->
			<div class="rounded-xl bg-white/[0.02] border border-white/[0.06] overflow-hidden">
				{#if viewMode === 'list'}
					<table class="w-full text-left">
						<thead>
							<tr class="border-b border-white/[0.04]">
								<th class="pb-2 pt-3 px-4 text-[11px] font-medium text-white/30 uppercase tracking-wider">Name</th>
								<th class="pb-2 pt-3 text-[11px] font-medium text-white/30 uppercase tracking-wider">Runtime</th>
								<th class="pb-2 pt-3 text-[11px] font-medium text-white/30 uppercase tracking-wider">Status</th>
								<th class="pb-2 pt-3 text-[11px] font-medium text-white/30 uppercase tracking-wider">Lifecycle</th>
								<th class="pb-2 pt-3 text-[11px] font-medium text-white/30 uppercase tracking-wider">Version</th>
								<th class="pb-2 pt-3 text-[11px] font-medium text-white/30 uppercase tracking-wider">Last Used</th>
								<th class="pb-2 pt-3 text-[11px] font-medium text-white/30 uppercase tracking-wider">Tools</th>
								<th class="pb-2 pt-3 text-[11px] font-medium text-white/30 uppercase tracking-wider text-right">Actions</th>
							</tr>
						</thead>
						<tbody class="text-xs">
							{#each filtered as server}
								<tr
									class="border-b border-white/[0.02] hover:bg-white/[0.02] transition-colors cursor-pointer {$selectedServerId === server.id ? 'bg-white/[0.03]' : ''}"
									on:click={() => selectServer(server.id)}
								>
									<td class="py-3 px-4">
										<div class="flex items-center gap-3">
											<div class="w-8 h-8 rounded-md bg-white/[0.04] border border-white/[0.06] flex items-center justify-center text-sm">
												{server.name.charAt(0)}
											</div>
											<div>
												<p class="text-white/80 font-medium">{server.name}</p>
												<p class="text-[10px] text-white/25 truncate max-w-[200px]">{server.description}</p>
												<span class="inline-block mt-0.5 px-1.5 py-0.5 rounded text-[9px] font-medium border {trustClass(server.trust)}">{server.trust}</span>
											</div>
										</div>
									</td>
									<td>
										<span class="px-2 py-0.5 rounded text-[10px] font-medium {runtimeClass(server.runtime)}">{server.runtime}</span>
									</td>
									<td>
										<div class="flex flex-col gap-0.5">
											<span class="flex items-center gap-1.5 {statusClass(server.status)}">
												<span class="w-1.5 h-1.5 rounded-full {statusDot(server.status)}"></span>
												{server.status.charAt(0).toUpperCase() + server.status.slice(1)}
											</span>
											{#if server.status_detail}
												<span class="text-[10px] text-white/20">{server.status_detail}</span>
											{/if}
										</div>
									</td>
									<td class="text-white/40">{server.lifecycle}</td>
									<td class="text-white/50">{server.version}</td>
									<td class="text-white/30">{formatTimeAgo(server.last_used_at)}</td>
									<td class="text-white/50">{server.tools_count}</td>
									<td class="text-right pr-4">
										<div class="flex items-center justify-end gap-1">
											<button
												on:click|stopPropagation={() => handleToggle(server.id)}
												class="p-1.5 rounded-md hover:bg-white/[0.06] text-white/30 hover:text-white/70 transition-colors"
												title={server.status === 'running' ? 'Stop' : 'Start'}
											>
												{#if server.status === 'running'}
													<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="6" y="4" width="4" height="16"/><rect x="14" y="4" width="4" height="16"/></svg>
												{:else}
													<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="5 3 19 12 5 21 5 3"/></svg>
												{/if}
											</button>
	
										</div>
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				{:else}
					<div class="grid grid-cols-2 lg:grid-cols-3 gap-4 p-4">
						{#each filtered as server}
							<div
								role="button"
								tabindex="0"
								on:keydown={() => selectServer(server.id)}
								class="rounded-lg bg-white/[0.02] border border-white/[0.06] p-4 hover:border-orange-500/20 hover:bg-white/[0.03] transition-all cursor-pointer {$selectedServerId === server.id ? 'border-orange-500/30 bg-white/[0.03]' : ''}"
								on:click={() => selectServer(server.id)}
							>
								<div class="flex items-start justify-between mb-3">
									<div class="flex items-center gap-2.5">
										<div class="w-9 h-9 rounded-md bg-white/[0.04] border border-white/[0.06] flex items-center justify-center text-sm">
											{server.name.charAt(0)}
										</div>
										<div>
											<p class="text-sm font-medium text-white/80">{server.name}</p>
											<span class="px-1.5 py-0.5 rounded text-[9px] font-medium border {trustClass(server.trust)}">{server.trust}</span>
										</div>
									</div>
									<span class="w-2 h-2 rounded-full {statusDot(server.status)}"></span>
								</div>
								<p class="text-[11px] text-white/25 mb-3 line-clamp-2">{server.description}</p>
								<div class="flex items-center justify-between">
									<span class="px-2 py-0.5 rounded text-[10px] font-medium {runtimeClass(server.runtime)}">{server.runtime}</span>
									<span class="text-[10px] text-white/30">{server.tools_count} tools</span>
								</div>
							</div>
						{/each}
					</div>
				{/if}
				{#if filtered.length === 0}
					<div class="flex flex-col items-center justify-center py-16 gap-3">
						<span class="text-2xl opacity-20">📦</span>
						<p class="text-sm text-white/40">No servers found</p>
						<button on:click={() => goto('/marketplace')} class="text-xs px-3 py-1.5 rounded-lg bg-orange-500 text-white hover:bg-orange-600 transition-colors">Browse Marketplace</button>
					</div>
				{/if}
			</div>

			<p class="text-[11px] text-white/20">Showing {filtered.length} of {servers.length} servers</p>
		</div>
	</div>

	<!-- Detail Panel -->
	{#if $selectedServerId && $serverDetail}
		<div class="w-80 border-l border-white/[0.06] bg-[#0a0a0f] flex flex-col fixed right-0 top-0 bottom-0 z-40">
			<!-- Detail Header -->
			<div class="p-5 border-b border-white/[0.06]">
				<div class="flex items-start justify-between mb-3">
					<div class="flex items-center gap-3">
						<div class="w-10 h-10 rounded-lg bg-white/[0.04] border border-white/[0.06] flex items-center justify-center text-lg">
							{$serverDetail.name.charAt(0)}
						</div>
						<div>
							<h3 class="text-sm font-bold text-white/90">{$serverDetail.name} MCP</h3>
							<p class="text-[11px] text-white/30">{$serverDetail.author} · {$serverDetail.runtime} · v{$serverDetail.version}</p>
						</div>
					</div>
					<div class="flex items-center gap-2">
						<span class="flex items-center gap-1.5 text-[11px] {statusClass($serverDetail.status)}">
							<span class="w-1.5 h-1.5 rounded-full {statusDot($serverDetail.status)}"></span>
							{$serverDetail.status.charAt(0).toUpperCase() + $serverDetail.status.slice(1)}
						</span>
						<button on:click={() => selectServer(null)} aria-label="Close" class="text-white/20 hover:text-white/60 transition-colors p-1">
							<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
						</button>
					</div>
				</div>
				<p class="text-xs text-white/40 mb-3">{$serverDetail.description}</p>
				<div class="flex items-center gap-2">
					<span class="px-2 py-0.5 rounded text-[10px] font-medium border {trustClass($serverDetail.trust)}">{$serverDetail.trust}</span>
					{#if $serverDetail.trust === 'anthropic-official'}
						<span class="px-2 py-0.5 rounded text-[10px] font-medium bg-blue-500/10 text-blue-400 border border-blue-500/20">anthropic-official</span>
					{/if}
				</div>
			</div>

			<!-- Detail Tabs -->
			<div class="flex items-center gap-1 border-b border-white/[0.06] px-2">
				{#each ['overview', 'tools', 'config', 'environment', 'logs'] as tab}
					<button
						on:click={() => detailTab = tab as typeof detailTab}
						class="px-3 py-2 text-[11px] transition-colors relative {detailTab === tab ? 'text-orange-400 font-medium' : 'text-white/30 hover:text-white/60'}"
					>
						{tab.charAt(0).toUpperCase() + tab.slice(1)}
						{#if detailTab === tab}
							<div class="absolute bottom-0 left-0 right-0 h-0.5 bg-orange-500 rounded-t-full"></div>
						{/if}
					</button>
				{/each}
			</div>

			<!-- Detail Content -->
			<div class="flex-1 overflow-y-auto p-5 space-y-5">
				{#if detailTab === 'overview'}
					<!-- Process Info -->
					{#if $serverDetail.process}
						<div class="grid grid-cols-2 gap-3">
							<div>
								<p class="text-[10px] text-white/30 uppercase tracking-wider mb-1">Status</p>
								<p class="text-xs text-white/70 flex items-center gap-1.5">
									<span class="w-1.5 h-1.5 rounded-full {statusDot($serverDetail.status)}"></span>
									{$serverDetail.status.charAt(0).toUpperCase() + $serverDetail.status.slice(1)}
								</p>
							</div>
							<div>
								<p class="text-[10px] text-white/30 uppercase tracking-wider mb-1">Memory</p>
								<p class="text-xs text-white/70">{$serverDetail.process.memory_mb} MB</p>
							</div>
							<div>
								<p class="text-[10px] text-white/30 uppercase tracking-wider mb-1">PID</p>
								<p class="text-xs text-white/70">{$serverDetail.process.pid ?? '—'}</p>
							</div>
							<div>
								<p class="text-[10px] text-white/30 uppercase tracking-wider mb-1">CPU</p>
								<p class="text-xs text-white/70">{$serverDetail.process.cpu_percent}%</p>
							</div>
							<div>
								<p class="text-[10px] text-white/30 uppercase tracking-wider mb-1">Uptime</p>
								<p class="text-xs text-white/70">{formatUptime($serverDetail.process.uptime_seconds)}</p>
							</div>
							<div>
								<p class="text-[10px] text-white/30 uppercase tracking-wider mb-1">Restarts</p>
								<p class="text-xs text-white/70">{$serverDetail.process.restarts}</p>
							</div>
						</div>
					{/if}

					<!-- Lifecycle -->
					<div class="rounded-lg bg-white/[0.02] border border-white/[0.06] p-4">
						<h4 class="text-xs font-medium text-white/70 mb-3">Lifecycle</h4>
						<div class="space-y-2">
							<div class="flex items-center justify-between">
								<span class="text-[11px] text-white/40">Mode</span>
								<span class="text-xs text-white/70">{$serverDetail.lifecycle}</span>
							</div>
							{#if $serverDetail.idle_timeout}
								<div class="flex items-center justify-between">
									<span class="text-[11px] text-white/40">Idle Timeout</span>
									<span class="text-xs text-white/70">{$serverDetail.idle_timeout}</span>
								</div>
							{/if}
						</div>
						<button on:click={() => toast.info('Lifecycle configuration coming soon')} class="mt-3 w-full py-1.5 rounded-md bg-white/[0.03] border border-white/[0.06] text-[11px] text-white/50 hover:text-white/80 hover:bg-white/[0.06] transition-colors">
							Edit Lifecycle
						</button>
					</div>

					<!-- Last Used -->
					<div class="rounded-lg bg-white/[0.02] border border-white/[0.06] p-4">
						<h4 class="text-xs font-medium text-white/70 mb-2">Last Used</h4>
						<p class="text-xs text-white/50">{formatTimeAgo($serverDetail.last_used_at)}</p>
						{#if $serverDetail.last_used_by}
							<div class="flex items-center gap-2 mt-2">
								<span class="w-5 h-5 rounded bg-white/[0.04] flex items-center justify-center text-[10px] text-white/30">VS</span>
								<span class="text-[11px] text-white/40">{$serverDetail.last_used_by}</span>
							</div>
						{/if}
					</div>

					<!-- Tools -->
					<div class="rounded-lg bg-white/[0.02] border border-white/[0.06] p-4">
						<div class="flex items-center justify-between mb-3">
							<h4 class="text-xs font-medium text-white/70">Tools</h4>
							<button on:click={() => detailTab = 'tools'} class="text-[11px] text-orange-400 hover:text-orange-300 transition-colors">View Tools</button>
						</div>
						<p class="text-xs text-white/50">{$serverDetail.tools_count} tools available</p>
					</div>

					<!-- Actions -->
					<div class="flex items-center gap-3">
						<button on:click={() => handleRestart($serverDetail.id)} disabled={serverLoading} class="flex-1 flex items-center justify-center gap-2 py-2 rounded-lg bg-white/[0.03] border border-white/[0.06] text-xs text-white/70 hover:text-white/90 hover:bg-white/[0.06] transition-colors disabled:opacity-50 disabled:cursor-not-allowed">
							<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="23 4 23 10 17 10"/><path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"/></svg>
							Restart
						</button>
						<button on:click={() => handleUninstall($serverDetail.id)} disabled={serverLoading} class="flex-1 flex items-center justify-center gap-2 py-2 rounded-lg bg-red-500/10 border border-red-500/20 text-xs text-red-400 hover:bg-red-500/20 transition-colors disabled:opacity-50 disabled:cursor-not-allowed">
							<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>
							Uninstall
						</button>
					</div>
				{:else if detailTab === 'tools'}
					{#if $serverTools.length === 0}
						<p class="text-xs text-white/20 text-center py-8">No tools available</p>
					{:else}
						<div class="space-y-2">
							{#each $serverTools as tool}
								<div class="rounded-lg bg-white/[0.02] border border-white/[0.06] p-3">
									<p class="text-xs font-medium text-white/70">{tool.name}</p>
									<p class="text-[11px] text-white/30 mt-0.5">{tool.description}</p>
								</div>
							{/each}
						</div>
					{/if}
				{:else if detailTab === 'config'}
					{#if $serverConfig}
						<div class="space-y-4">
							<div>
								<p class="text-[10px] text-white/30 uppercase tracking-wider mb-1">Command</p>
								<p class="text-xs text-white/70 font-mono bg-white/[0.02] rounded p-2">{$serverConfig.command}</p>
							</div>
							<div>
								<p class="text-[10px] text-white/30 uppercase tracking-wider mb-1">Args</p>
								<div class="text-xs text-white/70 font-mono bg-white/[0.02] rounded p-2 space-y-1">
									{#each $serverConfig.args as arg}
										<p>{arg}</p>
									{/each}
								</div>
							</div>
							<div>
								<p class="text-[10px] text-white/30 uppercase tracking-wider mb-1">Working Directory</p>
								<p class="text-xs text-white/70 font-mono bg-white/[0.02] rounded p-2">{$serverConfig.cwd || '—'}</p>
							</div>
						</div>
					{:else}
						<p class="text-xs text-white/20 text-center py-8">No config available</p>
					{/if}
				{:else if detailTab === 'environment'}
					{#if $serverConfig && $serverConfig.env.length > 0}
						<div class="space-y-2">
							{#each $serverConfig.env as env}
								<div class="rounded-lg bg-white/[0.02] border border-white/[0.06] p-3">
									<div class="flex items-center justify-between mb-1">
										<span class="text-xs font-medium text-white/60">{env.key}</span>
										{#if env.secret}
											<span class="px-1.5 py-0.5 rounded text-[9px] bg-amber-500/10 text-amber-400 border border-amber-500/20">Secret</span>
										{/if}
									</div>
									<p class="text-[11px] text-white/30 font-mono">{env.secret ? '••••••••' : env.value}</p>
								</div>
							{/each}
						</div>
					{:else}
						<p class="text-xs text-white/20 text-center py-8">No environment variables set</p>
					{/if}
				{:else if detailTab === 'logs'}
					{#if $serverLogs.length === 0}
						<p class="text-xs text-white/20 text-center py-8">No logs available</p>
					{:else}
						<div class="space-y-1 font-mono text-[10px]">
							{#each $serverLogs as log}
								<div class="flex items-start gap-2 py-1 border-b border-white/[0.02]">
									<span class="text-white/20 whitespace-nowrap">{new Date(log.timestamp).toLocaleTimeString()}</span>
									<span class="w-8 text-right {log.level === 'error' ? 'text-red-400' : log.level === 'warn' ? 'text-amber-400' : 'text-emerald-400/60'}">{log.level.toUpperCase()}</span>
									<span class="text-white/60">{log.message}</span>
								</div>
							{/each}
						</div>
					{/if}
				{/if}
			</div>
		</div>
	{/if}
</div>
