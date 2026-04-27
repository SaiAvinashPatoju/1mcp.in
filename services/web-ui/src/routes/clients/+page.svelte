<script lang="ts">
	import { clients, connectClient, disconnectClient } from '$lib/stores';
	import ClientCard from '$lib/components/ClientCard.svelte';

	$: connectedCount = $clients.filter((c) => c.connected).length;
</script>

<div class="p-8">
	<div class="mb-8">
		<h1 class="text-xl font-bold text-white/95">Clients</h1>
		<p class="text-sm text-white/30 mt-1">
                        Connect 1mcp to your favorite AI tools. {connectedCount} of {$clients.length} connected.
                </p>
        </div>

        <!-- Info banner -->
        <div class="mb-6 p-4 rounded-xl bg-violet-900/10 border border-violet-800/30 flex items-start gap-3">
                <span class="text-lg">⚡</span>
                <div>
                        <p class="text-sm font-medium text-white/80">Seamless 1mcp Setup</p>
                        <p class="text-xs text-white/40 mt-1 leading-relaxed">
                                Click <strong class="text-violet-400">Setup 1mcp</strong> on any client below to seamlessly configure it.
                                This routes your AI tool through 1mcp, gaining access to all installed servers dynamically.                        </p>
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
