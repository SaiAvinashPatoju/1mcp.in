<script lang="ts">
	import type { Skill } from '$lib/types';
	import { installed } from '$lib/stores';

	export let skill: Skill;
	export let onInstall: () => void;
	export let onUninstall: () => void;

	$: mcpDetails = $installed.filter((m) => skill.mcp_ids.includes(m.id));
	$: installedCount = mcpDetails.length;
	$: totalMcps = skill.mcp_ids.length;
</script>

<div class="rounded-xl border border-white/[0.06] bg-white/[0.03] p-5 flex flex-col gap-3 hover:border-white/[0.12] transition-all duration-200 backdrop-blur-sm">
	<div class="flex items-start gap-3">
		<span class="text-2xl flex-shrink-0">{skill.icon}</span>
		<div class="min-w-0 flex-1">
			<h3 class="text-sm font-semibold text-white/90">{skill.name}</h3>
			<p class="text-xs text-white/30 mt-0.5">{skill.description}</p>
		</div>
	</div>

	<div class="flex flex-wrap gap-1.5">
		{#each mcpDetails as mcp}
			<span class="text-xs px-2 py-0.5 rounded bg-violet-900/20 text-violet-400 border border-violet-800/30">{mcp.name}</span>
		{/each}
		{#each skill.mcp_ids.filter((id) => !mcpDetails.some((m) => m.id === id)) as missingId}
			<span class="text-xs px-2 py-0.5 rounded bg-white/[0.04] text-white/30 border border-white/[0.06] border-dashed">{missingId}</span>
		{/each}
	</div>

	<div class="text-xs text-white/30 mt-auto">
		{installedCount}/{totalMcps} MCPs
	</div>

	{#if skill.installed}
		<button on:click={onUninstall} class="w-full text-xs py-1.5 px-3 rounded-lg border border-white/[0.06] text-white/40 hover:text-red-400 hover:border-red-900/60 hover:bg-red-900/10 transition-colors">
			Uninstall
		</button>
	{:else}
		<button on:click={onInstall} class="w-full text-xs py-1.5 px-3 rounded-lg bg-violet-600 text-white hover:bg-violet-700 transition-colors font-medium">
			Install
		</button>
	{/if}
</div>