<script lang="ts">
	import '../app.css';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { user, signOut, isAuthenticated } from '$lib/auth';
	import UpdateBanner from '$lib/components/UpdateBanner.svelte';
	import { browser } from '$app/environment';
	import { onMount, afterUpdate } from 'svelte';
	
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

	// Console State & Mock Logs
	let isConsoleExpanded = false;
	let consoleContainer: HTMLElement;
	let logs: { time: string; source: string; message: string; type: 'info'|'warn'|'error'|'success' }[] = [];

	onMount(() => {
		// Mock log stream simulating backend/Tauri activity
		const sources = ['rust/main', 'mcp-server', 'auth-service', 'database', 'system'];
		const mockMessages = [
			{ msg: 'Connected to control plane', type: 'info' },
			{ msg: 'Starting internal server on port 45521...', type: 'info' },
			{ msg: 'Warning: Failed to fetch latest metadata, retrying', type: 'warn' },
			{ msg: 'MCP server "github" initialized successfully', type: 'success' },
			{ msg: 'Syncing registry manifest', type: 'info' },
			{ msg: 'Heartbeat signal acknowledged', type: 'info' },
			{ msg: 'Error: Connection timeout during handshake', type: 'error' },
			{ msg: 'Reconnecting to data store', type: 'info' },
			{ msg: 'Client connection closed', type: 'warn' },
			{ msg: 'User session verified', type: 'success' }
		];

		const pushLog = () => {
			const randomMsg = mockMessages[Math.floor(Math.random() * mockMessages.length)];
			const randomSrc = sources[Math.floor(Math.random() * sources.length)];
			const time = new Date().toISOString().split('T')[1].slice(0, 12);
			
			logs = [...logs, {
				time,
				source: randomSrc,
				message: randomMsg.msg,
				type: randomMsg.type as any
			}];
			
			// Keep only last 100 logs
			if (logs.length > 100) {
				logs = logs.slice(logs.length - 100);
			}

			// Schedule next log
			if (browser) {
				setTimeout(pushLog, 2000 + Math.random() * 5000);
			}
		};

		// Push initial logs
		logs.push({ time: new Date().toISOString().split('T')[1].slice(0, 12), source: 'system', message: 'Application initialized. Waiting for events...', type: 'success' });
		setTimeout(pushLog, 2000);
	});

	afterUpdate(() => {
		// Auto scroll to bottom
		if (consoleContainer && isConsoleExpanded) {
			consoleContainer.scrollTop = consoleContainer.scrollHeight;
		}
	});
</script>

{#if !$isAuthenticated}
	<slot />
{:else}
	<div class="flex h-screen bg-[#0a0a0f] overflow-hidden">
		<!-- Sidebar -->
		<nav class="w-52 flex flex-col border-r border-white/[0.04] bg-[#0a0a0f] flex-shrink-0">
			<div class="flex items-center gap-2.5 px-4 py-5 border-b border-white/[0.04]">
				<div class="w-7 h-7 rounded-lg bg-gradient-to-br from-violet-600 to-violet-800 flex items-center justify-center flex-shrink-0 text-[10px] font-black text-white">M1</div>
				<span class="text-sm font-bold text-white/90">Mach1</span>
				<span class="text-[10px] px-1.5 py-0.5 rounded bg-violet-900/40 text-violet-400 border border-violet-800/40 font-semibold ml-auto">MCP</span>
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
			
			<!-- Embedded Footer Console -->
			<div class="flex flex-col border-t border-white/[0.04] bg-[#0d0d12] transition-all duration-200 {isConsoleExpanded ? 'h-64' : 'h-8'} shrink-0 z-50 shadow-[0_-5px_20px_rgba(0,0,0,0.3)]">
				<!-- Header -->
				<div class="flex items-center justify-between px-3 h-8 text-[11px] font-sans font-semibold text-white/50 hover:text-white/80 transition-colors bg-[#0a0a0f]/80 w-full">
					<button type="button" class="flex items-center gap-2 flex-1 text-left" on:click={() => isConsoleExpanded = !isConsoleExpanded}>
						<svg class="w-3.5 h-3.5 transition-transform duration-200 {isConsoleExpanded ? 'rotate-90' : ''}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="9 18 15 12 9 6"></polyline></svg>
						<span class="uppercase tracking-wider">Console</span>
					</button>
					{#if isConsoleExpanded}
						<button type="button" class="hover:text-white p-0.5 rounded transition-colors" on:click={() => logs = []} title="Clear Console" aria-label="Clear Console">
							<svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="3 6 5 6 21 6"></polyline><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path></svg>
						</button>
					{/if}
				</div>
				
				<!-- Terminal Content -->
				<div class="flex-1 overflow-y-auto p-2 font-mono text-[11px] text-white/70 space-y-0.5 bg-[#050508] relative" bind:this={consoleContainer}>
					{#if logs.length === 0}
						<div class="flex items-center justify-center h-full text-white/20 italic select-none">No logs to display</div>
					{:else}
						{#each logs as log}
							<div class="flex gap-2.5 items-start hover:bg-white/[0.02] px-1 rounded transition-colors group">
								<span class="text-white/30 shrink-0 w-24">[{log.time}]</span>
								<span class="shrink-0 w-24 {log.type === 'error' ? 'text-red-400' : log.type === 'warn' ? 'text-yellow-400' : log.type === 'success' ? 'text-emerald-400' : 'text-blue-400'}">[{log.source}]</span>
								<span class="break-words font-medium {log.type === 'error' ? 'text-red-300' : 'text-white/80'}">{log.message}</span>
							</div>
						{/each}
					{/if}
				</div>
			</div>
		</div>
	</div>
	{#if isTauri}<UpdateBanner />{/if}
{/if}
