<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { toast } from '$lib/toast';
	import McpConfigModal from '$lib/components/McpConfigModal.svelte';
	import {
		installed,
		marketplace,
		mcpServers,
		bundles,
		selectedServerId,
		serverDetail,
		serverTools,
		serverLogs,
		serverConfig,
		fetchInstalled,
		fetchMcpServers,
		fetchMarketplace,
		selectServer,
		scanServer,
		restartSingleServer,
		uninstallSingleServer,
		startDashboardSync,
		startMCP,
		stopMCP,
		installMCP,
		setMcpEnv,
		healthCheck,
		autoDetectEnv,
		fetchServerConfig,
		fetchBundles,
		installBundle,
		uninstallBundle
	} from '$lib/stores';
	import type { McpHealthResult, ServerConfig } from '$lib/types';

	type Tab = 'installed' | 'running' | 'bundles' | 'disabled';
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

	// v0.3.4 — Config modal state
	let showConfigModal = false;
	let configMcpId: string | null = null;
	let apiKeyInput: Record<string, string> = {};
	let configMcpName = '';
	let configMcpDescription = '';
	let configMcpConfig: ServerConfig | null = null;
	let configMcpHealth: McpHealthResult | null = null;
	let configLoading = false;

	onMount(() => {
		(async () => {
			await fetchInstalled();
			await fetchMcpServers();
			await fetchMarketplace();
			await fetchBundles();
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

	function requiredEnvKeys(mcp: { patProvider?: string | null }): string[] {
		if (!mcp.patProvider) return [];
		return [`${mcp.patProvider.toUpperCase()}_TOKEN`];
	}

	function getMcpEnvValues(mcpId: string): Record<string, string> {
		const inst = $installed.find((m) => m.id === mcpId);
		return inst?.env ?? {};
	}

	function computeHealth(
		installedMcp: typeof $installed[number] | undefined,
		server: typeof $mcpServers[number] | undefined
	): { status: string; label: string; detail: string | null; dotClass: string; textClass: string } {
		if (!installedMcp) {
			return { status: 'not_installed', label: 'Not installed', detail: null, dotClass: 'bg-white/20', textClass: 'text-white/30' };
		}
		if (!installedMcp.enabled) {
			return { status: 'disabled', label: 'Disabled', detail: null, dotClass: 'bg-white/30', textClass: 'text-white/40' };
		}
		if (server?.status === 'error') {
			return { status: 'error', label: 'Error', detail: server.status_detail ?? 'Process failed', dotClass: 'bg-red-500', textClass: 'text-red-400' };
		}
		const missing = requiredEnvKeys(installedMcp).filter((k) => !installedMcp.env?.[k]);
		if (missing.length > 0) {
			return { status: 'missing_env', label: 'Missing env vars', detail: `Needs ${missing.join(', ')}`, dotClass: 'bg-amber-400', textClass: 'text-amber-400' };
		}
		if (server?.status === 'running') {
			return { status: 'healthy', label: 'Running + Healthy', detail: server.status_detail ?? null, dotClass: 'bg-emerald-500', textClass: 'text-emerald-400' };
		}
		return { status: 'disabled', label: 'Disabled', detail: null, dotClass: 'bg-white/30', textClass: 'text-white/40' };
	}

	// Build unified server list from marketplace + installed + mcpServers
	$: servers = (() => {
		const byId = new Map<string, any>();

		// Seed from marketplace (includes not-installed)
		for (const mkt of $marketplace) {
			byId.set(mkt.id, {
				id: mkt.id,
				name: mkt.name,
				description: mkt.shortDescription,
				version: mkt.version,
				runtime: mkt.runtime,
				trust: mkt.verificationStatus,
				author: mkt.author,
				tools_count: 0,
				installed: mkt.installed,
				enabled: false,
				patProvider: mkt.patProvider,
				status_raw: null,
				status_detail: null,
				last_used_at: null,
				process: undefined,
			});
		}

		// Override with installed data
		for (const inst of $installed) {
			const existing = byId.get(inst.id);
			const server = $mcpServers.find((s) => s.id === inst.id);
			const health = computeHealth(inst, server);

			byId.set(inst.id, {
				...(existing ?? {}),
				id: inst.id,
				name: inst.name,
				description: inst.description,
				version: inst.version,
				runtime: inst.runtime,
				installed: true,
				enabled: inst.enabled,
				patProvider: inst.patProvider,
				status_raw: health.status,
				status_label: health.label,
				status_detail: health.detail,
				status_dot: health.dotClass,
				status_text: health.textClass,
				trust: existing?.trust ?? 'community',
				author: existing?.author ?? 'unknown',
				tools_count: server?.tools_count ?? existing?.tools_count ?? 0,
				last_used_at: server?.last_used_at ?? null,
				process: server?.process,
			});
		}

		return Array.from(byId.values());
	})();

	$: filtered = (() => {
		let result = [...servers];
		if (activeTab === 'installed') result = result.filter((s) => s.installed);
		if (activeTab === 'running') result = result.filter((s) => s.status_raw === 'healthy');
		if (activeTab === 'disabled') result = result.filter((s) => s.status_raw === 'disabled' || s.status_raw === 'not_installed');
		if (query.trim()) {
			const q = query.toLowerCase();
			result = result.filter((s) =>
				s.name.toLowerCase().includes(q) ||
				s.description.toLowerCase().includes(q) ||
				s.id.toLowerCase().includes(q)
			);
		}
		if (runtimeFilter !== 'all') result = result.filter((s) => s.runtime === runtimeFilter);
		if (statusFilter !== 'all') {
			result = result.filter((s) => {
				if (statusFilter === 'running') return s.status_raw === 'healthy';
				if (statusFilter === 'sleeping') return s.status_raw === 'disabled';
				if (statusFilter === 'error') return s.status_raw === 'error';
				return true;
			});
		}
		result.sort((a, b) => {
			if (sort === 'name') return a.name.localeCompare(b.name);
			if (sort === 'status') return (a.status_raw ?? '').localeCompare(b.status_raw ?? '');
			return 0;
		});
		return result;
	})();

	$: installedCount = servers.filter((s) => s.installed).length;
	$: runningCount = servers.filter((s) => s.status_raw === 'healthy').length;
	$: sleepingCount = servers.filter((s) => s.status_raw === 'disabled').length;
	$: errorCount = servers.filter((s) => s.status_raw === 'error').length;
	$: missingEnvCount = servers.filter((s) => s.status_raw === 'missing_env').length;

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

	async function handleStart(id: string) {
		try {
			await startMCP(id);
			toast.success('MCP started');
			await fetchMcpServers();
		} catch (e) {
			toast.error('Failed to start MCP');
		}
	}

	async function handleStop(id: string) {
		try {
			await stopMCP(id);
			toast.success('MCP stopped');
			await fetchMcpServers();
		} catch (e) {
			toast.error('Failed to stop MCP');
		}
	}

	async function handleInstall(id: string) {
		try {
			await installMCP(id);
			toast.success('MCP installed');
			await fetchInstalled();
			await fetchMcpServers();
		} catch (e) {
			toast.error('Failed to install MCP');
		}
	}

	async function openConfigModal(id: string) {
		const mcp = servers.find((s) => s.id === id);
		if (!mcp) return;
		configMcpId = id;
		configMcpName = mcp.name;
		configMcpDescription = mcp.description;
		configMcpHealth = null;
		configLoading = true;
		showConfigModal = true;

		try {
			await fetchServerConfig(id);
			configMcpConfig = $serverConfig;
			// Also try a quick health check
			try {
				const h = await healthCheck(id);
				configMcpHealth = h as McpHealthResult;
			} catch {
				configMcpHealth = null;
			}
		} catch {
			configMcpConfig = null;
		}
		configLoading = false;
	}

	function closeConfigModal() {
		showConfigModal = false;
		configMcpId = null;
		configMcpConfig = null;
		configMcpHealth = null;
	}

	async function handleSaveAndStart(vars: Record<string, string>) {
		if (!configMcpId) return;
		try {
			await setMcpEnv(configMcpId, vars);
			await startMCP(configMcpId);
			toast.success('Saved and started MCP');
			closeConfigModal();
			await fetchInstalled();
			await fetchMcpServers();
		} catch (e) {
			toast.error('Failed to save or start');
		}
	}

	async function handleSaveOnly(vars: Record<string, string>) {
		if (!configMcpId) return;
		try {
			await setMcpEnv(configMcpId, vars);
			toast.success('Environment saved');
			closeConfigModal();
			await fetchInstalled();
		} catch (e) {
			toast.error('Failed to save environment');
		}
	}

	async function handleTestConnection(): Promise<McpHealthResult> {
		if (!configMcpId) throw new Error('No MCP selected');
		const h = await healthCheck(configMcpId);
		return h as McpHealthResult;
	}

	async function handleAutoDetect(): Promise<Record<string, string>> {
		if (!configMcpId) return {};
		return autoDetectEnv(configMcpId);
	}
</script>

<div class="flex h-full">
	<!-- Main Content -->
	<div class="flex-1 flex flex-col min-w-0" class:pr-96={$selectedServerId}>
		<div class="p-6 space-y-5">
			<!-- Header -->
			<div class="flex items-start justify-between">
				<div>
					<h1 class="text-xl font-bold text-white/95">Servers</h1>
					<p class="text-sm text-white/30 mt-1">
						Manage installed MCP servers.
						<span class="text-white/50">{installedCount} installed</span>,
						<span class="text-emerald-400">{runningCount} running</span>,
						<span class="text-amber-400">{missingEnvCount} missing env</span>,
						<span class="text-red-400">{errorCount} error</span>.
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
				{#each ['installed', 'running', 'bundles', 'disabled'] as tab}
					<button
						on:click={() => activeTab = tab as Tab}
						class="px-4 py-2 text-sm transition-colors relative {activeTab === tab ? 'text-orange-400 font-medium' : 'text-white/30 hover:text-white/60'}"
					>
						{tab.charAt(0).toUpperCase() + tab.slice(1)}
						{#if tab === 'running'}
							<span class="ml-1 text-xs text-white/40">({runningCount})</span>
						{:else if tab === 'disabled'}
							<span class="ml-1 text-xs text-white/40">({sleepingCount + servers.filter(s => s.status_raw === 'not_installed').length})</span>
						{:else if tab === 'bundles'}
							<span class="ml-1 text-xs text-white/40">({$bundles.length})</span>
						{/if}
						{#if activeTab === tab}
							<div class="absolute bottom-0 left-0 right-0 h-0.5 bg-orange-500 rounded-t-full"></div>
						{/if}
					</button>
				{/each}
			</div>

			{#if activeTab === 'bundles'}
				<div class="flex items-center justify-between mb-4">
					<div>
						<p class="text-sm text-white/80 font-medium">Available Bundles</p>
						<p class="text-xs text-white/30">Collections of MCPs that work together</p>
					</div>
				</div>
				<div class="rounded-xl bg-white/[0.02] border border-white/[0.06] overflow-hidden">
					{#if $bundles.length === 0}
						<div class="flex flex-col items-center justify-center py-16 gap-3">
							<p class="text-sm font-medium text-white/80">No bundles found</p>
							<p class="text-xs text-white/30">Bundles are collections of MCPs that work together.</p>
						</div>
					{:else}
						<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4 p-4">
							{#each $bundles as bundle (bundle.id)}
								<div class="rounded-xl border border-white/[0.06] bg-white/[0.03] p-5 flex flex-col gap-3 hover:border-white/[0.12] transition-all">
									<div class="flex items-center gap-2">
										<span class="text-lg">📦</span>
										<h3 class="text-sm font-semibold text-white/90">{bundle.name}</h3>
									</div>
									<p class="text-xs text-white/40 leading-relaxed flex-1">{bundle.description}</p>
									<div class="text-[10px] text-white/30">v{bundle.version}</div>
									<div class="flex flex-wrap gap-1">
										{#each bundle.mcp_ids as mcpId}
											<span class="text-[10px] px-2 py-0.5 rounded bg-white/[0.04] text-white/40 border border-white/[0.06]">{mcpId}</span>
										{/each}
									</div>
									{#if bundle.installed}
										<button on:click={() => uninstallBundle(bundle.id)} disabled={serverLoading} class="w-full text-xs py-1.5 px-3 rounded-lg border border-white/[0.06] text-white/40 hover:text-red-400 hover:border-red-900/60 hover:bg-red-900/10 transition-colors disabled:opacity-50 disabled:cursor-not-allowed">Uninstall</button>
									{:else}
										<button on:click={() => installBundle(bundle.id)} disabled={serverLoading} class="w-full text-xs py-1.5 px-3 rounded-lg bg-orange-500 text-white hover:bg-orange-600 transition-colors font-medium disabled:opacity-50 disabled:cursor-not-allowed">Install Bundle</button>
									{/if}
								</div>
							{/each}
						</div>
					{/if}
				</div>
			{:else}
				<!-- Stats Cards -->
				<div class="grid grid-cols-5 gap-4">
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
						<div class="w-10 h-10 rounded-lg bg-amber-500/10 text-amber-400 flex items-center justify-center">
							<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>
						</div>
						<div>
							<p class="text-xl font-bold text-white/90">{missingEnvCount}</p>
							<p class="text-xs text-white/30">Missing Env</p>
						</div>
					</div>
					<div class="rounded-xl bg-white/[0.02] border border-white/[0.06] p-4 flex items-center gap-3">
						<div class="w-10 h-10 rounded-lg bg-red-500/10 text-red-400 flex items-center justify-center">
							<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="15" y1="9" x2="9" y2="15"/><line x1="9" y1="9" x2="15" y2="15"/></svg>
						</div>
						<div>
							<p class="text-xl font-bold text-white/90">{errorCount}</p>
							<p class="text-xs text-white/30">Errored</p>
						</div>
					</div>
					<div class="rounded-xl bg-white/[0.02] border border-white/[0.06] p-4 flex items-center gap-3">
						<div class="w-10 h-10 rounded-lg bg-white/5 text-white/40 flex items-center justify-center">
							<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/></svg>
						</div>
						<div>
							<p class="text-xl font-bold text-white/90">{sleepingCount}</p>
							<p class="text-xs text-white/30">Disabled</p>
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
						<option value="sleeping">Disabled</option>
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
												<span class="flex items-center gap-1.5 {server.status_text}">
													<span class="w-1.5 h-1.5 rounded-full {server.status_dot}"></span>
													{server.status_label}
												</span>
												{#if server.status_detail}
													<span class="text-[10px] text-white/20">{server.status_detail}</span>
												{/if}
											</div>
										</td>
										<td class="text-white/50">{server.tools_count}</td>
										<td class="text-right pr-4">
											<div class="flex items-center justify-end gap-1">
												{#if server.installed}
													<button
														on:click|stopPropagation={() => openConfigModal(server.id)}
														class="p-1.5 rounded-md hover:bg-white/[0.06] text-white/30 hover:text-white/70 transition-colors"
														title="Configure"
													>
														<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>
													</button>
													{#if server.status_raw === 'healthy'}
														<button
															on:click|stopPropagation={() => handleStop(server.id)}
															class="p-1.5 rounded-md hover:bg-white/[0.06] text-white/30 hover:text-white/70 transition-colors"
															title="Stop"
														>
															<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="6" y="4" width="4" height="16"/><rect x="14" y="4" width="4" height="16"/></svg>
														</button>
													{:else if server.status_raw !== 'error'}
														<button
															on:click|stopPropagation={() => handleStart(server.id)}
															class="p-1.5 rounded-md hover:bg-white/[0.06] text-white/30 hover:text-emerald-400 transition-colors"
															title="Start"
														>
															<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="5 3 19 12 5 21 5 3"/></svg>
														</button>
													{/if}
												{:else}
													<button
														on:click|stopPropagation={() => handleInstall(server.id)}
														class="px-2 py-1 rounded-md bg-orange-500/10 text-orange-400 hover:bg-orange-500/20 transition-colors text-[10px] font-medium"
														title="Install"
													>
														Install
													</button>
												{/if}
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
										<span class="w-2 h-2 rounded-full {server.status_dot}"></span>
									</div>
									<p class="text-[11px] text-white/25 mb-3 line-clamp-2">{server.description}</p>
									<div class="flex items-center justify-between mb-3">
										<span class="px-2 py-0.5 rounded text-[10px] font-medium {runtimeClass(server.runtime)}">{server.runtime}</span>
										<span class="text-[10px] text-white/30">{server.tools_count} tools</span>
									</div>
									<div class="flex items-center gap-1.5">
										{#if server.installed}
											<button
												on:click|stopPropagation={() => openConfigModal(server.id)}
												class="flex-1 flex items-center justify-center gap-1 py-1.5 rounded-md bg-white/[0.03] border border-white/[0.06] text-[10px] text-white/50 hover:text-white/80 hover:bg-white/[0.06] transition-colors"
											>
												<svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>
												Configure
											</button>
											{#if server.status_raw === 'healthy'}
												<button
													on:click|stopPropagation={() => handleStop(server.id)}
													class="flex-1 flex items-center justify-center gap-1 py-1.5 rounded-md bg-white/[0.03] border border-white/[0.06] text-[10px] text-white/50 hover:text-red-400 hover:bg-red-500/10 transition-colors"
												>
													<svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="6" y="4" width="4" height="16"/><rect x="14" y="4" width="4" height="16"/></svg>
													Stop
												</button>
											{:else if server.status_raw !== 'error'}
												<button
													on:click|stopPropagation={() => handleStart(server.id)}
													class="flex-1 flex items-center justify-center gap-1 py-1.5 rounded-md bg-emerald-500/10 border border-emerald-500/20 text-[10px] text-emerald-400 hover:bg-emerald-500/20 transition-colors"
												>
													<svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="5 3 19 12 5 21 5 3"/></svg>
													Start
												</button>
											{/if}
										{:else}
											<button
												on:click|stopPropagation={() => handleInstall(server.id)}
												class="w-full flex items-center justify-center gap-1 py-1.5 rounded-md bg-orange-500/10 border border-orange-500/20 text-[10px] text-orange-400 hover:bg-orange-500/20 transition-colors"
											>
												<svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
												Install
											</button>
										{/if}
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
			{/if}
		</div>
	</div>

	<!-- Detail Panel -->
	{#if $selectedServerId && $serverDetail}
		<div class="w-96 border-l border-white/[0.06] bg-[#0a0a0f] flex flex-col fixed right-0 top-0 bottom-0 z-40">
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
						{#if $serverDetail.status === 'running'}
							<span class="flex items-center gap-1.5 text-[11px] text-emerald-400">
								<span class="w-1.5 h-1.5 rounded-full bg-emerald-500"></span>
								Running
							</span>
						{:else if $serverDetail.status === 'error'}
							<span class="flex items-center gap-1.5 text-[11px] text-red-400">
								<span class="w-1.5 h-1.5 rounded-full bg-red-500"></span>
								Error
							</span>
						{:else}
							<span class="flex items-center gap-1.5 text-[11px] text-white/40">
								<span class="w-1.5 h-1.5 rounded-full bg-white/30"></span>
								Sleeping
							</span>
						{/if}
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
									<span class="w-1.5 h-1.5 rounded-full {$serverDetail.status === 'running' ? 'bg-emerald-500' : $serverDetail.status === 'error' ? 'bg-red-500' : 'bg-white/30'}"></span>
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

					<!-- API Key / Auth -->
					{#if $serverDetail}
						{@const mcp = servers.find(s => s.id === $serverDetail.id)}
						{@const envVars = getMcpEnvValues($serverDetail.id)}
						{@const patKey = mcp?.patProvider ? `${mcp.patProvider.toUpperCase()}_TOKEN` : null}
						{#if patKey}
							<div class="rounded-lg bg-white/[0.02] border border-white/[0.06] p-4">
								<h4 class="text-xs font-medium text-white/70 mb-3">Authentication</h4>
								<div class="space-y-3">
									<div class="flex items-center justify-between">
										<span class="text-[11px] text-white/40">Required Token</span>
										<span class="text-xs text-white/70 font-mono">{patKey}</span>
									</div>
									<div class="flex items-center justify-between">
										<span class="text-[11px] text-white/40">Status</span>
										{#if envVars[patKey]}
											<span class="flex items-center gap-1 text-[11px] text-emerald-400">
												<span class="w-1.5 h-1.5 rounded-full bg-emerald-500"></span>
												Set
											</span>
										{:else}
											<span class="flex items-center gap-1 text-[11px] text-amber-400">
												<span class="w-1.5 h-1.5 rounded-full bg-amber-400"></span>
												Not set
											</span>
										{/if}
									</div>
									<div class="flex gap-2">
										<input
											value={apiKeyInput[$serverDetail.id] ?? ''}
											on:input={(e) => { if ($serverDetail) apiKeyInput[$serverDetail.id] = (e.target as HTMLInputElement).value; }}
											placeholder={envVars[patKey] ? 'Update token...' : `Enter ${mcp.patProvider} token...`}
											type="password"
											class="flex-1 bg-white/[0.03] border border-white/[0.06] rounded-lg px-3 py-1.5 text-xs text-white/80 placeholder-white/20 focus:outline-none focus:border-orange-500/30"
										/>
										<button
											on:click={async () => {
												const val = apiKeyInput[$serverDetail.id];
												if (val && $serverDetail) {
													await setMcpEnv($serverDetail.id, { [patKey]: val });
													toast.success('API key saved');
													apiKeyInput[$serverDetail.id] = '';
													await fetchServerConfig($serverDetail.id);
												}
											}}
											class="px-3 py-1.5 rounded-lg bg-orange-500/10 border border-orange-500/20 text-xs text-orange-400 hover:bg-orange-500/20 transition-colors"
										>
											Save
										</button>
									</div>
								</div>
							</div>
						{/if}
					{/if}

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

{#if showConfigModal && configMcpId}
	<McpConfigModal
		mcpId={configMcpId}
		mcpName={configMcpName}
		mcpDescription={configMcpDescription}
		config={configMcpConfig}
		health={configMcpHealth}
		onClose={closeConfigModal}
		onSaveAndStart={handleSaveAndStart}
		onSaveOnly={handleSaveOnly}
		onTestConnection={handleTestConnection}
		onAutoDetect={handleAutoDetect}
	/>
{/if}
