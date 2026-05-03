<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { toast } from '$lib/toast';
	import {
		marketplace,
		installed,
		installMcp,
		uninstallMcp,
		skills,
		installSkill,
		uninstallSkill,
		bundles,
		installBundle,
		uninstallBundle,
		fetchMarketplace,
		fetchInstalled,
		fetchBundles,
		selectedMarketplaceItem,
		marketplaceItemDetail,
		selectMarketplaceItem,
		setMcpEnv,
		autoDetectEnv,
		fetchMarketplaceItemDetail
	} from '$lib/stores';
	import { user } from '$lib/auth';
	import PublishModal from '$lib/components/PublishModal.svelte';
	import EnvSetupModal from '$lib/components/EnvSetupModal.svelte';

	type Tab = 'mcps' | 'skills' | 'bundles';
	type SortOption = 'downloads' | 'rating' | 'newest';
	type TrustFilter = 'all' | 'anthropic-official' | '1mcp-verified' | 'community';
	type RuntimeFilter = 'all' | 'node' | 'python' | 'go' | 'binary';
	type StatusFilter = 'all' | 'installed' | 'available';

	let activeTab: Tab = 'mcps';
	let query = '';
	let sort: SortOption = 'downloads';
	let trustFilter: TrustFilter = 'all';
	let runtimeFilter: RuntimeFilter = 'all';
	let statusFilter: StatusFilter = 'all';
	let showPublish = false;
	let actionLoading = false;
	let envInputs: Record<string, string> = {};
	let envVisible: Record<string, boolean> = {};
	let envConfigured: Record<string, boolean> = {};
	let currentPage = 1;
	const pageSize = 8;

	// Env setup modal state
	let showEnvModal = false;
	let envModalMcpId = '';
	let envModalMcpName = '';
	let envModalRequiredEnv: string[] = [];

	onMount(async () => {
		await fetchInstalled();
		await fetchMarketplace();
		await fetchBundles();
	});

	function fmt(n: number): string {
		if (n >= 1000) return `${(n / 1000).toFixed(n >= 10000 ? 0 : 1)}k`;
		return String(n);
	}

	function trustClass(status: string): string {
		switch (status) {
			case 'anthropic-official':
				return 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20';
			case '1mcp.in-verified':
			case '1mcp-verified':
				return 'bg-blue-500/10 text-blue-400 border-blue-500/20';
			case 'community':
				return 'bg-amber-500/10 text-amber-400 border-amber-500/20';
			default:
				return 'bg-white/5 text-white/40 border-white/10';
		}
	}

	function trustLabel(status: string): string {
		switch (status) {
			case 'anthropic-official': return 'anthropic-official';
			case '1mcp.in-verified':
			case '1mcp-verified': return '1mcp-verified';
			case 'community': return 'community';
			default: return status;
		}
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

	function statusDot(installed: boolean): string {
		return installed ? 'bg-emerald-500' : 'bg-white/20';
	}

	function statusText(installed: boolean): string {
		return installed ? 'Installed' : 'Available';
	}

	function cleanAuthor(author: string): string {
		try {
			const u = new URL(author);
			if (u.hostname) {
				const parts = u.pathname.replace(/^\//, '').replace(/\.git$/, '').split('/');
				return parts.slice(0, 2).join('/');
			}
		} catch {
			// not a URL
		}
		return author;
	}

	$: filteredMcps = (() => {
		let result = [...$marketplace];
		if (query.trim()) {
			const q = query.toLowerCase();
			result = result.filter(
				(m) =>
					m.name.toLowerCase().includes(q) ||
					m.shortDescription.toLowerCase().includes(q) ||
					m.tags.some((t) => t.includes(q)) ||
					m.author.toLowerCase().includes(q)
			);
		}
		if (trustFilter !== 'all') {
			result = result.filter((m) => m.verificationStatus === trustFilter);
		}
		if (runtimeFilter !== 'all') {
			result = result.filter((m) => m.runtime === runtimeFilter);
		}
		if (statusFilter === 'installed') {
			result = result.filter((m) => m.installed);
		} else if (statusFilter === 'available') {
			result = result.filter((m) => !m.installed);
		}
		result.sort((a, b) => {
			if (sort === 'downloads') return b.downloads - a.downloads;
			if (sort === 'rating') return b.rating - a.rating;
			return new Date(b.publishedAt).getTime() - new Date(a.publishedAt).getTime();
		});
		return result;
	})();

	$: totalPages = Math.max(1, Math.ceil(filteredMcps.length / pageSize));
	$: paginatedMcps = filteredMcps.slice((currentPage - 1) * pageSize, currentPage * pageSize);

	$: filteredSkills = $skills.filter((s) => {
		if (!query.trim()) return true;
		const q = query.toLowerCase();
		return s.name.toLowerCase().includes(q) || s.description.toLowerCase().includes(q);
	});

	function handlePublish(data: { name: string; description: string; version: string; runtime: 'node' | 'python' | 'go' | 'binary'; tags: string[]; verificationStatus: 'verified' | 'unverified'; fileName: string }) {
		marketplace.update((list) => [
			{
				id: `${$user?.name ?? 'anon'}-${data.name.toLowerCase().replace(/\s+/g, '-')}-${Date.now()}`,
				name: data.name,
				shortDescription: data.description,
				version: data.version,
				runtime: data.runtime,
				author: $user?.name ?? 'you',
				tags: data.tags,
				rating: 0,
				reviewCount: 0,
				downloads: 0,
				verificationStatus: data.verificationStatus === 'verified' ? 'pending' : 'community',
				publishedAt: new Date().toISOString().split('T')[0],
				installed: false
			},
			...list
		]);
	}

	async function handleInstall(id: string) {
		actionLoading = true;
		try {
			await installMcp(id);
			// Auto-select the item to show detail view
			selectMarketplaceItem(id);
			// Refresh the detail to get updated installed status
			await fetchMarketplaceItemDetail(id);
			// Show env setup modal if required
			const detail = $marketplaceItemDetail;
			if (detail?.requires_env && detail.requires_env.length > 0) {
				envModalMcpId = id;
				envModalMcpName = detail.name;
				envModalRequiredEnv = detail.requires_env;
				showEnvModal = true;
			}
		} finally {
			actionLoading = false;
		}
	}

	async function handleUninstall(id: string) {
		actionLoading = true;
		try {
			await uninstallMcp(id);
			if ($selectedMarketplaceItem === id) {
				selectMarketplaceItem(id);
			}
		} finally {
			actionLoading = false;
		}
	}
</script>

<div class="flex h-full">
	<!-- Main Content -->
	<div class="flex-1 flex flex-col min-w-0" class:pr-96={$selectedMarketplaceItem}>
		<div class="p-6 space-y-5">
			<!-- Header -->
			<div class="flex items-start justify-between">
				<div>
					<h1 class="text-xl font-bold text-white/95">Marketplace</h1>
					<p class="text-sm text-white/30 mt-1">Discover and install verified MCP servers.</p>
				</div>
				<div class="flex items-center gap-3">
					<div class="flex items-center gap-2 text-[11px] text-white/40">
						<span class="w-1.5 h-1.5 rounded-full bg-emerald-500"></span>
						Registry synced
						<span class="text-white/20">2m ago</span>
						<button on:click={() => toast.info('Syncing registry...')} class="text-white/20 hover:text-white/50 transition-colors" aria-label="Sync registry">
							<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 12a9 9 0 0 0-9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/><path d="M3 3v5h5"/><path d="M3 12a9 9 0 0 0 9 9 9.75 9.75 0 0 0 6.74-2.74L21 16"/><path d="M16 16h5v5"/></svg>
						</button>
					</div>
					<button on:click={async () => { try { await fetchMarketplace(); toast.success('Registry up to date'); } catch { toast.error('Sync failed'); } }} class="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-white/[0.03] border border-white/[0.06] text-xs text-white/60 hover:text-white/90 hover:bg-white/[0.06] transition-colors">
						<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 12a9 9 0 0 0-9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/><path d="M3 3v5h5"/><path d="M3 12a9 9 0 0 0 9 9 9.75 9.75 0 0 0 6.74-2.74L21 16"/><path d="M16 16h5v5"/></svg>
						Check for updates
					</button>
					{#if activeTab === 'mcps'}
						<button on:click={() => (showPublish = true)} class="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-orange-500 text-white text-xs font-medium hover:bg-orange-600 transition-colors">
							<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
							Publish MCP
						</button>
					{/if}
				</div>
			</div>

			<!-- Tabs -->
			<div class="flex items-center gap-1 border-b border-white/[0.06]">
				<button
					on:click={() => activeTab = 'mcps'}
					class="px-4 py-2 text-sm transition-colors relative {activeTab === 'mcps' ? 'text-orange-400 font-medium' : 'text-white/30 hover:text-white/60'}"
				>
					MCPs
					{#if activeTab === 'mcps'}
						<div class="absolute bottom-0 left-0 right-0 h-0.5 bg-orange-500 rounded-t-full"></div>
					{/if}
				</button>
				<button
					on:click={() => activeTab = 'skills'}
					class="px-4 py-2 text-sm transition-colors relative {activeTab === 'skills' ? 'text-orange-400 font-medium' : 'text-white/30 hover:text-white/60'}"
				>
					Skills
					{#if activeTab === 'skills'}
						<div class="absolute bottom-0 left-0 right-0 h-0.5 bg-orange-500 rounded-t-full"></div>
					{/if}
				</button>
				<button
					on:click={() => activeTab = 'bundles'}
					class="px-4 py-2 text-sm transition-colors relative {activeTab === 'bundles' ? 'text-orange-400 font-medium' : 'text-white/30 hover:text-white/60'}"
				>
					Bundles
					{#if activeTab === 'bundles'}
						<div class="absolute bottom-0 left-0 right-0 h-0.5 bg-orange-500 rounded-t-full"></div>
					{/if}
				</button>
			</div>

			{#if activeTab === 'mcps'}
				<!-- Filters -->
				<div class="flex items-center gap-3">
					<div class="relative flex-1 max-w-sm">
						<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="absolute left-3 top-1/2 -translate-y-1/2 text-white/25"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>
						<input bind:value={query} placeholder="Search MCPs, tools, capabilities, or manifests..." class="w-full bg-white/[0.03] border border-white/[0.06] rounded-lg pl-9 pr-3 py-2 text-sm text-white/80 placeholder-white/20 focus:outline-none focus:border-orange-500/30 transition-colors" />
					</div>
					<select bind:value={trustFilter} class="bg-white/[0.03] border border-white/[0.06] rounded-lg px-3 py-2 text-xs text-white/60 focus:outline-none focus:border-orange-500/30">
						<option value="all">Trust: All</option>
						<option value="anthropic-official">Anthropic Official</option>
						<option value="1mcp.in-verified">1mcp Verified</option>
						<option value="community">Community</option>
					</select>
					<select bind:value={runtimeFilter} class="bg-white/[0.03] border border-white/[0.06] rounded-lg px-3 py-2 text-xs text-white/60 focus:outline-none focus:border-orange-500/30">
						<option value="all">Runtime: All</option>
						<option value="node">Node</option>
						<option value="python">Python</option>
						<option value="go">Go</option>
						<option value="binary">Binary</option>
					</select>
					<select bind:value={statusFilter} class="bg-white/[0.03] border border-white/[0.06] rounded-lg px-3 py-2 text-xs text-white/60 focus:outline-none focus:border-orange-500/30">
						<option value="all">Status: All</option>
						<option value="installed">Installed</option>
						<option value="available">Available</option>
					</select>
					<select bind:value={sort} class="bg-white/[0.03] border border-white/[0.06] rounded-lg px-3 py-2 text-xs text-white/60 focus:outline-none focus:border-orange-500/30">
						<option value="downloads">Sort by: Top Downloads</option>
						<option value="rating">Sort by: Rating</option>
						<option value="newest">Sort by: Newest</option>
					</select>
				</div>

				<!-- Table -->
				<div class="rounded-xl bg-white/[0.02] border border-white/[0.06] overflow-hidden">
					<table class="w-full text-left">
						<thead>
							<tr class="border-b border-white/[0.04]">
								<th class="pb-2 pt-3 px-4 text-[11px] font-medium text-white/30 uppercase tracking-wider">Name</th>
								<th class="pb-2 pt-3 text-[11px] font-medium text-white/30 uppercase tracking-wider">Maintainer</th>
								<th class="pb-2 pt-3 text-[11px] font-medium text-white/30 uppercase tracking-wider">Runtime</th>
								<th class="pb-2 pt-3 text-[11px] font-medium text-white/30 uppercase tracking-wider">Trust</th>
								<th class="pb-2 pt-3 text-[11px] font-medium text-white/30 uppercase tracking-wider">Version</th>
								<th class="pb-2 pt-3 text-[11px] font-medium text-white/30 uppercase tracking-wider">Installs</th>
								<th class="pb-2 pt-3 text-[11px] font-medium text-white/30 uppercase tracking-wider">Status</th>
								<th class="pb-2 pt-3 text-[11px] font-medium text-white/30 uppercase tracking-wider text-right">Actions</th>
							</tr>
						</thead>
						<tbody class="text-xs">
							{#each paginatedMcps as mcp}
								<tr
									class="border-b border-white/[0.02] hover:bg-white/[0.02] transition-colors cursor-pointer {$selectedMarketplaceItem === mcp.id ? 'bg-white/[0.03]' : ''}"
									on:click={() => selectMarketplaceItem(mcp.id)}
								>
									<td class="py-3 px-4">
										<div class="flex items-center gap-3">
											<div class="w-8 h-8 rounded-md bg-white/[0.04] border border-white/[0.06] flex items-center justify-center text-sm">
												{mcp.name.charAt(0)}
											</div>
											<div>
												<div class="flex items-center gap-2">
													<p class="text-white/80 font-medium">{mcp.name}</p>
													{#if mcp.requires_env && mcp.requires_env.length > 0}
														<span class="px-1.5 py-0.5 rounded text-[9px] font-medium bg-amber-500/10 text-amber-400 border border-amber-500/20" title="Requires API token: {mcp.requires_env.join(', ')}">
															<svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="inline -mt-0.5 mr-0.5"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
															API Key
														</span>
													{/if}
												</div>
												<p class="text-[10px] text-white/25 truncate max-w-[240px]">{mcp.shortDescription}</p>
											</div>
										</div>
									</td>
									<td class="text-white/50">{cleanAuthor(mcp.author)}</td>
									<td>
										<span class="px-2 py-0.5 rounded text-[10px] font-medium {runtimeClass(mcp.runtime)}">{mcp.runtime}</span>
									</td>
									<td>
										<span class="flex items-center gap-1 px-2 py-0.5 rounded text-[10px] font-medium border {trustClass(mcp.verificationStatus)}">
											<svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/></svg>
											{trustLabel(mcp.verificationStatus)}
										</span>
									</td>
									<td class="text-white/50">v{mcp.version}</td>
									<td class="text-white/50">{fmt(mcp.downloads)}</td>
									<td>
										<span class="flex items-center gap-1.5 {mcp.installed ? 'text-emerald-400' : 'text-white/30'}">
											<span class="w-1.5 h-1.5 rounded-full {statusDot(mcp.installed)}"></span>
											{statusText(mcp.installed)}
										</span>
									</td>
									<td class="text-right pr-4">
										<div class="flex items-center justify-end gap-1">
											{#if mcp.installed}
												<button
													on:click|stopPropagation={() => handleUninstall(mcp.id)}
													disabled={actionLoading}
													class="p-1.5 rounded-md hover:bg-white/[0.06] text-emerald-400 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
													title="Installed"
												>
													<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="20 6 9 17 4 12"/></svg>
												</button>
											{:else}
												<button
													on:click|stopPropagation={() => handleInstall(mcp.id)}
													disabled={actionLoading}
													class="p-1.5 rounded-md hover:bg-white/[0.06] text-orange-400 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
													title="Install"
												>
													<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
												</button>
											{/if}

										</div>
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
					{#if paginatedMcps.length === 0}
						<div class="flex flex-col items-center justify-center py-16 gap-3">
							<span class="text-2xl opacity-20">📦</span>
							<p class="text-sm text-white/40">No MCPs found</p>
						</div>
					{/if}
				</div>

				<!-- Pagination -->
				<div class="flex items-center justify-between">
					<p class="text-[11px] text-white/20">Showing {(currentPage - 1) * pageSize + 1} to {Math.min(currentPage * pageSize, filteredMcps.length)} of {filteredMcps.length} MCPs</p>
					{#if totalPages > 1}
						<div class="flex items-center gap-1">
							<button on:click={() => currentPage = Math.max(1, currentPage - 1)} class="p-1.5 rounded-md text-white/30 hover:text-white/60 hover:bg-white/[0.04] transition-colors" disabled={currentPage === 1} aria-label="Previous page">
								<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="15 18 9 12 15 6"/></svg>
							</button>
							{#each Array(totalPages) as _, i}
								<button on:click={() => currentPage = i + 1} class="w-7 h-7 rounded-md text-xs transition-colors {currentPage === i + 1 ? 'bg-orange-500/10 text-orange-400 border border-orange-500/20' : 'text-white/30 hover:text-white/60 hover:bg-white/[0.04]'}">
									{i + 1}
								</button>
							{/each}
							<button on:click={() => currentPage = Math.min(totalPages, currentPage + 1)} class="p-1.5 rounded-md text-white/30 hover:text-white/60 hover:bg-white/[0.04] transition-colors" disabled={currentPage === totalPages} aria-label="Next page">
								<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="9 18 15 12 9 6"/></svg>
							</button>
						</div>
					{/if}
				</div>
			{:else if activeTab === 'skills'}
				<!-- Skills tab -->
				<div class="flex items-center gap-3 mb-6">
					<div class="relative flex-1 max-w-sm">
						<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="absolute left-3 top-1/2 -translate-y-1/2 text-white/25"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>
						<input bind:value={query} placeholder="Search skills..." class="w-full bg-white/[0.03] border border-white/[0.06] rounded-lg pl-9 pr-3 py-2 text-sm text-white/80 placeholder-white/20 focus:outline-none focus:border-orange-500/30 transition-colors" />
					</div>
				</div>
				{#if filteredSkills.length === 0}
					<div class="flex flex-col items-center justify-center py-24 gap-3">
						<p class="text-sm font-medium text-white/80">No skills found</p>
						<p class="text-xs text-white/30">Try a different search term.</p>
					</div>
				{:else}
					<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
						{#each filteredSkills as skill (skill.id)}
							<div class="rounded-xl border border-white/[0.06] bg-white/[0.03] p-5 flex flex-col gap-3 hover:border-white/[0.12] transition-all">
								<div class="flex items-center gap-2">
									<span class="text-lg">{skill.icon}</span>
									<h3 class="text-sm font-semibold text-white/90">{skill.name}</h3>
								</div>
								<p class="text-xs text-white/40 leading-relaxed flex-1">{skill.description}</p>
								<div class="flex flex-wrap gap-1">
									{#each skill.mcp_ids as mcpId}
										<span class="text-[10px] px-2 py-0.5 rounded bg-white/[0.04] text-white/40 border border-white/[0.06]">{mcpId}</span>
									{/each}
								</div>
								{#if skill.installed}
									<button on:click={() => uninstallSkill(skill.id)} disabled={actionLoading} class="w-full text-xs py-1.5 px-3 rounded-lg border border-white/[0.06] text-white/40 hover:text-red-400 hover:border-red-900/60 hover:bg-red-900/10 transition-colors disabled:opacity-50 disabled:cursor-not-allowed">Uninstall</button>
								{:else}
									<button on:click={() => installSkill(skill.id)} disabled={actionLoading} class="w-full text-xs py-1.5 px-3 rounded-lg bg-orange-500 text-white hover:bg-orange-600 transition-colors font-medium disabled:opacity-50 disabled:cursor-not-allowed">Install</button>
								{/if}
							</div>
						{/each}
					</div>
				{/if}
			{/if}

			<!-- Bundles Tab -->
			{#if activeTab === 'bundles'}
				{#if $bundles.length === 0}
					<div class="flex flex-col items-center justify-center py-24 gap-3">
						<p class="text-sm font-medium text-white/80">No bundles found</p>
						<p class="text-xs text-white/30">Bundles are collections of MCPs that work together.</p>
					</div>
				{:else}
					<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
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
									<button on:click={() => uninstallBundle(bundle.id)} disabled={actionLoading} class="w-full text-xs py-1.5 px-3 rounded-lg border border-white/[0.06] text-white/40 hover:text-red-400 hover:border-red-900/60 hover:bg-red-900/10 transition-colors disabled:opacity-50 disabled:cursor-not-allowed">Uninstall</button>
								{:else}
									<button on:click={() => installBundle(bundle.id)} disabled={actionLoading} class="w-full text-xs py-1.5 px-3 rounded-lg bg-orange-500 text-white hover:bg-orange-600 transition-colors font-medium disabled:opacity-50 disabled:cursor-not-allowed">Install Bundle</button>
								{/if}
							</div>
						{/each}
					</div>
				{/if}
			{/if}
		</div>
	</div>

	<!-- Detail Panel -->
	{#if $selectedMarketplaceItem && $marketplaceItemDetail}
		<div class="w-96 border-l border-white/[0.06] bg-[#0a0a0f] flex flex-col fixed right-0 top-0 bottom-0 z-40 overflow-y-auto">
			<div class="p-5 space-y-5">
				<!-- Header -->
				<div class="flex items-start justify-between">
					<div class="flex items-center gap-3">
						<div class="w-10 h-10 rounded-lg bg-white/[0.04] border border-white/[0.06] flex items-center justify-center text-lg">
							{$marketplaceItemDetail.name.charAt(0)}
						</div>
						<div>
							<h3 class="text-sm font-bold text-white/90">{$marketplaceItemDetail.name}</h3>
							<p class="text-[11px] text-white/30">by {cleanAuthor($marketplaceItemDetail.author)}</p>
						</div>
					</div>
					<div class="flex items-center gap-2">
						<span class="px-2 py-0.5 rounded text-[10px] font-medium border {trustClass($marketplaceItemDetail.trust)}">{$marketplaceItemDetail.trust}</span>
						<button on:click={() => selectMarketplaceItem(null)} class="text-white/20 hover:text-white/60 transition-colors p-1" aria-label="Close">
							<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
						</button>
					</div>
				</div>

				<p class="text-xs text-white/40">{$marketplaceItemDetail.description}</p>

				<!-- Stats -->
				<div class="grid grid-cols-3 gap-2">
					<div class="rounded-lg bg-white/[0.02] border border-white/[0.06] p-3 text-center">
						<div class="flex items-center justify-center gap-1 text-white/60 mb-1">
							<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
							<span class="text-xs font-medium">{fmt($marketplaceItemDetail.downloads)}</span>
						</div>
						<p class="text-[10px] text-white/25">Installs</p>
					</div>
					<div class="rounded-lg bg-white/[0.02] border border-white/[0.06] p-3 text-center">
						<div class="flex items-center justify-center gap-1 text-amber-400 mb-1">
							<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"/></svg>
							<span class="text-xs font-medium">{$marketplaceItemDetail.rating.toFixed(1)}</span>
						</div>
						<p class="text-[10px] text-white/25">Rating</p>
					</div>
					<div class="rounded-lg bg-white/[0.02] border border-white/[0.06] p-3 text-center">
						<div class="flex items-center justify-center gap-1 text-white/60 mb-1">
							<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="4" width="18" height="18" rx="2" ry="2"/><line x1="16" y1="2" x2="16" y2="6"/><line x1="8" y1="2" x2="8" y2="6"/><line x1="3" y1="10" x2="21" y2="10"/></svg>
							<span class="text-xs font-medium">{$marketplaceItemDetail.updated_at}</span>
						</div>
						<p class="text-[10px] text-white/25">Updated</p>
					</div>
				</div>

				<!-- Details -->
				<div class="rounded-lg bg-white/[0.02] border border-white/[0.06] p-4 space-y-2.5">
					<h4 class="text-xs font-medium text-white/70 mb-1">Details</h4>
					<div class="flex items-center justify-between">
						<span class="text-[11px] text-white/40">Runtime</span>
						<span class="text-xs text-white/70">{$marketplaceItemDetail.runtime}</span>
					</div>
					<div class="flex items-center justify-between">
						<span class="text-[11px] text-white/40">Version</span>
						<span class="text-xs text-white/70">v{$marketplaceItemDetail.version}</span>
					</div>
					<div class="flex items-center justify-between">
						<span class="text-[11px] text-white/40">License</span>
						<span class="text-xs text-white/70">{$marketplaceItemDetail.license}</span>
					</div>
					<div class="flex items-center justify-between">
						<span class="text-[11px] text-white/40">Trust</span>
						<span class="px-1.5 py-0.5 rounded text-[9px] font-medium border {trustClass($marketplaceItemDetail.trust)}">{$marketplaceItemDetail.trust}</span>
					</div>
					<div class="flex items-center justify-between">
						<span class="text-[11px] text-white/40">Digest</span>
						<div class="flex items-center gap-1">
							<span class="text-[10px] text-white/30 font-mono">{$marketplaceItemDetail.sha256.slice(0, 16)}...</span>
							<button on:click={() => { navigator.clipboard.writeText($marketplaceItemDetail.sha256); toast.success('SHA256 copied'); }} class="text-white/20 hover:text-white/50 transition-colors" aria-label="Copy SHA256">
								<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>
							</button>
						</div>
					</div>
					<div class="flex items-center justify-between">
						<span class="text-[11px] text-white/40">Verified</span>
						<div class="flex items-center gap-1">
							<span class="text-[10px] text-white/30">{$marketplaceItemDetail.verified_at}</span>
							<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="text-emerald-400"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/></svg>
						</div>
					</div>
				</div>

				<!-- Capabilities -->
				<div>
					<h4 class="text-xs font-medium text-white/70 mb-2">Capabilities</h4>
					<div class="flex flex-wrap gap-1.5">
						{#each $marketplaceItemDetail.capabilities as cap}
							<span class="px-2 py-1 rounded bg-white/[0.03] border border-white/[0.06] text-[11px] text-white/50">{cap}</span>
						{/each}
					</div>
				</div>

				<!-- Security -->
				<div class="rounded-lg bg-white/[0.02] border border-white/[0.06] p-4 space-y-2">
					<h4 class="text-xs font-medium text-white/70 mb-1">Security</h4>
					{#each $marketplaceItemDetail.security_checks as check}
						<div class="flex items-center gap-2">
							{#if check.status === 'passed'}
								<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="text-emerald-400"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/></svg>
							{:else if check.status === 'warning'}
								<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="text-amber-400"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>
							{:else}
								<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="text-red-400"><circle cx="12" cy="12" r="10"/><line x1="15" y1="9" x2="9" y2="15"/><line x1="9" y1="9" x2="15" y2="15"/></svg>
							{/if}
							<span class="text-[11px] text-white/50">{check.label}</span>
						</div>
					{/each}
					{#each $marketplaceItemDetail.requires_env as env}
						<div class="flex items-center gap-2">
							<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="text-amber-400"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>
							<span class="text-[11px] text-white/50">Environment variable required: {env}</span>
						</div>
					{/each}
				</div>

				<!-- Environment Variables (Local Storage) -->
				{#if $marketplaceItemDetail.installed && $marketplaceItemDetail.requires_env && $marketplaceItemDetail.requires_env.length > 0}
					<div class="rounded-lg bg-white/[0.02] border border-white/[0.06] p-4 space-y-3">
						<div class="flex items-center justify-between">
							<h4 class="text-xs font-medium text-white/70">Environment Variables</h4>
							<span class="text-[9px] text-white/20">stored locally</span>
						</div>
						{#each $marketplaceItemDetail.requires_env as envKey}
							<div class="space-y-1.5">
								<div class="flex items-center gap-1">
									<span class="text-[10px] text-white/40 font-mono">{envKey}</span>
									{#if envConfigured[envKey]}
										<span class="text-[9px] text-emerald-400">Saved</span>
									{/if}
								</div>
								<div class="flex gap-2">
									<div class="relative flex-1">
										<input
											value={envInputs[envKey] ?? ''}
											on:input={(e) => { envInputs[envKey] = (e.target as HTMLInputElement).value; }}
											placeholder="Enter value..."
											type={envVisible[envKey] ? 'text' : 'password'}
											class="w-full bg-white/[0.03] border border-white/[0.06] rounded-lg pl-3 pr-8 py-1.5 text-xs text-white/80 placeholder-white/20 focus:outline-none focus:border-orange-500/30"
										/>
										<button
											on:click={() => { envVisible[envKey] = !envVisible[envKey]; }}
											class="absolute right-2 top-1/2 -translate-y-1/2 text-white/30 hover:text-white/60 transition-colors"
											aria-label={envVisible[envKey] ? 'Hide token' : 'Show token'}
										>
											{#if envVisible[envKey]}
												<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24"/><line x1="1" y1="1" x2="23" y2="23"/></svg>
											{:else}
												<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/></svg>
											{/if}
										</button>
									</div>
									<button
										on:click={async () => {
											const val = envInputs[envKey];
											if (!val) return;
											try {
												await setMcpEnv($marketplaceItemDetail.id, { [envKey]: val });
												envConfigured[envKey] = true;
												toast.success(`${envKey} saved`);
												envInputs[envKey] = '';
											} catch {
												toast.error(`Failed to save ${envKey}`);
											}
										}}
										class="px-3 py-1.5 rounded-lg bg-orange-500/10 border border-orange-500/20 text-xs text-orange-400 hover:bg-orange-500/20 transition-colors"
									>
										Save
									</button>
								</div>
							</div>
						{/each}
						{#if $marketplaceItemDetail.patProvider}
							<button
								on:click={async () => {
									try {
										const detected = await autoDetectEnv($marketplaceItemDetail.id);
										const keys = Object.keys(detected);
										if (keys.length > 0) {
											await setMcpEnv($marketplaceItemDetail.id, detected);
											for (const k of keys) envConfigured[k] = true;
											toast.success(`Auto-detected ${keys.join(', ')}`);
										} else {
											toast.info('No tokens found to auto-detect');
										}
									} catch { toast.error('Auto-detect failed'); }
								}}
								class="w-full text-[11px] text-orange-400/60 hover:text-orange-400 transition-colors"
							>
								Auto-detect token from local config
							</button>
						{/if}
						<button on:click={() => goto('/servers')} class="w-full text-[11px] text-white/30 hover:text-white/50 transition-colors">
							Open full server config →
						</button>
					</div>
				{/if}

			<!-- Actions -->
				<div class="space-y-2">
					{#if $marketplaceItemDetail.installed}
						<button on:click={() => handleUninstall($marketplaceItemDetail.id)} disabled={actionLoading} class="w-full flex items-center justify-center gap-2 py-2 rounded-lg bg-red-500/10 border border-red-500/20 text-xs text-red-400 hover:bg-red-500/20 transition-colors disabled:opacity-50 disabled:cursor-not-allowed">
							<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>
							Uninstall
						</button>
					{:else}
						<button on:click={() => handleInstall($marketplaceItemDetail.id)} disabled={actionLoading} class="w-full flex items-center justify-center gap-2 py-2 rounded-lg bg-orange-500 text-white text-xs font-medium hover:bg-orange-600 transition-colors disabled:opacity-50 disabled:cursor-not-allowed">
							<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
							Install
						</button>
					{/if}
				<button on:click={() => goto('/servers')} class="w-full flex items-center justify-center gap-2 py-2 rounded-lg bg-white/[0.03] border border-white/[0.06] text-xs text-white/60 hover:text-white/90 hover:bg-white/[0.06] transition-colors">
						<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"/></svg>
						Manage Server Settings
					</button>
					<button on:click={() => toast.info('Manifest view coming soon')} class="w-full flex items-center justify-center gap-2 py-2 rounded-lg bg-white/[0.03] border border-white/[0.06] text-xs text-white/60 hover:text-white/90 hover:bg-white/[0.06] transition-colors">
						<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="16 18 22 12 16 6"/><polyline points="8 6 2 12 8 18"/></svg>
						View Manifest JSON
					</button>
				</div>
			</div>
		</div>
	{/if}
</div>

{#if showPublish}
	<PublishModal onClose={() => (showPublish = false)} onPublish={handlePublish} />
{/if}

<EnvSetupModal
	bind:show={showEnvModal}
	mcpId={envModalMcpId}
	mcpName={envModalMcpName}
	requiredEnv={envModalRequiredEnv}
	on:complete={() => {
		showEnvModal = false;
		// Refresh installed list to show updated status
		fetchInstalled();
	}}
	on:skip={() => {
		showEnvModal = false;
		toast.info('You can configure credentials later in the server settings');
	}}
/>

<style>
</style>
