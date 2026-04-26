<script lang="ts">
	import type { ClientApp } from '$lib/types';

	export let client: ClientApp;
	export let onActivate: () => void;
	export let onDeactivate: () => void;
</script>

<div class="rounded-xl border border-white/[0.06] bg-white/[0.03] p-5 flex items-center gap-4 hover:border-white/[0.12] transition-all duration-200 backdrop-blur-sm">
	<span class="text-2xl select-none">{client.icon}</span>
	<div class="flex-1 min-w-0">
		<div class="flex items-center gap-2">
			<h3 class="text-sm font-semibold text-white/90">{client.name}</h3>
			{#if client.connected}
				<span class="flex items-center gap-1 text-xs px-1.5 py-0.5 rounded bg-emerald-900/20 text-emerald-400 border border-emerald-800/50">
					<span class="w-1.5 h-1.5 rounded-full bg-emerald-400 shadow-[0_0_6px_#34d399]"></span>
					Connected
				</span>
			{/if}
		</div>
		<p class="text-xs text-white/30 mt-0.5">{client.description}</p>
	</div>
	{#if client.connected}
		<button on:click={onDeactivate} class="text-xs px-4 py-1.5 rounded-lg border border-white/[0.06] text-white/40 hover:text-red-400 hover:border-red-900/60 hover:bg-red-900/10 transition-colors flex-shrink-0">
			Disconnect
		</button>
	{:else}
		<button on:click={onActivate} class="text-xs px-4 py-1.5 rounded-lg bg-violet-600 text-white hover:bg-violet-700 transition-colors font-medium flex-shrink-0">
			Activate Mach1
		</button>
	{/if}
</div>
