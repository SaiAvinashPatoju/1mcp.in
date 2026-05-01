<script lang="ts">
	import type { ServerConfig, McpHealthResult } from '$lib/types';
	import { onMount, onDestroy } from 'svelte';

	export let mcpId: string;
	export let mcpName: string;
	export let mcpDescription: string;
	export let config: ServerConfig | null = null;
	export let health: McpHealthResult | null = null;
	export let onClose: () => void;
	export let onSaveAndStart: (vars: Record<string, string>) => void;
	export let onSaveOnly: (vars: Record<string, string>) => void;
	export let onTestConnection: () => Promise<McpHealthResult>;
	export let onAutoDetect: () => Promise<Record<string, string>>;

	let vars: Record<string, string> = {};
	let showSecret: Record<string, boolean> = {};
	let testing = false;
	let saving = false;
	let localHealth: McpHealthResult | null = health;
	let autoDetectLoading = false;
	let modalEl: HTMLDivElement;

	$: envList = config?.env ?? [];
	$: requiredEnvKeys = envList.map((e) => e.key);
	$: missingKeys = requiredEnvKeys.filter((k) => !vars[k] || vars[k].trim() === '');

	function initVars() {
		const initial: Record<string, string> = {};
		for (const e of envList) {
			initial[e.key] = e.value ?? '';
		}
		vars = initial;
	}

	$: if (config) initVars();

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') onClose();
	}

	onMount(() => {
		document.addEventListener('keydown', handleKeydown);
		if (modalEl) {
			const firstInput = modalEl.querySelector('input, button, select, textarea') as HTMLElement | null;
			firstInput?.focus();
		}
	});

	onDestroy(() => {
		document.removeEventListener('keydown', handleKeydown);
	});

	async function handleTestConnection() {
		testing = true;
		try {
			localHealth = await onTestConnection();
		} catch (e) {
			localHealth = {
				status: 'unhealthy',
				process_status: 'unknown',
				auth_status: 'failed',
				last_check: new Date().toISOString(),
				error: e instanceof Error ? e.message : 'Health check failed'
			};
		}
		testing = false;
	}

	async function handleAutoDetect() {
		autoDetectLoading = true;
		try {
			const detected = await onAutoDetect();
			vars = { ...vars, ...detected };
		} catch {
			// silently fail — user can input manually
		}
		autoDetectLoading = false;
	}

	function handleSaveAndStart() {
		saving = true;
		onSaveAndStart({ ...vars });
	}

	function handleSaveOnly() {
		saving = true;
		onSaveOnly({ ...vars });
	}

	function envVarDescription(key: string): string {
		const lower = key.toLowerCase();
		if (lower.includes('github')) return 'GitHub Personal Access Token with repo scope';
		if (lower.includes('linear')) return 'Linear API key from your Linear settings';
		if (lower.includes('slack')) return 'Slack Bot User OAuth Token';
		if (lower.includes('notion')) return 'Notion integration token';
		if (lower.includes('openai')) return 'OpenAI API key';
		if (lower.includes('postgres') || lower.includes('database')) return 'PostgreSQL connection string or password';
		return `Value for ${key}`;
	}

	function healthBadgeClass(status: string): string {
		switch (status) {
			case 'healthy': return 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20';
			case 'unhealthy': return 'bg-red-500/10 text-red-400 border-red-500/20';
			default: return 'bg-white/5 text-white/40 border-white/10';
		}
	}
</script>

<!-- svelte-ignore a11y-click-events-have-key-events -->
<!-- svelte-ignore a11y-no-static-element-interactions -->
<div class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm" on:click|self={onClose}>
	<div bind:this={modalEl} id="mcp-config-{mcpId}" class="w-full max-w-lg bg-[#12121a] border border-white/[0.06] rounded-2xl shadow-2xl flex flex-col max-h-[90vh]" role="dialog" aria-modal="true" aria-labelledby="mcp-config-title">
		<!-- Header -->
		<div class="flex items-center justify-between px-5 py-4 border-b border-white/[0.06] shrink-0">
			<div>
				<h2 id="mcp-config-title" class="text-sm font-semibold text-white/90">{mcpName}</h2>
				<p class="text-xs text-white/30 mt-0.5 line-clamp-1">{mcpDescription}</p>
			</div>
			<button type="button" aria-label="Close modal" on:click={onClose} class="w-8 h-8 flex items-center justify-center rounded-lg text-white/30 hover:text-white/80 hover:bg-white/[0.06] transition-colors">
				<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
			</button>
		</div>

		<!-- Body -->
		<div class="flex-1 overflow-y-auto px-5 py-5 space-y-6">
			<!-- Environment Variables -->
			<div>
				<div class="flex items-center justify-between mb-3">
					<p class="text-xs font-semibold text-white/30 uppercase tracking-wider">Environment Variables</p>
					<button
						on:click={handleAutoDetect}
						disabled={autoDetectLoading}
						class="text-xs flex items-center gap-1.5 px-2 py-1 rounded-md bg-white/[0.03] border border-white/[0.06] text-white/50 hover:text-white/80 hover:bg-white/[0.06] transition-colors disabled:opacity-50"
					>
						<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class={autoDetectLoading ? 'animate-spin' : ''}><path d="M21 12a9 9 0 0 0-9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/><path d="M3 3v5h5"/><path d="M3 12a9 9 0 0 0 9 9 9.75 9.75 0 0 0 6.74-2.74L21 16"/><path d="M16 16h5v5"/></svg>
						Auto-detect
					</button>
				</div>

				{#if envList.length === 0}
					<p class="text-xs text-white/20">No environment variables required for this MCP.</p>
				{:else}
					<div class="space-y-3">
						{#each envList as env}
							<div class="rounded-lg bg-white/[0.02] border border-white/[0.06] p-3">
								<div class="flex items-center justify-between mb-1.5">
									<label for="env-{env.key}" class="text-xs font-medium text-white/70">{env.key}</label>
									<div class="flex items-center gap-1.5">
										{#if env.secret}
											<span class="px-1.5 py-0.5 rounded text-[9px] bg-amber-500/10 text-amber-400 border border-amber-500/20">Secret</span>
										{/if}
										{#if vars[env.key]?.trim()}
											<span class="flex items-center gap-1 text-[10px] text-emerald-400">
												<svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="20 6 9 17 4 12"/></svg>
												Set
											</span>
										{:else}
											<span class="flex items-center gap-1 text-[10px] text-red-400">
												<svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
												Missing
											</span>
										{/if}
									</div>
								</div>
								<p class="text-[11px] text-white/25 mb-2">{envVarDescription(env.key)}</p>
								<div class="relative">
									<input
										id="env-{env.key}"
										type={env.secret && !showSecret[env.key] ? 'password' : 'text'}
										value={vars[env.key] ?? ''}
										on:input={(e) => { vars = { ...vars, [env.key]: e.currentTarget.value }; }}
										placeholder={env.secret ? '••••••••' : `Enter ${env.key}`}
										class="w-full bg-black/40 border border-white/[0.06] rounded-lg px-3 py-2 text-sm text-white/80 placeholder-white/20 focus:outline-none focus:border-orange-500/40 transition-colors pr-10"
									/>
									{#if env.secret}
										<button
											type="button"
											aria-label={showSecret[env.key] ? 'Hide value' : 'Show value'}
											on:click={() => showSecret = { ...showSecret, [env.key]: !showSecret[env.key] }}
											class="absolute right-2.5 top-1/2 -translate-y-1/2 text-white/30 hover:text-white/70"
										>
											{#if showSecret[env.key]}
												<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24"/><line x1="1" y1="1" x2="23" y2="23"/></svg>
											{:else}
												<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/></svg>
											{/if}
										</button>
									{/if}
								</div>
							</div>
						{/each}
					</div>
				{/if}
			</div>

			<!-- Health Status -->
			<div class="rounded-lg bg-white/[0.02] border border-white/[0.06] p-4">
				<div class="flex items-center justify-between mb-3">
					<p class="text-xs font-semibold text-white/30 uppercase tracking-wider">Health Status</p>
					<button
						on:click={handleTestConnection}
						disabled={testing}
						class="text-xs flex items-center gap-1.5 px-2.5 py-1.5 rounded-md bg-white/[0.03] border border-white/[0.06] text-white/60 hover:text-white/90 hover:bg-white/[0.06] transition-colors disabled:opacity-50"
					>
						<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class={testing ? 'animate-spin' : ''}><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/></svg>
						Test Connection
					</button>
				</div>

				{#if localHealth}
					<div class="space-y-2">
						<div class="flex items-center justify-between">
							<span class="text-[11px] text-white/40">Overall</span>
							<span class="text-xs px-2 py-0.5 rounded border {healthBadgeClass(localHealth.status)}">
								{localHealth.status.charAt(0).toUpperCase() + localHealth.status.slice(1)}
							</span>
						</div>
						<div class="flex items-center justify-between">
							<span class="text-[11px] text-white/40">Process</span>
							<span class="text-xs text-white/70">{localHealth.process_status}</span>
						</div>
						<div class="flex items-center justify-between">
							<span class="text-[11px] text-white/40">Auth</span>
							<span class="text-xs text-white/70">{localHealth.auth_status}</span>
						</div>
						{#if localHealth.error}
							<div class="rounded-md bg-red-500/5 border border-red-500/10 p-2 mt-1">
								<p class="text-[11px] text-red-400">{localHealth.error}</p>
							</div>
						{/if}
					</div>
				{:else}
					<p class="text-xs text-white/20">Click "Test Connection" to check MCP health.</p>
				{/if}
			</div>
		</div>

		<!-- Footer -->
		<div class="flex items-center justify-end gap-2 px-5 py-4 border-t border-white/[0.06] shrink-0">
			<button on:click={onClose} class="text-sm px-4 py-1.5 rounded-lg text-white/40 hover:text-white/80 hover:bg-white/[0.06] transition-colors">Cancel</button>
			<button on:click={handleSaveOnly} disabled={saving} class="text-sm px-4 py-1.5 rounded-lg bg-white/[0.05] border border-white/[0.08] text-white/70 hover:text-white/90 hover:bg-white/[0.08] transition-colors disabled:opacity-50">
				Save Only
			</button>
			<button on:click={handleSaveAndStart} disabled={saving} class="text-sm px-4 py-1.5 rounded-lg bg-orange-500 text-white hover:bg-orange-600 transition-colors font-medium disabled:opacity-50">
				Save & Start
			</button>
		</div>
	</div>
</div>
