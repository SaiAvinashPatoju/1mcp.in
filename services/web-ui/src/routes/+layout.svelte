<script lang="ts">
	import '../app.css';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { user, signOut, isAuthenticated } from '$lib/auth';
	import UpdateBanner from '$lib/components/UpdateBanner.svelte';
	import { browser } from '$app/environment';
	import { onMount, afterUpdate, onDestroy } from 'svelte';
	import { listen } from '@tauri-apps/api/event';
	import { startUserCounter, fetchMarketplace } from '$lib/stores';
	
	const isTauri = browser && '__TAURI_INTERNALS__' in window;

	const NAV = [
		{ href: '/dashboard', label: 'Dashboard', icon: '📊' },
		{ href: '/marketplace', label: 'Marketplace', icon: '🏪' },
		{ href: '/clients', label: 'Clients', icon: '🔌' },
		{ href: '/account', label: 'Account', icon: '👤' }
	];

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
	let isConsoleExpanded = false;
	let consoleContainer: HTMLElement;
	let consoleTab: 'output' | 'problems' | 'debug' = 'output';
	let logs: { time: string; level: string; source: string; message: string; type: 'info'|'warn'|'error'|'success' }[] = [];
	let filterText = '';
	let tauriUnlisten: (() => void) | null = null;

	function pushLog(source: string, message: string, type: 'info'|'warn'|'error'|'success' = 'info') {
		const time = new Date().toISOString().split('T')[1].slice(0, 12);
		const level = type === 'error' ? 'ERR' : type === 'warn' ? 'WRN' : type === 'success' ? 'OK' : 'INF';
		logs = [...logs.slice(-199), { time, level, source, message, type }];
	}

	onMount(async () => {
		pushLog('system', 'mach1 initialized - waiting for events...', 'success');

		// Kick off background data syncs
		startUserCounter();
		fetchMarketplace();

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

{#if !$isAuthenticated}
	<slot />
{:else}
	<div class="flex h-screen bg-[#0a0a0f] overflow-hidden">
		<!-- Sidebar -->
		<nav class="w-52 flex flex-col border-r border-white/[0.04] bg-[#0a0a0f] flex-shrink-0">
			<div class="flex items-center gap-2.5 px-4 py-5 border-b border-white/[0.04]">
                                <div class="w-7 h-7 rounded-lg bg-gradient-to-br from-violet-600 to-violet-800 flex items-center justify-center flex-shrink-0 text-[10px] font-black text-white">1M</div>
                                <span class="text-sm font-bold text-white/90">1mcp.in</span>
			</div>

			<div class="flex-1 px-3 py-4 space-y-0.5">
				{#each NAV as { href, label, icon }}
					<a
						{href}
						class="flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm transition-all
							{currentPath === href || currentPath.startsWith(href + '/')
								? 'bg-violet-600/15 text-violet-400 font-medium border border-violet-600/20'
								: 'text-white/40 hover:text-white/70 hover:bg-white/[0.03] border border-transparent'}"
					>
						<span class="text-sm">{icon}</span>
						{label}
					</a>
				{/each}
			</div>

			{#if $user}
				<div class="px-3 py-4 border-t border-white/[0.04]">
					<div class="flex items-center gap-2.5 px-2 py-1.5">
						<div class="w-7 h-7 rounded-full bg-violet-900/40 flex items-center justify-center text-xs font-bold text-violet-400 flex-shrink-0 select-none">
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
				</div>
			{/if}
		</nav>

		<div class="flex flex-col flex-1 min-w-0 h-screen">
			<main class="flex-1 overflow-y-auto">
				<slot />
			</main>
			
			<!-- Console Panel (VS Code-style) -->
			<div class="flex flex-col border-t border-white/[0.06] bg-[#0d0d12] transition-all duration-300 ease-in-out {isConsoleExpanded ? 'h-56' : 'h-7'} shrink-0 z-50">
				<!-- Tab Bar -->
				<div class="flex items-center h-7 text-[11px] font-sans bg-[#0a0a0f] border-b border-white/[0.04] select-none shrink-0">
					<button type="button" class="flex items-center gap-1.5 px-3 h-full transition-colors {consoleTab === 'output' && isConsoleExpanded ? 'text-white/80 border-b border-violet-500' : 'text-white/30 hover:text-white/50'}" on:click={() => { consoleTab = 'output'; isConsoleExpanded = true; }}>
						Output
						{#if logs.filter(l => l.type === 'error').length > 0}
							<span class="ml-1 px-1 rounded bg-red-900/40 text-red-400 text-[9px] leading-tight">{logs.filter(l => l.type === 'error').length}</span>
						{/if}
					</button>
					<button type="button" class="flex items-center gap-1.5 px-3 h-full transition-colors {consoleTab === 'problems' && isConsoleExpanded ? 'text-white/80 border-b border-violet-500' : 'text-white/30 hover:text-white/50'}" on:click={() => { consoleTab = 'problems'; isConsoleExpanded = true; }}>
						Problems
					</button>
					<button type="button" class="flex items-center gap-1.5 px-3 h-full transition-colors {consoleTab === 'debug' && isConsoleExpanded ? 'text-white/80 border-b border-violet-500' : 'text-white/30 hover:text-white/50'}" on:click={() => { consoleTab = 'debug'; isConsoleExpanded = true; }}>
						Debug
					</button>

					<div class="flex-1"></div>

					{#if isConsoleExpanded}
						<input type="text" bind:value={filterText} placeholder="Filter…" class="h-5 w-32 mr-2 px-1.5 text-[10px] bg-white/[0.04] border border-white/[0.06] rounded text-white/60 placeholder-white/20 focus:outline-none focus:border-violet-500/40" />
						<button type="button" class="text-white/30 hover:text-white/60 px-1.5 transition-colors" on:click={() => logs = []} title="Clear">
							<svg class="w-3 h-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>
						</button>
					{/if}
					<button type="button" class="text-white/30 hover:text-white/60 px-1.5 transition-colors" on:click={() => isConsoleExpanded = !isConsoleExpanded} title={isConsoleExpanded ? 'Minimize' : 'Expand'}>
						<svg class="w-3 h-3 transition-transform {isConsoleExpanded ? 'rotate-180' : ''}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><polyline points="18 15 12 9 6 15"/></svg>
					</button>
				</div>
				
				<!-- Panel Content -->
				{#if isConsoleExpanded}
					<div class="flex-1 overflow-y-auto font-mono text-[11px] leading-[18px] bg-[#050508] relative" bind:this={consoleContainer}>
						{#if consoleTab === 'output'}
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
						{:else if consoleTab === 'problems'}
							<div class="flex items-center justify-center h-full text-white/15 text-xs select-none">No problems detected</div>
						{:else}
							<div class="flex items-center justify-center h-full text-white/15 text-xs select-none">Debug console — attach via Tauri DevTools</div>
						{/if}
					</div>
				{/if}
			</div>
		</div>
	</div>
	{#if isTauri}<UpdateBanner />{/if}
{/if}
