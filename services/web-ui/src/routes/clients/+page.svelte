<script lang="ts">
	import { clients, connectClient, disconnectClient } from '$lib/stores';
	import ClientCard from '$lib/components/ClientCard.svelte';

	$: connectedCount = $clients.filter((c) => c.connected).length;
</script>

<div class="p-8">
	<div class="mb-8">
		<h1 class="text-xl font-bold text-white/95">Clients</h1>
		<p class="text-sm text-white/30 mt-1">
			Connect Mach1 to your favorite AI tools. {connectedCount} of {$clients.length} connected.
		</p>
	</div>

	<!-- Info banner -->
	<div class="mb-6 p-4 rounded-xl bg-violet-900/10 border border-violet-800/30 flex items-start gap-3">
		<span class="text-lg">⚡</span>
		<div>
			<p class="text-sm font-medium text-white/80">One-click Mach1 activation</p>
			<p class="text-xs text-white/40 mt-1 leading-relaxed">
				Click <strong class="text-violet-400">Activate Mach1</strong> on any client below to instantly configure it.
				This runs <code class="text-violet-400/80 bg-black/30 px-1 py-0.5 rounded text-xs">onemcpctl {'{connect command}'}</code> which writes the MCP config file so your AI tool routes through Mach1 with access to all your installed servers.
			</p>
		</div>
	</div>

	<div class="space-y-3">
		{#each $clients as client (client.id)}
			<ClientCard
				{client}
				onActivate={() => connectClient(client.id)}
				onDeactivate={() => disconnectClient(client.id)}
			/>
		{/each}
	</div>
</div>
