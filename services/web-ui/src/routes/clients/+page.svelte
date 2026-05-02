<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { toast } from '$lib/toast';
	import {
		clients,
		routerStatus,
		selectedClientId,
		clientDetail,
		clientRoutingHealth,
		clientConfigPreview,
		refreshClientConnections,
		fetchRouterStatus,
		selectClient,
		connectClient,
		disconnectClient,
		connectAllSupportedClients,
		disconnectAllClients
	} from '$lib/stores';

	type Tab = 'all' | 'supported';
	type StatusFilter = 'all' | 'connected' | 'not_connected' | 'disconnected';

	let activeTab: Tab = 'all';
	let query = '';
	let statusFilter: StatusFilter = 'all';
	let actionLoading = false;

	onMount(() => {
		refreshClientConnections();
		fetchRouterStatus();
	});

	function formatUptime(seconds: number): string {
		const h = Math.floor(seconds / 3600);
		const m = Math.floor((seconds % 3600) / 60);
		const s = seconds % 60;
		return `${h}h ${m}m ${s}s`;
	}

	function clientStatus(client: typeof $clients[0]): { label: string; class: string; dot: string } {
		if (client.connected) {
			return { label: 'CONNECTED', class: 'text-emerald-400', dot: 'bg-emerald-500' };
		}
		if (client.id === 'claude' || client.id === 'claudecode' || client.id === 'codex') {
			return { label: 'MANUAL ONLY', class: 'text-amber-500', dot: 'bg-amber-500' };
		}
		return { label: 'DISCONNECTED', class: 'text-white/30', dot: 'bg-white/20' };
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

	function clientTransport(clientId: string): string {
		if (clientId === 'claude') return 'file';
		if (clientId === 'claudecode') return 'file';
		if (clientId === 'windsurf') return 'file';
		if (clientId === 'opencode') return 'file';
		if (clientId === 'antigravity') return 'file';
		if (clientId === 'continue') return 'file';
		return 'stdio';
	}

	// TODO: config paths should come from the API, not hardcoded here
	function clientConfigPath(clientId: string): string {
		const paths: Record<string, string> = {
			vscode: '~/.vscode/mcp.json',
			cursor: '~/.cursor/mcp.json',
			claude: '~/.claude_desktop_config.json',
			claudecode: '~/.claude.json',
			windsurf: '~/.codeium/mcp_config.json',
			codex: '~/.codex/config.toml',
			opencode: '~/.config/opencode/mcp.json',
			antigravity: '~/.antigravity/mcp.json',
			continue: '~/.continue/mcp.json'
		};
		return paths[clientId] ?? '~/.config/mcp.json';
	}

	function clientLastSeen(client: typeof $clients[0]): string {
		if (client.last_seen) {
			try { return formatTimeAgo(client.last_seen); } catch { return client.last_seen; }
		}
		if (client.connected) return 'just now';
		return '—';
	}

	function clientRouting(client: typeof $clients[0]): { label: string; detail: string; class: string } {
		if (!client.connected) {
			return { label: 'INACTIVE', detail: '', class: 'text-white/20' };
		}
		const health = $selectedClientId === client.id ? $clientRoutingHealth : null;
		if (health) {
			return {
				label: health.requests > 0 ? 'ACTIVE' : 'IDLE',
				detail: `${health.requests} req / ${health.period}`,
				class: 'text-emerald-400'
			};
		}
		if (client.routing_status === 'active') {
			return { label: 'ACTIVE', detail: client.routing_detail ?? '—', class: 'text-emerald-400' };
		}
		return { label: 'IDLE', detail: '0 req / 5m', class: 'text-emerald-400' };
	}

	$: filtered = (() => {
		let result = [...$clients];
		if (activeTab === 'supported') {
			result = result.filter((c) => ['vscode', 'cursor', 'claude', 'claudecode', 'windsurf', 'codex', 'opencode'].includes(c.id));
		}
		if (query.trim()) {
			const q = query.toLowerCase();
			result = result.filter((c) =>
				c.name.toLowerCase().includes(q) ||
				c.description.toLowerCase().includes(q)
			);
		}
		if (statusFilter === 'connected') {
			result = result.filter((c) => c.connected);
		} else if (statusFilter === 'not_connected') {
			result = result.filter((c) => !c.connected && ['claude', 'claudecode'].includes(c.id));
		} else if (statusFilter === 'disconnected') {
			result = result.filter((c) => !c.connected && !['claude', 'claudecode'].includes(c.id));
		}
		return result;
	})();

	$: connectedCount = $clients.filter((c) => c.connected).length;

	async function handleSetup(id: string) {
		actionLoading = true;
		try {
			await connectClient(id);
			selectClient(id);
		} catch (e) {
			console.error('Setup failed', e);
		}
		actionLoading = false;
	}

	async function handleDisconnect(id: string) {
		actionLoading = true;
		try {
			await disconnectClient(id);
			if ($selectedClientId === id) selectClient(null);
		} catch (e) {
			console.error('Disconnect failed', e);
		}
		actionLoading = false;
	}
</script>

<div class="flex h-full">
	<!-- Main Content -->
	<div class="flex-1 flex flex-col min-w-0" class:pr-80={$selectedClientId}>
		<div class="p-6 space-y-5">
			<!-- Header -->
			<div class="flex items-start justify-between">
				<div>
					<h1 class="text-xl font-bold text-white/95">Clients</h1>
					<p class="text-sm text-white/30 mt-1">Connect mach1 to replace all your MCP configs. {connectedCount} of {$clients.length} connected.</p>
				</div>
				<div class="flex items-center gap-3">
					<button on:click={() => { refreshClientConnections(); fetchRouterStatus(); }} class="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-white/[0.03] border border-white/[0.06] text-xs text-white/60 hover:text-white/90 hover:bg-white/[0.06] transition-colors">
						<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 12a9 9 0 0 0-9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/><path d="M3 3v5h5"/><path d="M3 12a9 9 0 0 0 9 9 9.75 9.75 0 0 0 6.74-2.74L21 16"/><path d="M16 16h5v5"/></svg>
						Refresh
					</button>
					<button on:click={connectAllSupportedClients} class="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-orange-500 text-white text-xs font-medium hover:bg-orange-600 transition-colors">
						<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/></svg>
						Connect All Supported
					</button>
				</div>
			</div>

			<!-- Router Status Card -->
			<div class="rounded-xl bg-white/[0.02] border border-white/[0.06] p-4 flex items-center gap-4">
				<div class="w-10 h-10 rounded-lg bg-orange-500/10 text-orange-400 flex items-center justify-center">
					<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="12 2 2 7 12 12 22 7 12 2"/><polyline points="2 17 12 22 22 17"/><polyline points="2 12 12 17 22 12"/></svg>
				</div>
				<div class="flex-1">
					<div class="flex items-center gap-2 mb-0.5">
						<h3 class="text-sm font-bold text-white/90">mach1 Router</h3>
						{#if $routerStatus.status === 'running'}
							<span class="px-2 py-0.5 rounded-full bg-emerald-500/10 text-emerald-400 text-[10px] font-medium border border-emerald-500/20">RUNNING</span>
						{:else}
							<span class="px-2 py-0.5 rounded-full bg-red-500/10 text-red-400 text-[10px] font-medium border border-red-500/20">STOPPED</span>
						{/if}
					</div>
					<div class="flex items-center gap-4 text-[11px] text-white/30">
						<span>Transport: <span class="text-white/50">{$routerStatus.transport}</span></span>
						<span>Uptime: <span class="text-white/50">{formatUptime($routerStatus.uptime_seconds)}</span></span>
						<span>Port: <span class="text-white/50">{$routerStatus.port}</span></span>
						<span>Metrics: <span class="text-white/50">{$routerStatus.metrics_endpoint}</span></span>
						<span class="ml-auto text-[9px] text-orange-400/60 leading-relaxed text-right max-w-[280px]">For best results customise clients to use rules/subagents for tool optimization — 1mcp does this for you. Verify if anything gone wrong.</span>
					</div>
				</div>
			</div>

			<!-- Tabs & Filters -->
			<div class="flex items-center justify-between">
				<div class="flex items-center gap-1 border-b border-white/[0.06]">
					<button on:click={() => activeTab = 'all'} class="px-4 py-2 text-sm transition-colors relative {activeTab === 'all' ? 'text-orange-400 font-medium' : 'text-white/30 hover:text-white/60'}">
						All Clients
						{#if activeTab === 'all'}
							<div class="absolute bottom-0 left-0 right-0 h-0.5 bg-orange-500 rounded-t-full"></div>
						{/if}
					</button>
					<button on:click={() => activeTab = 'supported'} class="px-4 py-2 text-sm transition-colors relative {activeTab === 'supported' ? 'text-orange-400 font-medium' : 'text-white/30 hover:text-white/60'}">
						Supported Clients
						{#if activeTab === 'supported'}
							<div class="absolute bottom-0 left-0 right-0 h-0.5 bg-orange-500 rounded-t-full"></div>
						{/if}
					</button>
				</div>
				<div class="flex items-center gap-2">
					<div class="relative">
						<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="absolute left-3 top-1/2 -translate-y-1/2 text-white/25"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>
						<input bind:value={query} placeholder="Search clients..." class="pl-9 pr-3 py-1.5 rounded-lg bg-white/[0.03] border border-white/[0.06] text-xs text-white/80 placeholder-white/20 focus:outline-none focus:border-orange-500/30 w-48" />
					</div>
					<select bind:value={statusFilter} class="bg-white/[0.03] border border-white/[0.06] rounded-lg px-3 py-1.5 text-xs text-white/60 focus:outline-none focus:border-orange-500/30">
						<option value="all">All Status</option>
						<option value="connected">Connected</option>
						<option value="not_connected">Not Connected</option>
						<option value="disconnected">Disconnected</option>
					</select>
				</div>
			</div>

			<!-- Scrolling banner for unsupported clients -->
			{#if ['claude', 'claudecode', 'codex'].includes($selectedClientId ?? '')}
				<div class="overflow-hidden rounded-xl bg-amber-900/10 border border-amber-600/20">
					<div class="animate-marquee whitespace-nowrap py-3 text-xs text-amber-400/80 font-medium">
						⚠️ &nbsp; Claude Desktop, Claude Code, and Codex do not support auto-setup. &nbsp; Please copy the mach1 config and connect manually. &nbsp; See docs: &nbsp;
						<a href="https://github.com/SaiAvinashPatoju/1mcp.in" target="_blank" rel="noopener noreferrer" class="underline hover:text-amber-300">github.com/SaiAvinashPatoju/1mcp.in</a>
						&nbsp; ⚠️ &nbsp; • &nbsp;
					</div>
				</div>
			{/if}

			<!-- Table -->
			<div class="rounded-xl bg-white/[0.02] border border-white/[0.06] overflow-hidden">
				<table class="w-full text-left">
					<thead>
						<tr class="border-b border-white/[0.04]">
							<th class="pb-2 pt-3 px-4 text-[11px] font-medium text-white/30 uppercase tracking-wider">Client</th>
							<th class="pb-2 pt-3 text-[11px] font-medium text-white/30 uppercase tracking-wider">Status</th>
							<th class="pb-2 pt-3 text-[11px] font-medium text-white/30 uppercase tracking-wider">Transport</th>
							<th class="pb-2 pt-3 text-[11px] font-medium text-white/30 uppercase tracking-wider">Config Path</th>
							<th class="pb-2 pt-3 text-[11px] font-medium text-white/30 uppercase tracking-wider">Last Seen</th>
							<th class="pb-2 pt-3 text-[11px] font-medium text-white/30 uppercase tracking-wider">Routing</th>
							<th class="pb-2 pt-3 text-[11px] font-medium text-white/30 uppercase tracking-wider text-right">Actions</th>
						</tr>
					</thead>
					<tbody class="text-xs">
						{#each filtered as client}
							{@const status = clientStatus(client)}
							{@const routing = clientRouting(client)}
							<tr
								class="border-b border-white/[0.02] hover:bg-white/[0.02] transition-colors cursor-pointer {$selectedClientId === client.id ? 'bg-white/[0.03]' : ''}"
								on:click={() => selectClient(client.id)}
							>
								<td class="py-3 px-4">
									<div class="flex items-center gap-3">
										<div class="w-8 h-8 rounded-md bg-white/[0.04] border border-white/[0.06] flex items-center justify-center text-lg">
											{client.icon}
										</div>
										<div>
											<p class="text-white/80 font-medium">{client.name}</p>
											<p class="text-[10px] text-white/25">{client.description}</p>
										</div>
									</div>
								</td>
								<td>
									<div class="flex flex-col gap-0.5">
										<span class="flex items-center gap-1.5 {status.class}">
											<span class="w-1.5 h-1.5 rounded-full {status.dot}"></span>
											{status.label}
										</span>
									</div>
								</td>
								<td class="text-white/40">{clientTransport(client.id)}</td>
								<td class="text-white/40">
									<div class="flex items-center gap-1.5">
										<span class="font-mono text-[10px]">{clientConfigPath(client.id)}</span>
										<button on:click|stopPropagation={() => { navigator.clipboard.writeText(clientConfigPath(client.id)); toast.success('Path copied'); }} class="text-white/20 hover:text-white/50 transition-colors" title="Copy path">
											<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>
										</button>
									</div>
								</td>
								<td class="text-white/30">{clientLastSeen(client)}</td>
								<td>
									<div class="flex flex-col gap-0.5">
										{#if client.connected}
											<span class="flex items-center gap-1 {routing.class}">
												<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/></svg>
												{routing.label}
											</span>
											<span class="text-[10px] text-white/20">{routing.detail}</span>
										{:else}
											<span class="flex items-center gap-1 text-white/20">
												<span class="w-1.5 h-1.5 rounded-full bg-white/15"></span>
												INACTIVE
											</span>
										{/if}
									</div>
								</td>
								<td class="text-right pr-4">
									<div class="flex items-center justify-end gap-1">
										{#if client.connected}
											<button
												on:click|stopPropagation={() => handleDisconnect(client.id)}
												disabled={actionLoading}
												class="px-3 py-1.5 rounded-md bg-white/[0.03] border border-white/[0.06] text-[11px] text-white/50 hover:text-white/80 hover:bg-white/[0.06] transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
												title="Restores your original MCP config"
											>
												Disconnect
											</button>
										{:else if client.id === 'claude' || client.id === 'claudecode' || client.id === 'codex'}
											<a
												href="https://github.com/SaiAvinashPatoju/1mcp.in"
												target="_blank"
												rel="noopener noreferrer"
												class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-md bg-white/[0.03] border border-white/[0.06] text-[11px] text-amber-400/70 hover:text-amber-300 hover:bg-white/[0.06] transition-colors"
												title="View docs for manual setup"
											>
												<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/></svg>
												Docs
											</a>
										{:else}
											<button
												on:click|stopPropagation={() => handleSetup(client.id)}
												disabled={actionLoading}
												class="px-3 py-1.5 rounded-md bg-orange-500 text-[11px] text-white font-medium hover:bg-orange-600 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
												title="Replaces all MCPs with mach1 (backs up existing)"
											>
												Connect via mach1
											</button>
										{/if}
									</div>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
				{#if filtered.length === 0}
					<div class="flex flex-col items-center justify-center py-16 gap-3">
						<span class="text-2xl opacity-20">🔌</span>
						<p class="text-sm text-white/40">No clients found</p>
					</div>
				{/if}
			</div>

			<p class="text-[11px] text-white/20">Showing {filtered.length} of {$clients.length} clients</p>
		</div>
	</div>

	<!-- Detail Panel -->
	{#if $selectedClientId && $clientDetail}
		<div class="w-80 border-l border-white/[0.06] bg-[#0a0a0f] flex flex-col fixed right-0 top-0 bottom-0 z-40 overflow-y-auto">
			<div class="p-5 space-y-5">
				<!-- Header -->
				<div class="flex items-start justify-between">
					<div class="flex items-center gap-3">
						<div class="w-10 h-10 rounded-lg bg-white/[0.04] border border-white/[0.06] flex items-center justify-center text-lg">
							{$clients.find(c => c.id === $selectedClientId)?.icon ?? ''}
						</div>
						<div>
							<div class="flex items-center gap-2">
								<h3 class="text-sm font-bold text-white/90">{$clientDetail.name}</h3>
								{#if $clientDetail.status === 'connected'}
									<span class="px-2 py-0.5 rounded-full bg-emerald-500/10 text-emerald-400 text-[10px] font-medium border border-emerald-500/20">CONNECTED</span>
								{:else if $clientDetail.status === 'connected_idle'}
									<span class="px-2 py-0.5 rounded-full bg-emerald-500/10 text-emerald-400 text-[10px] font-medium border border-emerald-500/20">CONNECTED</span>
								{:else}
									<span class="px-2 py-0.5 rounded-full bg-orange-500/10 text-orange-400 text-[10px] font-medium border border-orange-500/20">NOT CONNECTED</span>
								{/if}
							</div>
							<p class="text-[11px] text-white/30">{$clientDetail.subtitle}</p>
						</div>
					</div>
					<button on:click={() => selectClient(null)} class="text-white/20 hover:text-white/60 transition-colors p-1" aria-label="Close">
						<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
					</button>
				</div>

				<!-- Connection -->
				<div class="rounded-lg bg-white/[0.02] border border-white/[0.06] p-4 space-y-2.5">
					<h4 class="text-xs font-medium text-white/70 mb-1">Connection</h4>
					<div class="flex items-center justify-between">
						<span class="text-[11px] text-white/40">Transport</span>
						<span class="text-xs text-white/70">{$clientDetail.transport}</span>
					</div>
					<div class="flex items-center justify-between">
						<span class="text-[11px] text-white/40">Config Path</span>
						<div class="flex items-center gap-1">
							<span class="text-[10px] text-white/50 font-mono">{$clientDetail.config_path}</span>
							<button on:click={() => { navigator.clipboard.writeText($clientDetail.config_path); toast.success('Path copied'); }} class="text-white/20 hover:text-white/50 transition-colors" aria-label="Copy config path">
								<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>
							</button>
						</div>
					</div>
					<div class="flex items-center justify-between">
						<span class="text-[11px] text-white/40">Last Handshake</span>
						<span class="text-xs text-white/70">{$clientDetail.last_handshake}</span>
					</div>
					<div class="flex items-center justify-between">
						<span class="text-[11px] text-white/40">Router Binding</span>
						<span class="text-xs text-white/70">{$clientDetail.router_binding}</span>
					</div>
					<div class="flex items-center justify-between">
						<span class="text-[11px] text-white/40">Process ID</span>
						<span class="text-xs text-white/70 font-mono">{$clientDetail.process_id}</span>
					</div>
				</div>

				<!-- Routing Health -->
				{#if $clientRoutingHealth}
					<div class="rounded-lg bg-white/[0.02] border border-white/[0.06] p-4 space-y-2.5">
						<div class="flex items-center justify-between mb-1">
							<h4 class="text-xs font-medium text-white/70">Routing Health</h4>
							<span class="text-[10px] text-white/20">{$clientRoutingHealth.period}</span>
						</div>
						<div class="flex items-center justify-between">
							<span class="text-[11px] text-white/40">Requests</span>
							<span class="text-xs text-white/70">{$clientRoutingHealth.requests}</span>
						</div>
						<div class="flex items-center justify-between">
							<span class="text-[11px] text-white/40">Active Tools</span>
							<span class="text-xs text-white/70">{$clientRoutingHealth.active_tools.join(', ')}</span>
						</div>
						<div class="flex items-center justify-between">
							<span class="text-[11px] text-white/40">Latency (avg)</span>
							<span class="text-xs text-white/70">{$clientRoutingHealth.latency_avg_ms}ms</span>
						</div>
						<div class="flex items-center justify-between">
							<span class="text-[11px] text-white/40">Errors</span>
							<span class="text-xs text-white/70">{$clientRoutingHealth.errors}</span>
						</div>
					</div>
				{/if}

				<!-- Setup & Config -->
				<div class="rounded-lg bg-white/[0.02] border border-white/[0.06] p-4 space-y-3">
					<h4 class="text-xs font-medium text-white/70">Setup & Config</h4>
					{#if ['claude', 'claudecode', 'codex'].includes($selectedClientId ?? '')}
						<div class="rounded-md bg-amber-900/10 border border-amber-600/20 p-3 space-y-1.5">
							<div class="flex items-center gap-1.5 text-[11px] text-amber-400/80">
								<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="12" y1="16" x2="12" y2="12"/><line x1="12" y1="8" x2="12.01" y2="8"/></svg>
								<span class="font-medium">Manual setup required</span>
							</div>
							<p class="text-[10px] text-white/30 leading-relaxed">
								Auto-setup is not supported for this client. Please copy the mach1 MCP config and add it to your client's MCP configuration manually. See <a href="https://github.com/SaiAvinashPatoju/1mcp.in" target="_blank" rel="noopener noreferrer" class="underline hover:text-amber-300">docs</a>.
							</p>
						</div>
					{:else}
						<div class="rounded-md bg-orange-500/5 border border-orange-500/10 p-3 space-y-1.5">
							<div class="flex items-center gap-1.5 text-[11px] text-orange-400/80">
								<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg>
								<span class="font-medium">mach1 is in control</span>
							</div>
							<p class="text-[10px] text-white/30 leading-relaxed">
								All MCPs have been replaced with mach1. Your original MCP config was backed up and will be restored when you disconnect.
							</p>
						</div>
					{/if}
					<button on:click={() => toast.info('Config editing available in desktop app')} class="w-full flex items-center justify-between py-2 px-3 rounded-md bg-white/[0.03] border border-white/[0.06] text-xs text-white/60 hover:text-white/90 hover:bg-white/[0.06] transition-colors">
						<span>Open Config File</span>
						<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/><polyline points="15 3 21 3 21 9"/><line x1="10" y1="14" x2="21" y2="3"/></svg>
					</button>
					{#if $clientConfigPreview}
						<div class="rounded-md bg-[#050508] border border-white/[0.06] p-3 font-mono text-[10px] text-white/50 overflow-x-auto">
							<pre class="whitespace-pre">{$clientConfigPreview.content}</pre>
						</div>
					{/if}
				</div>

				<!-- Actions -->
				<div class="flex items-center gap-2">
					<button on:click={() => goto('/dashboard')} class="flex-1 flex items-center justify-center gap-2 py-2 rounded-lg bg-white/[0.03] border border-white/[0.06] text-xs text-white/60 hover:text-white/90 hover:bg-white/[0.06] transition-colors">
						<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>
						View Logs
					</button>
					{#if $clientDetail.status === 'connected' || $clientDetail.status === 'connected_idle'}
						<button on:click={() => handleDisconnect($clientDetail.id)} disabled={actionLoading} class="flex-1 flex items-center justify-center gap-2 py-2 rounded-lg bg-red-500/10 border border-red-500/20 text-xs text-red-400 hover:bg-red-500/20 transition-colors disabled:opacity-50 disabled:cursor-not-allowed">
							<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="1" y="4" width="22" height="16" rx="2" ry="2"/><line x1="1" y1="10" x2="23" y2="10"/></svg>
							Disconnect
						</button>
					{:else if ['claude', 'claudecode', 'codex'].includes($clientDetail.id)}
						<div class="flex-1 flex items-center justify-center gap-2 py-2 rounded-lg bg-amber-900/10 border border-amber-600/20 text-xs text-amber-400/80">
							<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="12" y1="16" x2="12" y2="12"/><line x1="12" y1="8" x2="12.01" y2="8"/></svg>
							Manual Setup Only
						</div>
					{:else}
						<button on:click={() => handleSetup($clientDetail.id)} disabled={actionLoading} class="flex-1 flex items-center justify-center gap-2 py-2 rounded-lg bg-orange-500 text-white text-xs font-medium hover:bg-orange-600 transition-colors disabled:opacity-50 disabled:cursor-not-allowed">
							<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/></svg>
							Connect
						</button>
					{/if}
				</div>
			</div>
		</div>
	{/if}
</div>

<style>
	@keyframes marquee {
		0% { transform: translateX(100%); }
		100% { transform: translateX(-100%); }
	}
	.animate-marquee {
		display: inline-block;
		animation: marquee 20s linear infinite;
	}
</style>
