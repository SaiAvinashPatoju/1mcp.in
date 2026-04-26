<script lang="ts">
	import '../app.css';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { user, signOut, isAuthenticated } from '$lib/auth';
	import UpdateBanner from '$lib/components/UpdateBanner.svelte';
	import { browser } from '$app/environment';
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

		<main class="flex-1 overflow-y-auto">
			<slot />
		</main>
	</div>
	{#if isTauri}<UpdateBanner />{/if}
{/if}
