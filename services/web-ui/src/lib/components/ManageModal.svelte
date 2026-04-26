<script lang="ts">
	import type { InstalledMcp } from '$lib/types';

	export let mcp: InstalledMcp;
	export let onClose: () => void;
	export let onSave: (tokens: Record<string, string>) => void;
	export let onToggle: () => void;

	const PAT_PROVIDERS: Record<string, { label: string; placeholder: string; url: string | null; urlLabel: string | null; hint: string | null }> = {
		github: { label: 'GitHub Personal Access Token', placeholder: 'ghp_xxxxxxxxxxxxxxxxxxxx', url: 'https://github.com/settings/tokens', urlLabel: 'Manage on GitHub', hint: 'Required scopes: repo, read:org' },
		gitlab: { label: 'GitLab Personal Access Token', placeholder: 'glpat-xxxxxxxxxxxxxxxxxxxx', url: 'https://gitlab.com/-/user_settings/personal_access_tokens', urlLabel: 'Manage on GitLab', hint: 'Required scopes: api, read_repository' },
		linear: { label: 'Linear API Key', placeholder: 'lin_api_xxxxxxxxxxxxxxxx', url: 'https://linear.app/settings/api', urlLabel: 'Manage on Linear', hint: 'Personal API key from Linear settings' },
		custom: { label: 'API Token', placeholder: 'Enter your token', url: null, urlLabel: null, hint: null }
	};

	let tokens: Record<string, string> = {};
	let showToken = false;
	let localEnabled = mcp.enabled;

	$: provider = mcp.patProvider ? PAT_PROVIDERS[mcp.patProvider] : null;
	$: tokenKey = mcp.patProvider ?? 'token';

	function handleToggle() {
		localEnabled = !localEnabled;
		onToggle();
	}

	function handleSave() {
		onSave(tokens);
		onClose();
	}
</script>

<!-- svelte-ignore a11y-click-events-have-key-events -->
<!-- svelte-ignore a11y-no-static-element-interactions -->
<div class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm" on:click|self={onClose}>
	<div class="w-full max-w-md bg-[#12121a] border border-white/[0.06] rounded-2xl shadow-2xl">
		<!-- Header -->
		<div class="flex items-center justify-between px-5 py-4 border-b border-white/[0.06]">
			<div>
				<h2 class="text-sm font-semibold text-white/90">{mcp.name}</h2>
				<p class="text-xs text-white/30 mt-0.5">v{mcp.version} · {mcp.runtime}</p>
			</div>
			<button on:click={onClose} class="w-8 h-8 flex items-center justify-center rounded-lg text-white/30 hover:text-white/80 hover:bg-white/[0.06] transition-colors">
				<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
			</button>
		</div>

		<div class="px-5 py-5 space-y-6">
			<!-- Toggle -->
			<div class="flex items-center justify-between">
				<div class="flex items-center gap-2.5">
					<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" class="{localEnabled ? 'text-emerald-400' : 'text-white/30'}">
						<path d="M18.36 6.64a9 9 0 1 1-12.73 0"/><line x1="12" y1="2" x2="12" y2="12"/>
					</svg>
					<div>
						<p class="text-sm font-medium text-white/90">{localEnabled ? 'Running' : 'Disabled'}</p>
						<p class="text-xs text-white/30 mt-0.5">{localEnabled ? 'Server is active and routing traffic' : 'Server is paused'}</p>
					</div>
				</div>
				<button on:click={handleToggle} class="relative w-11 h-6 rounded-full transition-colors flex-shrink-0" style="background: {localEnabled ? '#7c3aed' : '#2a2a3a'}">
					<span class="absolute top-1 w-4 h-4 bg-white rounded-full transition-all shadow-sm" style="left: {localEnabled ? '1.375rem' : '0.25rem'}"></span>
				</button>
			</div>

			<!-- Command -->
			<div>
				<p class="text-xs font-semibold text-white/30 uppercase tracking-wider mb-2">Command</p>
				<code class="block text-xs text-white/70 bg-black/40 border border-white/[0.06] rounded-lg px-3 py-2.5 font-mono break-all leading-relaxed">{mcp.command}</code>
			</div>

			<!-- PAT -->
			{#if provider}
				<div>
					<div class="flex items-center justify-between mb-2">
						<p class="text-xs font-semibold text-white/30 uppercase tracking-wider flex items-center gap-1.5">
							<svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="m21 2-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0 3 3L22 7l-3-3m-3.5 3.5L19 4"/></svg>
							Access Token
						</p>
						{#if provider.url}
							<a href={provider.url} target="_blank" rel="noopener noreferrer" class="text-xs text-violet-400 hover:text-violet-300 flex items-center gap-1 transition-colors">
								{provider.urlLabel}
								<svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/><polyline points="15 3 21 3 21 9"/><line x1="10" y1="14" x2="21" y2="3"/></svg>
							</a>
						{/if}
					</div>
					<p class="text-xs text-white/30 mb-2">{provider.label}</p>
					<div class="relative">
						<input
							type={showToken ? 'text' : 'password'}
							value={tokens[tokenKey] ?? ''}
							on:input={(e) => { tokens = { ...tokens, [tokenKey]: e.currentTarget.value }; }}
							placeholder={provider.placeholder}
							class="w-full bg-black/40 border border-white/[0.06] rounded-lg px-3 py-2 text-sm text-white/80 placeholder-white/20 focus:outline-none focus:border-violet-500 transition-colors pr-10"
						/>
						<button type="button" on:click={() => (showToken = !showToken)} class="absolute right-2.5 top-1/2 -translate-y-1/2 text-white/30 hover:text-white/70">
							{#if showToken}
								<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24"/><line x1="1" y1="1" x2="23" y2="23"/></svg>
							{:else}
								<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/></svg>
							{/if}
						</button>
					</div>
					{#if provider.hint}
						<p class="text-xs text-white/15 mt-1.5">{provider.hint}</p>
					{/if}
				</div>
			{/if}
		</div>

		<!-- Footer -->
		<div class="flex items-center justify-end gap-2 px-5 py-4 border-t border-white/[0.06]">
			<button on:click={onClose} class="text-sm px-4 py-1.5 rounded-lg text-white/40 hover:text-white/80 hover:bg-white/[0.06] transition-colors">Cancel</button>
			<button on:click={handleSave} class="text-sm px-4 py-1.5 rounded-lg bg-violet-600 text-white hover:bg-violet-700 transition-colors font-medium">Save Changes</button>
		</div>
	</div>
</div>
