<script lang="ts">
	import { marketplace, installed, installMcp, uninstallMcp } from '$lib/stores';
	import { user } from '$lib/auth';
	import McpCard from '$lib/components/McpCard.svelte';
	import PublishModal from '$lib/components/PublishModal.svelte';

	type SortOption = 'rating' | 'downloads' | 'newest';
	type FilterOption = 'all' | 'verified' | 'community' | 'installed';

	let query = '';
	let sort: SortOption = 'rating';
	let filter: FilterOption = 'all';
	let showPublish = false;
	const filterOptions: FilterOption[] = ['all', 'verified', 'community', 'installed'];

	$: filtered = (() => {
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
		if (filter === 'verified') result = result.filter((m) => trustRank(m.verificationStatus) <= 1);
		else if (filter === 'community') result = result.filter((m) => trustRank(m.verificationStatus) > 1);
		else if (filter === 'installed') result = result.filter((m) => m.installed);

		result.sort((a, b) => {
			if (trustRank(a.verificationStatus) !== trustRank(b.verificationStatus)) return trustRank(a.verificationStatus) - trustRank(b.verificationStatus);
			if (sort === 'rating') return b.rating - a.rating;
			if (sort === 'downloads') return b.downloads - a.downloads;
			return new Date(b.publishedAt).getTime() - new Date(a.publishedAt).getTime();
		});
		return result;
	})();

	$: verifiedList = filtered.filter((m) => trustRank(m.verificationStatus) <= 1);
	$: communityList = filtered.filter((m) => trustRank(m.verificationStatus) > 1);
	$: showSectioned = filter === 'all' && !query.trim();

	$: totalVerified = $marketplace.filter((m) => trustRank(m.verificationStatus) <= 1).length;
	$: totalCommunity = $marketplace.filter((m) => trustRank(m.verificationStatus) > 1).length;

	function trustRank(status: string): number {
		if (status === 'anthropic-official') return 0;
		if (status === 'onemcp-verified' || status === 'verified') return 1;
		if (status === 'pending') return 2;
		return 3;
	}

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
</script>

<div class="p-8">
	<div class="flex items-center justify-between mb-6">
		<div>
			<h1 class="text-xl font-bold text-white/95">Marketplace</h1>
			<p class="text-sm text-white/30 mt-1">{totalVerified} verified · {totalCommunity} community</p>
		</div>
		<button on:click={() => (showPublish = true)} class="flex items-center gap-2 text-sm px-4 py-2 rounded-lg bg-violet-600 text-white hover:bg-violet-700 transition-colors font-medium">
			⬆ Publish MCP
		</button>
	</div>

	<!-- Filters -->
	<div class="flex items-center gap-3 mb-8 flex-wrap">
		<div class="relative flex-1 min-w-48 max-w-sm">
			<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" class="absolute left-3 top-1/2 -translate-y-1/2 text-white/25"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>
			<input bind:value={query} placeholder="Search servers, tags, authors…" class="w-full bg-white/[0.03] border border-white/[0.06] rounded-lg pl-9 pr-3 py-2 text-sm text-white/80 placeholder-white/20 focus:outline-none focus:border-violet-500 transition-colors" />
		</div>
		<div class="flex items-center gap-1 bg-white/[0.03] border border-white/[0.06] rounded-lg p-1">
			{#each filterOptions as f}
				<button on:click={() => (filter = f)} class="text-xs px-3 py-1 rounded-md transition-colors capitalize {filter === f ? 'bg-violet-600 text-white' : 'text-white/30 hover:text-white/60'}">
					{f}
				</button>
			{/each}
		</div>
		<div class="flex items-center gap-2 bg-white/[0.03] border border-white/[0.06] rounded-lg px-3 py-2">
			<svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" class="text-white/25"><line x1="4" y1="21" x2="4" y2="14"/><line x1="4" y1="10" x2="4" y2="3"/><line x1="12" y1="21" x2="12" y2="12"/><line x1="12" y1="8" x2="12" y2="3"/><line x1="20" y1="21" x2="20" y2="16"/><line x1="20" y1="12" x2="20" y2="3"/></svg>
			<select bind:value={sort} class="bg-transparent text-xs text-white/80 focus:outline-none cursor-pointer">
				<option value="rating">Top Rated</option>
				<option value="downloads">Most Downloaded</option>
				<option value="newest">Newest</option>
			</select>
		</div>
	</div>

	{#if showSectioned}
		{#if verifiedList.length > 0}
			<section class="mb-10">
				<div class="flex items-center gap-2 mb-4">
					<span class="text-emerald-400 text-xs">🛡</span>
					<span class="text-xs font-semibold text-white/30 uppercase tracking-wider">Verified</span>
				</div>
				<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
					{#each verifiedList as mcp (mcp.id)}
						<McpCard {mcp} onInstall={() => installMcp(mcp.id)} onUninstall={() => uninstallMcp(mcp.id)} />
					{/each}
				</div>
			</section>
		{/if}

		{#if communityList.length > 0}
			<section>
				<div class="flex items-center gap-2 mb-4">
					<span class="text-yellow-400 text-xs">⚠</span>
					<span class="text-xs font-semibold text-white/30 uppercase tracking-wider">Community</span>
				</div>
				<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
					{#each communityList as mcp (mcp.id)}
						<McpCard {mcp} onInstall={() => installMcp(mcp.id)} onUninstall={() => uninstallMcp(mcp.id)} />
					{/each}
				</div>
			</section>
		{/if}
	{:else}
		<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
			{#if filtered.length === 0}
				<div class="col-span-full flex flex-col items-center justify-center py-24 gap-3">
					<p class="text-sm font-medium text-white/80">No results found</p>
					<p class="text-xs text-white/30">Try a different search term or filter.</p>
				</div>
			{:else}
				{#each filtered as mcp (mcp.id)}
					<McpCard {mcp} onInstall={() => installMcp(mcp.id)} onUninstall={() => uninstallMcp(mcp.id)} />
				{/each}
			{/if}
		</div>
	{/if}

	{#if showPublish}
		<PublishModal onClose={() => (showPublish = false)} onPublish={handlePublish} />
	{/if}
</div>
