<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { goto } from '$app/navigation';
	import { toast } from '$lib/toast';
	import {
		routerStatus,
		systemUsage,
		activityLog,
		mcpServers,
		clients,
		installed,
		isConsoleExpanded,
		consoleTab,
		startDashboardSync,
		restartRouter,
		executeCommand,
		fetchInstalled,
		refreshClientConnections,
		toggleMcp,
		restartSingleServer
	} from '$lib/stores';

	let stopSync: (() => void) | null = null;
	let commandInput = '';
	let commandHistory: { command: string; output: string; error: string }[] = [];
	let commandLoading = false;
	let restartLoading = false;
	let query = '';

	$: filtered = $mcpServers.filter(s => !query || s.name.toLowerCase().includes(query.toLowerCase()));

	onMount(async () => {
		await fetchInstalled();
		await refreshClientConnections();
		stopSync = startDashboardSync();
	});

	onDestroy(() => {
		if (stopSync) stopSync();
	});

	function formatUptime(seconds: number): string {
		const h = Math.floor(seconds / 3600);
		const m = Math.floor((seconds % 3600) / 60);
		const s = seconds % 60;
		return `${h}h ${m}m ${s}s`;
	}

	function formatTimeAgo(timestamp: string): string {
		const diff = Date.now() - new Date(timestamp).getTime();
		const minutes = Math.floor(diff / 60000);
		const hours = Math.floor(diff / 3600000);
		const days = Math.floor(diff / 86400000);
		if (minutes < 1) return 'just now';
		if (minutes < 60) return `${minutes} minute${minutes !== 1 ? 's' : ''} ago`;
		if (hours < 24) return `${hours} hour${hours !== 1 ? 's' : ''} ago`;
		return `${days} day${days !== 1 ? 's' : ''} ago`;
	}

	function trustClass(trust: string): string {
		switch (trust) {
			case '1mcp-verified':
				return 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20';
			case 'internal':
				return 'bg-white/5 text-white/40 border-white/10';
			case 'community':
				return 'bg-amber-500/10 text-amber-400 border-amber-500/20';
			default:
				return 'bg-white/5 text-white/40 border-white/10';
		}
	}

	function runtimeClass(runtime: string): string {
		switch (runtime) {
			case 'node':
				return 'bg-emerald-500/10 text-emerald-400';
			case 'python':
				return 'bg-blue-500/10 text-blue-400';
			case 'binary':
				return 'bg-orange-500/10 text-orange-400';
			default:
				return 'bg-white/5 text-white/40';
		}
	}

	function activityIcon(type: string): string {
		switch (type) {
			case 'router_started':
				return '▶';
			case 'client_connected':
				return '🔗';
			case 'mcp_started':
				return '📦';
			case 'mcp_stopped':
				return '⏸';
			case 'user_registered':
				return '👤';
			case 'error':
				return '⚠';
			default:
				return '•';
		}
	}

	function activityBg(type: string): string {
		switch (type) {
			case 'router_started':
				return 'bg-emerald-500/10 text-emerald-400';
			case 'client_connected':
				return 'bg-blue-500/10 text-blue-400';
			case 'mcp_started':
				return 'bg-orange-500/10 text-orange-400';
			case 'mcp_stopped':
				return 'bg-white/5 text-white/40';
			case 'user_registered':
				return 'bg-violet-500/10 text-violet-400';
			case 'error':
				return 'bg-red-500/10 text-red-400';
			default:
				return 'bg-white/5 text-white/40';
		}
	}

	async function handleRestart() {
		restartLoading = true;
		try {
			await restartRouter();
		} catch (e) {
			console.error('Restart failed', e);
		}
		restartLoading = false;
	}

	async function handleServerRestart(id: string) {
		try {
			await restartSingleServer(id);
			toast.success('Server restarted');
		} catch (e: any) {
			toast.error('Restart failed: ' + e.message);
		}
	}

	function focusCommand() {
		(document.querySelector('input[placeholder="Type a command..."]') as HTMLInputElement)?.focus();
	}

	async function handleCommand(e: KeyboardEvent) {
		if (e.key !== 'Enter' || !commandInput.trim() || commandLoading) return;
		const cmd = commandInput.trim();
		commandInput = '';
		commandLoading = true;
		try {
			const result = await executeCommand(cmd);
			commandHistory = [...commandHistory, { command: cmd, output: result.output, error: result.error }];
		} catch (err: any) {
			commandHistory = [...commandHistory, { command: cmd, output: '', error: err?.message ?? 'Command failed' }];
		}
		commandLoading = false;
	}

	$: runningCount = $mcpServers.filter(s => s.status === 'running').length;
	$: sleepingCount = $mcpServers.filter(s => s.status === 'sleeping').length;
	$: totalCount = $mcpServers.length;
</script>

<div class="p-6 space-y-6">
	<!-- Top Bar -->
	<div class="flex items-center justify-between">
		<h1 class="text-lg font-bold text-white/90">Dashboard</h1>
		<div class="flex items-center gap-3">
			<button on:click={() => { consoleTab.set('output'); isConsoleExpanded.set(true); }} class="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-white/[0.03] border border-white/[0.06] text-xs text-white/60 hover:text-white/90 hover:bg-white/[0.06] transition-colors">
				<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/><polyline points="10 9 9 9 8 9"/></svg>
				Open Logs
			</button>
			<button on:click={() => window.open($routerStatus.metrics_endpoint, '_blank')} class="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-white/[0.03] border border-white/[0.06] text-xs text-white/60 hover:text-white/90 hover:bg-white/[0.06] transition-colors">
				<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M18 20V10"/><path d="M12 20V4"/><path d="M6 20v-6"/></svg>
				Metrics
			</button>
			<button on:click={handleRestart} disabled={restartLoading} class="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-orange-500 text-white text-xs font-medium hover:bg-orange-600 transition-colors disabled:opacity-50 disabled:cursor-not-allowed">
				<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><polyline points="23 4 23 10 17 10"/><path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"/></svg>
				Restart Router
			</button>
		</div>
	</div>

	<div class="grid grid-cols-12 gap-6">
		<!-- Main Column -->
		<div class="col-span-12 lg:col-span-9 space-y-6">
			<!-- Router Status Card -->
			<div class="rounded-xl bg-white/[0.02] border border-white/[0.06] p-5">
				<div class="flex items-start justify-between mb-4">
					<div>
						<div class="flex items-center gap-3 mb-1">
							<h2 class="text-base font-bold text-white/90">mach1 Router</h2>
							{#if $routerStatus.status === 'running'}
								<span class="px-2 py-0.5 rounded-full bg-emerald-500/10 text-emerald-400 text-[10px] font-medium border border-emerald-500/20">RUNNING</span>
							{:else}
								<span class="px-2 py-0.5 rounded-full bg-red-500/10 text-red-400 text-[10px] font-medium border border-red-500/20">STOPPED</span>
							{/if}
						</div>
						<p class="text-xs text-white/40">Single local router for all your MCP servers.</p>
					</div>
					<div class="text-right">
						<p class="text-[10px] text-white/30 uppercase tracking-wider mb-1">Connected Clients</p>
						<div class="flex items-center gap-2">
							{#each $clients.slice(0, 4) as client}
								<div class="flex items-center gap-1.5 px-2 py-1 rounded-md bg-white/[0.03] border border-white/[0.06]">
									<span class="text-[10px] text-white/60">{client.name}</span>
									{#if client.connected}
										<span class="w-1.5 h-1.5 rounded-full bg-emerald-500"></span>
									{:else}
										<span class="w-1.5 h-1.5 rounded-full bg-white/20"></span>
									{/if}
								</div>
							{/each}
						</div>
					</div>
				</div>
				<div class="flex items-center gap-6 text-xs">
					<div class="flex items-center gap-2">
						<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="text-white/20"><rect x="2" y="2" width="20" height="8" rx="2" ry="2"/><rect x="2" y="14" width="20" height="8" rx="2" ry="2"/><line x1="6" y1="6" x2="6.01" y2="6"/><line x1="6" y1="18" x2="6.01" y2="18"/></svg>
						<span class="text-white/40">Transport</span>
						<span class="text-white/70 font-medium">{$routerStatus.transport}</span>
					</div>
					<div class="flex items-center gap-2">
						<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="text-white/20"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>
						<span class="text-white/40">Uptime</span>
						<span class="text-white/70 font-medium">{formatUptime($routerStatus.uptime_seconds)}</span>
					</div>
					<div class="flex items-center gap-2">
						<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="text-white/20"><path d="M5 12h14"/><path d="M12 5v14"/></svg>
						<span class="text-white/40">Port (HTTP)</span>
						<span class="text-white/70 font-medium">{$routerStatus.port}</span>
					</div>
					<div class="flex items-center gap-2">
						<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="text-white/20"><path d="M18 20V10"/><path d="M12 20V4"/><path d="M6 20v-6"/></svg>
						<span class="text-white/40">Metrics</span>
						<span class="text-white/70 font-medium">{$routerStatus.metrics_endpoint}</span>
					</div>
				</div>
			</div>

			<!-- MCP Servers Table -->
			<div class="rounded-xl bg-white/[0.02] border border-white/[0.06] p-5">
				<div class="flex items-center justify-between mb-4">
					<div>
						<h2 class="text-sm font-bold text-white/90">MCP Servers</h2>
						<p class="text-xs text-white/30 mt-0.5">{totalCount} servers installed · {runningCount} running · {sleepingCount} sleeping</p>
					</div>
					<div class="flex items-center gap-2">
						<div class="relative">
							<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="absolute left-2.5 top-1/2 -translate-y-1/2 text-white/20"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>
							<input type="text" placeholder="Search servers..." bind:value={query} class="pl-8 pr-3 py-1.5 rounded-lg bg-white/[0.03] border border-white/[0.06] text-xs text-white/70 placeholder-white/20 focus:outline-none focus:border-orange-500/30 w-48" />
						</div>
						<button on:click={() => toast.warning('Filter options coming soon')} class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-white/[0.03] border border-white/[0.06] text-xs text-white/50 hover:text-white/80 transition-colors">
							<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="22 3 2 3 10 12.46 10 19 14 21 14 12.46 22 3"/></svg>
							Filters
						</button>
					</div>
				</div>
				<div class="min-w-0">
					<table class="w-full text-left">
						<thead>
							<tr class="border-b border-white/[0.04]">
								<th class="pb-2 text-[11px] font-medium text-white/30 uppercase tracking-wider">Name</th>
								<th class="pb-2 text-[11px] font-medium text-white/30 uppercase tracking-wider">Runtime</th>
								<th class="pb-2 text-[11px] font-medium text-white/30 uppercase tracking-wider">Version</th>
								<th class="pb-2 text-[11px] font-medium text-white/30 uppercase tracking-wider">Status</th>
								<th class="pb-2 text-[11px] font-medium text-white/30 uppercase tracking-wider">Lifecycle</th>
								<th class="pb-2 text-[11px] font-medium text-white/30 uppercase tracking-wider">Trust</th>
								<th class="pb-2 text-[11px] font-medium text-white/30 uppercase tracking-wider">Last Used</th>
								<th class="pb-2 text-[11px] font-medium text-white/30 uppercase tracking-wider text-right">Actions</th>
							</tr>
						</thead>
						<tbody class="text-xs">
							{#each filtered as server}
								<tr class="border-b border-white/[0.02] hover:bg-white/[0.01] transition-colors">
									<td class="py-3">
										<div class="flex items-center gap-2.5">
											<div class="w-7 h-7 rounded-md bg-white/[0.04] border border-white/[0.06] flex items-center justify-center text-xs">
												{server.name.charAt(0)}
											</div>
											<div>
												<p class="text-white/80 font-medium">{server.name}</p>
												{#if server.id === 'mach1'}
													<p class="text-[10px] text-white/25">Core</p>
												{/if}
											</div>
										</div>
									</td>
									<td>
										<span class="px-2 py-0.5 rounded text-[10px] font-medium {runtimeClass(server.runtime)}">
											{server.runtime}
										</span>
									</td>
									<td class="text-white/50">{server.version}</td>
									<td>
										{#if server.status === 'running'}
											<span class="flex items-center gap-1.5 text-emerald-400">
												<span class="w-1.5 h-1.5 rounded-full bg-emerald-500"></span>
												Running
											</span>
										{:else if server.status === 'sleeping'}
											<span class="flex items-center gap-1.5 text-white/30">
												<span class="w-1.5 h-1.5 rounded-full bg-white/20"></span>
												Sleeping
											</span>
										{:else}
											<span class="flex items-center gap-1.5 text-red-400">
												<span class="w-1.5 h-1.5 rounded-full bg-red-500"></span>
												Error
											</span>
										{/if}
									</td>
									<td class="text-white/40">{server.lifecycle}</td>
									<td>
										<span class="px-2 py-0.5 rounded text-[10px] font-medium border {trustClass(server.trust)}">
											{server.trust}
										</span>
									</td>
									<td class="text-white/30">{server.last_used_at ? formatTimeAgo(server.last_used_at) : '—'}</td>
									<td class="text-right">
										<div class="flex items-center justify-end gap-1">
											<button on:click={() => toggleMcp(server.id)} class="p-1.5 rounded-md hover:bg-white/[0.06] text-white/30 hover:text-white/70 transition-colors" title={server.status === 'running' ? 'Stop' : 'Start'}>
												{#if server.status === 'running'}
													<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="6" y="4" width="4" height="16"/><rect x="14" y="4" width="4" height="16"/></svg>
												{:else}
													<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="5 3 19 12 5 21 5 3"/></svg>
												{/if}
											</button>
											<button on:click|stopPropagation={() => handleServerRestart(server.id)} class="p-1.5 rounded-md hover:bg-white/[0.06] text-white/30 hover:text-white/70 transition-colors" title="Restart">
												<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="23 4 23 10 17 10"/><path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"/></svg>
											</button>
										</div>
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
				{#if $mcpServers.length === 0}
					<div class="flex flex-col items-center justify-center py-12 gap-3">
						<span class="text-2xl opacity-20">📦</span>
						<p class="text-sm text-white/40">No servers installed</p>
						<button on:click={() => goto('/discover')} class="text-xs px-3 py-1.5 rounded-lg bg-orange-500 text-white hover:bg-orange-600 transition-colors">Browse Marketplace</button>
					</div>
				{:else}
					<p class="text-[11px] text-white/20 mt-3">Showing {filtered.length} of {$mcpServers.length} servers.</p>
				{/if}
			</div>


		</div>

		<!-- Right Column -->
		<div class="col-span-12 lg:col-span-3 space-y-6">
			<!-- System Usage -->
			<div class="rounded-xl bg-white/[0.02] border border-white/[0.06] p-5">
				<h3 class="text-sm font-bold text-white/90 mb-4">System Usage</h3>
				<div class="space-y-4">
					<div>
						<div class="flex items-center justify-between mb-1.5">
							<span class="text-[11px] text-white/40">CPU Usage</span>
							<span class="text-xs font-medium text-emerald-400">{$systemUsage.cpu_percent}%</span>
						</div>
						<div class="h-1.5 bg-white/[0.04] rounded-full overflow-hidden">
							<div class="h-full bg-emerald-500 rounded-full transition-all" style="width: {$systemUsage.cpu_percent}%"></div>
						</div>
					</div>
					<div>
						<div class="flex items-center justify-between mb-1.5">
							<span class="text-[11px] text-white/40">Memory Usage</span>
							<span class="text-xs font-medium text-orange-400">{$systemUsage.memory_percent}%</span>
						</div>
						<div class="h-1.5 bg-white/[0.04] rounded-full overflow-hidden">
							<div class="h-full bg-orange-500 rounded-full transition-all" style="width: {$systemUsage.memory_percent}%"></div>
						</div>
					</div>
					<div>
						<div class="flex items-center justify-between mb-1.5">
							<span class="text-[11px] text-white/40">Disk Usage</span>
							<span class="text-xs font-medium text-blue-400">{$systemUsage.disk_percent}%</span>
						</div>
						<div class="h-1.5 bg-white/[0.04] rounded-full overflow-hidden">
							<div class="h-full bg-blue-500 rounded-full transition-all" style="width: {$systemUsage.disk_percent}%"></div>
						</div>
					</div>
				</div>
			</div>

			<!-- Client Connections -->
			<div class="rounded-xl bg-white/[0.02] border border-white/[0.06] p-5">
				<div class="flex items-center justify-between mb-4">
					<h3 class="text-sm font-bold text-white/90">Client Connections</h3>
					<button on:click={() => goto('/clients')} class="text-[11px] text-orange-400 hover:text-orange-300 transition-colors flex items-center gap-0.5">
						View all
						<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="9 18 15 12 9 6"/></svg>
					</button>
				</div>
				<div class="space-y-2">
					{#each $clients.slice(0, 5) as client}
						<div class="flex items-center justify-between py-1.5">
							<div class="flex items-center gap-2.5">
								<div class="w-6 h-6 rounded bg-white/[0.04] border border-white/[0.06] flex items-center justify-center text-[10px] text-white/40">
									{client.name.charAt(0)}
								</div>
								<span class="text-xs text-white/60">{client.name}</span>
							</div>
							{#if client.connected}
								<span class="text-[10px] text-emerald-400 flex items-center gap-1">
									Connected
									<span class="w-1.5 h-1.5 rounded-full bg-emerald-500"></span>
								</span>
							{:else}
								<span class="text-[10px] text-white/25 flex items-center gap-1">
									Not connected
									<span class="w-1.5 h-1.5 rounded-full bg-white/15"></span>
								</span>
							{/if}
						</div>
					{/each}
				</div>
			</div>

			<!-- Recent Activity -->
			<div class="rounded-xl bg-white/[0.02] border border-white/[0.06] p-5">
				<div class="flex items-center justify-between mb-4">
					<h3 class="text-sm font-bold text-white/90">Recent Activity</h3>
					<button on:click={() => goto('/activity')} class="text-[11px] text-orange-400 hover:text-orange-300 transition-colors flex items-center gap-0.5">
						View all
						<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="9 18 15 12 9 6"/></svg>
					</button>
				</div>
				<div class="space-y-3">
					{#each $activityLog as activity}
						<div class="flex items-start gap-2.5">
							<div class="w-6 h-6 rounded-full {activityBg(activity.type)} flex items-center justify-center text-[10px] flex-shrink-0 mt-0.5">
								{activityIcon(activity.type)}
							</div>
							<div>
								<p class="text-xs text-white/70">{activity.message}</p>
								<p class="text-[10px] text-white/25 mt-0.5">{formatTimeAgo(activity.timestamp)}</p>
							</div>
						</div>
					{/each}
					{#if $activityLog.length === 0}
						<p class="text-xs text-white/20 text-center py-4">No recent activity</p>
					{/if}
				</div>
			</div>
		</div>
	</div>
</div>
