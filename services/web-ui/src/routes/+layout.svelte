<script lang="ts">
	import '../app.css';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { user, signOut, isAuthenticated, restoreSession } from '$lib/auth';
	import UpdateBanner from '$lib/components/UpdateBanner.svelte';
	import Toast from '$lib/components/Toast.svelte';
	import ZoomManager from '$lib/components/ZoomManager.svelte';
	import { browser } from '$app/environment';
	import { onMount, afterUpdate, onDestroy } from 'svelte';
	import { listen } from '@tauri-apps/api/event';
	import { startUserCounter, fetchInstalled, fetchMarketplace, fetchSkills, isConsoleExpanded, consoleTab } from '$lib/stores';
	
	const isTauri = browser && '__TAURI_INTERNALS__' in window;
	let sessionRestoring = true;

	const NAV = [
		{ href: '/dashboard', label: 'Dashboard', icon: 'dashboard' },
		{ href: '/servers', label: 'Servers', icon: 'servers' },
		{ href: '/discover', label: 'Discover', icon: 'marketplace' },
		{ href: '/clients', label: 'Clients', icon: 'clients' },
		{ href: '/settings', label: 'Settings', icon: 'settings' }
	];

	function navIcon(name: string): string {
		switch (name) {
			case 'dashboard':
				return `<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="7" height="7"/><rect x="14" y="3" width="7" height="7"/><rect x="14" y="14" width="7" height="7"/><rect x="3" y="14" width="7" height="7"/></svg>`;
			case 'servers':
				return `<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="2" y="2" width="20" height="8" rx="2" ry="2"/><rect x="2" y="14" width="20" height="8" rx="2" ry="2"/><line x1="6" y1="6" x2="6.01" y2="6"/><line x1="6" y1="18" x2="6.01" y2="18"/></svg>`;
			case 'clients':
				return `<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>`;
			case 'marketplace':
			case 'discover':
				return `<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M6 2L3 6v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2V6l-3-4z"/><line x1="3" y1="6" x2="21" y2="6"/><path d="M16 10a4 4 0 0 1-8 0"/></svg>`;
			case 'logs':
				return `<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/><polyline points="10 9 9 9 8 9"/></svg>`;
			case 'settings':
				return `<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"/></svg>`;
			default:
				return '';
		}
	}

	function handleSignOut() {
		signOut();
		goto('/');
	}

	$: currentPath = $page.url.pathname;

	function initials(name: string): string {
		return name
			.split(' ')
			.map((p) => p[0])
			.join('')
			.toUpperCase()
			.slice(0, 2);
	}

	// Console State
	let consoleContainer: HTMLElement;
	let logs: { time: string; level: string; source: string; message: string; type: 'info'|'warn'|'error'|'success' }[] = [];
	let filterText = '';
	let tauriUnlisten: (() => void) | null = null;

	let cliCommand = '';
	let cliHistory: { cmd: string; out: string; err: string }[] = [];
	let cliLoading = false;

	async function handleCliCommand(e: KeyboardEvent) {
		if (e.key !== 'Enter' || !cliCommand.trim() || cliLoading) return;
		const cmd = cliCommand.trim();
		cliCommand = '';
		cliLoading = true;
		try {
			const { invoke } = await import('@tauri-apps/api/core');
			const result = await invoke<{ output: string; error: string }>('execute_command', { command: cmd });
			cliHistory = [...cliHistory, { cmd, out: result.output, err: result.error }];
		} catch (err: any) {
			cliHistory = [...cliHistory, { cmd, out: '', err: err?.message ?? 'Command failed' }];
		}
		cliLoading = false;
	}

	function pushLog(source: string, message: string, type: 'info'|'warn'|'error'|'success' = 'info') {
		const time = new Date().toISOString().split('T')[1].slice(0, 12);
		const level = type === 'error' ? 'ERR' : type === 'warn' ? 'WRN' : type === 'success' ? 'OK' : 'INF';
		logs = [...logs.slice(-199), { time, level, source, message, type }];
	}

	onMount(async () => {
		pushLog('system', 'mach1 initialized - waiting for events...', 'success');

		// Restore session from stored token (localStorage/sessionStorage)
		sessionRestoring = true;
		await restoreSession();
		sessionRestoring = false;

		// Kick off background data syncs
		startUserCounter();
		await fetchInstalled();
		await fetchMarketplace();
		await fetchSkills();

		if (isTauri) {
			try {
				const unlisten = await listen<{ source: string; message: string; level: string }>('log-event', (event) => {
					const { source, message, level } = event.payload;
					const type = level === 'error' ? 'error' : level === 'warn' ? 'warn' : level === 'success' ? 'success' : 'info';
					pushLog(source, message, type);
				});
				tauriUnlisten = unlisten;
			} catch {
				pushLog('system', 'Running in browser mode — Tauri events unavailable', 'warn');
			}
		}
	});

	onDestroy(() => {
		if (tauriUnlisten) tauriUnlisten();
	});

	afterUpdate(() => {
		if (consoleContainer && isConsoleExpanded) {
			consoleContainer.scrollTop = consoleContainer.scrollHeight;
		}
	});

	$: filteredLogs = filterText
		? logs.filter(l => l.message.toLowerCase().includes(filterText.toLowerCase()) || l.source.toLowerCase().includes(filterText.toLowerCase()))
		: logs;
</script>

{#if sessionRestoring}
	<div class="flex items-center justify-center min-h-screen bg-black">
		<div class="flex flex-col items-center gap-4">
			<div class="w-8 h-8 border-2 border-orange-500/30 border-t-orange-500 rounded-full animate-spin"></div>
			<p class="text-sm text-white/30">Restoring session...</p>
		</div>
	</div>
{:else if !$isAuthenticated}
	<slot />
{:else}
	<div class="flex h-screen bg-[#0a0a0f] overflow-hidden">
		<!-- Sidebar -->
		<nav class="sidebar flex flex-col border-r border-white/[0.04] bg-[#0a0a0f] flex-shrink-0">
			<div class="flex items-center px-4 py-5 border-b border-white/[0.04]">
				<span class="text-sm font-bold text-white/90">1mcp.in</span>
			</div>

			<div class="flex-1 px-3 py-4 space-y-0.5">
				{#each NAV as { href, label, icon }}
					<a
						{href}
						class="flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm transition-all
							{currentPath === href || currentPath.startsWith(href + '/')
								? 'bg-orange-500/15 text-orange-400 font-medium border border-orange-500/20'
								: 'text-white/40 hover:text-white/70 hover:bg-white/[0.03] border border-transparent'}"
					>
						<span class="flex-shrink-0">{@html navIcon(icon)}</span>
						{label}
					</a>
				{/each}
			</div>

			{#if $user}
				<div class="px-3 py-4 border-t border-white/[0.04] space-y-3">
					<div class="flex items-center gap-2.5 px-2 py-1.5">
						<div class="w-8 h-8 rounded-full bg-orange-500 flex items-center justify-center text-xs font-bold text-white flex-shrink-0 select-none">
							{initials($user.name)}
						</div>
						<div class="flex-1 min-w-0">
							<p class="text-xs font-medium text-white/80 truncate">{$user.name}</p>
							<p class="text-xs text-white/25 truncate">{$user.email}</p>
						</div>
						<button on:click={handleSignOut} title="Sign out" class="text-white/25 hover:text-white/60 transition-colors p-1">
							<svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/><polyline points="16 17 21 12 16 7"/><line x1="21" y1="12" x2="9" y2="12"/></svg>
						</button>
					</div>
					<div class="flex items-center justify-between px-2 py-2 rounded-lg bg-white/[0.02] border border-white/[0.04]">
						<div class="flex items-center gap-2">
							<div class="w-2 h-2 rounded-full bg-emerald-500"></div>
							<span class="text-[11px] text-white/40">mach1ctl v1.0.0</span>
						</div>
						<button on:click={() => isConsoleExpanded.set(true)} class="text-white/25 hover:text-white/60 transition-colors p-1" title="Open terminal">
							<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="4 17 10 11 4 5"/><line x1="12" y1="19" x2="20" y2="19"/></svg>
						</button>
					</div>
				</div>
			{/if}
		</nav>

		<div class="flex flex-col flex-1 min-w-0 h-screen">
			<main class="flex-1 overflow-y-auto">
				<slot />
			</main>
			
			<!-- Console Panel (VS Code-style) -->
			<div class="flex flex-col border-t border-white/[0.06] bg-[#0d0d12] transition-all duration-300 ease-in-out {$isConsoleExpanded ? 'h-56' : 'h-7'} shrink-0 z-50">
				<!-- Tab Bar -->
				<div class="flex items-center h-7 text-[11px] font-sans bg-[#0a0a0f] border-b border-white/[0.04] select-none shrink-0">
					<button type="button" class="flex items-center gap-1.5 px-3 h-full transition-colors {$consoleTab === 'output' && $isConsoleExpanded ? 'text-white/80 border-b border-orange-500' : 'text-white/30 hover:text-white/50'}" on:click={() => { consoleTab.set('output'); isConsoleExpanded.set(true); }}>
						Output
						{#if logs.filter(l => l.type === 'error').length > 0}
							<span class="ml-1 px-1 rounded bg-red-900/40 text-red-400 text-[9px] leading-tight">{logs.filter(l => l.type === 'error').length}</span>
						{/if}
					</button>
					<button type="button" class="flex items-center gap-1.5 px-3 h-full transition-colors {$consoleTab === 'problems' && $isConsoleExpanded ? 'text-white/80 border-b border-orange-500' : 'text-white/30 hover:text-white/50'}" on:click={() => { consoleTab.set('problems'); isConsoleExpanded.set(true); }}>
						Problems
					</button>
					<button type="button" class="flex items-center gap-1.5 px-3 h-full transition-colors {$consoleTab === 'debug' && $isConsoleExpanded ? 'text-white/80 border-b border-orange-500' : 'text-white/30 hover:text-white/50'}" on:click={() => { consoleTab.set('debug'); isConsoleExpanded.set(true); }}>
						Debug
					</button>
					<button type="button" class="flex items-center gap-1.5 px-3 h-full transition-colors {$consoleTab === 'cli' && $isConsoleExpanded ? 'text-white/80 border-b border-orange-500' : 'text-white/30 hover:text-white/50'}" on:click={() => { consoleTab.set('cli'); isConsoleExpanded.set(true); }}>
						CLI
					</button>

					<div class="flex-1"></div>

					{#if $isConsoleExpanded}
						<input type="text" bind:value={filterText} placeholder="Filter…" class="h-5 w-32 mr-2 px-1.5 text-[10px] bg-white/[0.04] border border-white/[0.06] rounded text-white/60 placeholder-white/20 focus:outline-none focus:border-orange-500/40" />
						<button type="button" class="text-white/30 hover:text-white/60 px-1.5 transition-colors" on:click={() => logs = []} title="Clear">
							<svg class="w-3 h-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>
						</button>
					{/if}
					<button type="button" class="text-white/30 hover:text-white/60 px-1.5 transition-colors" on:click={() => isConsoleExpanded.update(v => !v)} title={$isConsoleExpanded ? 'Minimize' : 'Expand'}>
						<svg class="w-3 h-3 transition-transform {$isConsoleExpanded ? 'rotate-180' : ''}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><polyline points="18 15 12 9 6 15"/></svg>
					</button>
				</div>
				
				<!-- Panel Content -->
				{#if $isConsoleExpanded}
					<div class="flex-1 overflow-y-auto font-mono text-[11px] leading-[18px] bg-[#050508] relative" bind:this={consoleContainer}>
						{#if $consoleTab === 'output'}
							{#if filteredLogs.length === 0}
								<div class="flex items-center justify-center h-full text-white/15 text-xs select-none">No output</div>
							{:else}
								<table class="w-full">
									<tbody>
										{#each filteredLogs as log, i}
											<tr class="hover:bg-white/[0.02] transition-colors">
												<td class="text-white/20 px-2 py-px whitespace-nowrap align-top w-[90px] tabular-nums">{log.time}</td>
												<td class="px-1 py-px whitespace-nowrap align-top w-8 {log.type === 'error' ? 'text-red-400' : log.type === 'warn' ? 'text-amber-400' : log.type === 'success' ? 'text-emerald-400' : 'text-blue-400/60'}">{log.level}</td>
												<td class="text-violet-400/50 px-1 py-px whitespace-nowrap align-top w-[100px] truncate">{log.source}</td>
												<td class="px-2 py-px {log.type === 'error' ? 'text-red-300' : 'text-white/70'}">{log.message}</td>
											</tr>
										{/each}
									</tbody>
								</table>
							{/if}
						{:else if $consoleTab === 'problems'}
							<div class="flex items-center justify-center h-full text-white/15 text-xs select-none">No problems detected</div>
						{:else if $consoleTab === 'cli'}
							<div class="flex flex-col h-full">
								<div class="flex-1 overflow-y-auto p-2 space-y-1">
									{#each cliHistory as entry}
										<div class="text-white/40 font-mono text-[11px]">&gt; {entry.cmd}</div>
										{#if entry.out}
											<div class="text-emerald-400/80 pl-3 font-mono text-[11px]">{entry.out}</div>
										{/if}
										{#if entry.err}
											<div class="text-red-400/80 pl-3 font-mono text-[11px]">{entry.err}</div>
										{/if}
									{/each}
									{#if cliLoading}
										<div class="text-white/20 pl-3 font-mono text-[11px]">Executing...</div>
									{/if}
								</div>
								<div class="flex items-center gap-2 px-2 py-1.5 border-t border-white/[0.04] bg-[#0a0a0f]">
									<span class="text-orange-400 text-xs font-mono">&gt;</span>
									<input
										type="text"
										bind:value={cliCommand}
										on:keydown={handleCliCommand}
										placeholder="Type a command..."
										disabled={cliLoading}
										class="flex-1 bg-transparent text-xs text-white/70 font-mono placeholder-white/20 focus:outline-none"
									/>
								</div>
							</div>
						{:else}
							<div class="flex items-center justify-center h-full text-white/15 text-xs select-none">Debug console — attach via Tauri DevTools</div>
						{/if}
					</div>
				{/if}
			</div>
		</div>
	</div>
	{#if isTauri}<UpdateBanner />{/if}
	<Toast />
	<ZoomManager />
{/if}
