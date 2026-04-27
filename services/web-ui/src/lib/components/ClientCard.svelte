<script lang="ts">
	import type { ClientApp } from '$lib/types';
	import { browser } from '$app/environment';
	import { connectClient, disconnectClient } from '$lib/stores';

	export let client: ClientApp;

	const isTauri = browser && '__TAURI_INTERNALS__' in window;
	let settingUp = false;
	let showManual = false;
	let removing = false;

	const manualInstructions: Record<string, string> = {
		vscode: '1. Run "MCP: Open User Configuration" in VS Code\n2. Add under "servers":\n   "mach1": { "command": "<absolute-path-to-mach1>", "args": ["--db", "<path-to-db>"] }',
		cursor: '1. Create/edit ~/.cursor/mcp.json\n2. Add under "mcpServers":\n   "mach1": { "command": "<absolute-path-to-mach1>", "args": ["--db", "<path-to-db>"] }',
		claude: '1. Open Claude Desktop -> Settings -> Developer -> Edit Config\n2. Add under "mcpServers":\n   "mach1": { "command": "<absolute-path-to-mach1>", "args": ["--db", "<path-to-db>"] }',
		claudecode: '1. Edit ~/.claude.json\n2. Add under "mcpServers":\n   "mach1": { "command": "<absolute-path-to-mach1>", "args": ["--db", "<path-to-db>"] }',
		windsurf: '1. Edit ~/.codeium/windsurf/mcp_config.json\n2. Add under "mcpServers":\n   "mach1": { "command": "<absolute-path-to-mach1>", "args": ["--db", "<path-to-db>"] }',
		codex: '1. Edit ~/.codex/mcp.json\n2. Add under "mcpServers":\n   "mach1": { "command": "<absolute-path-to-mach1>", "args": ["--db", "<path-to-db>"] }',
		opencode: '1. Edit ~/.config/opencode/opencode.json\n2. Add under "mcp":\n   "mach1": { "type": "local", "command": ["<absolute-path-to-mach1>", "--db", "<path-to-db>"], "enabled": true }',
	};

	async function handleSetup() {
		if (!isTauri) {
			showManual = !showManual;
			return;
		}
		settingUp = true;
		try {
			await connectClient(client.id);
		} catch (e: any) {
			const msg = typeof e === 'string' ? e : e?.message ?? 'Unknown error';
			if (msg.includes('not yet supported')) {
				showManual = true;
			} else {
				alert(`Setup failed: ${msg}\n\nPlease configure manually.`);
				showManual = true;
			}
		}
		settingUp = false;
	}

	async function handleDisconnect() {
		removing = true;
		try {
			await disconnectClient(client.id);
		} catch (e: any) {
			const msg = typeof e === 'string' ? e : e?.message ?? 'Unknown error';
			alert(`Disconnect failed: ${msg}`);
		}
		removing = false;
	}
</script>

<div class="rounded-xl border border-white/[0.06] bg-white/[0.03] p-5 flex items-center gap-4 hover:border-white/[0.12] transition-all duration-200 backdrop-blur-sm">
	<span class="text-2xl select-none">{client.icon}</span>
	<div class="flex-1 min-w-0">
		<div class="flex items-center gap-2">
			<h3 class="text-sm font-semibold text-white/90">{client.name}</h3>
			{#if client.connected}
				<span class="flex items-center gap-1 text-xs px-1.5 py-0.5 rounded bg-emerald-900/20 text-emerald-400 border border-emerald-800/50">
					<span class="w-1.5 h-1.5 rounded-full bg-emerald-400 shadow-[0_0_6px_#34d399]"></span>
					Mach1 Connected
				</span>
			{/if}
		</div>
		<p class="text-xs text-white/30 mt-0.5">{client.description}</p>
	</div>
	{#if client.connected}
		<button disabled={removing} on:click={handleDisconnect} class="text-xs px-4 py-1.5 rounded-lg border border-white/[0.06] text-white/40 hover:text-red-400 hover:border-red-900/60 hover:bg-red-900/10 transition-colors flex-shrink-0 disabled:opacity-50 disabled:cursor-not-allowed">
			{removing ? 'Disconnecting...' : 'Disconnect'}
		</button>
	{:else}
		<button disabled={settingUp} on:click={handleSetup} class="text-xs px-4 py-1.5 rounded-lg bg-violet-600 text-white hover:bg-violet-700 transition-colors font-medium flex-shrink-0 disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2">
			{#if settingUp}
				<svg class="animate-spin h-3.5 w-3.5 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
					<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
					<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
				</svg>
				Setting up...
			{:else}
				Setup Mach1
			{/if}
		</button>
	{/if}
</div>

{#if showManual}
	<div class="mt-2 ml-12 p-3 rounded-lg bg-black/40 border border-white/[0.06] text-xs font-mono text-white/60 whitespace-pre-line leading-relaxed">
		{manualInstructions[client.id] ?? 'Add "mach1" to your client\'s MCP server configuration with command "mach1".'}
		<button on:click={() => showManual = false} class="mt-2 block text-violet-400 hover:text-violet-300 font-sans text-[11px]">Dismiss</button>
	</div>
{/if}
