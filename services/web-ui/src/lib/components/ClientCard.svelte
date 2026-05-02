<script lang="ts">
	import type { ClientApp } from '$lib/types';
	import { browser } from '$app/environment';
	import { connectClient, disconnectClient } from '$lib/stores';

	export let client: ClientApp;

	const isTauri = browser && '__TAURI_INTERNALS__' in window;
	let settingUp = false;
	let showManual = false;
	let removing = false;

	const daemonUrl = 'http://127.0.0.1:3200/mcp';

	const manualInstructions: Record<string, string> = {
		vscode: `Add to your VS Code mcp.json (run "MCP: Open User Configuration" from Command Palette):

"mach1": {
  "type": "http",
  "url": "${daemonUrl}"
}`,

		cursor: `Add to ~/.cursor/mcp.json:

"mach1": {
  "url": "${daemonUrl}"
}`,

		claude: `Add to Claude Desktop's claude_desktop_config.json:

"mach1": {
  "command": "<path-to-mach1>",
  "args": ["--db", "<path-to-registry.db>"]
}

Claude Desktop's local config path is stdio-only, so it uses the local spawn fallback.`,

		claudecode: `Add to ~/.claude.json under "mcpServers":

"mach1": {
  "type": "http",
  "url": "${daemonUrl}"
}

Or use the CLI:
claude mcp add --transport http mach1 ${daemonUrl}`,

		windsurf: `Add to ~/.codeium/mcp_config.json:

"mach1": {
  "serverUrl": "${daemonUrl}"
}`,

		codex: `Add to ~/.codex/config.toml:

[mcp_servers.mach1]
url = "${daemonUrl}"
type = "http"`,

		antigravity: `Add to ~/.antigravity/mcp.json:

"mach1": {
  "type": "http",
  "url": "${daemonUrl}"
}`,

		opencode: `Add to ~/.config/opencode/opencode.json:

"mach1": {
  "type": "remote",
  "url": "${daemonUrl}",
  "enabled": true
}`,
	};

	async function handleSetup() {
		if (!isTauri) {
			showManual = !showManual;
			if (showManual) {
				setTimeout(() => {
					alert('Desktop app required for auto-setup. Use the manual instructions shown below to configure your IDE.');
				}, 100);
			}
			return;
		}
		settingUp = true;
		try {
				await connectClient(client.id);
		} catch (error: unknown) {
			const message = typeof error === 'string' ? error : error instanceof Error ? error.message : 'Unknown error';
			if (message.includes('not yet supported')) {
				showManual = true;
			} else {
				alert(`Setup failed: ${message}\n\nPlease configure manually using the instructions below.`);
				showManual = true;
			}
		}
		settingUp = false;
	}

	async function handleDisconnect() {
		removing = true;
		try {
			await disconnectClient(client.id);
		} catch (error: unknown) {
			const message = typeof error === 'string' ? error : error instanceof Error ? error.message : 'Unknown error';
			alert(`Disconnect failed: ${message}`);
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
		{#if client.id === 'claude' || client.id === 'claudecode' || client.id === 'codex'}
			<a
				href="https://github.com/SaiAvinashPatoju/1mcp.in"
				target="_blank"
				rel="noopener noreferrer"
				class="text-xs px-4 py-1.5 rounded-lg border border-white/[0.06] text-amber-400/60 hover:text-amber-300 hover:bg-white/[0.04] transition-colors font-medium flex-shrink-0 inline-flex items-center gap-2"
			>
				<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/></svg>
				Docs
			</a>
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
	{/if}
</div>

{#if showManual}
	<div class="mt-2 ml-12 p-3 rounded-lg bg-black/40 border border-white/[0.06] text-xs font-mono text-white/60 whitespace-pre-line leading-relaxed">
		{manualInstructions[client.id] ?? 'Add "mach1" to your client MCP configuration and point it at http://127.0.0.1:3200/mcp.'}
		<button on:click={() => showManual = false} class="mt-2 block text-violet-400 hover:text-violet-300 font-sans text-[11px]">Dismiss</button>
	</div>
{/if}
