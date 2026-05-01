<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { routerStatus, getRouterStatus } from '$lib/auth';

	let intervalId: ReturnType<typeof setInterval> | null = null;
	let installTab = 'bash';

	onMount(async () => {
		await getRouterStatus();
		intervalId = setInterval(() => {
			getRouterStatus().catch(() => {});
		}, 5000);
		if (navigator.userAgent.includes('Windows')) {
			installTab = 'pwsh';
		}
	});

	onDestroy(() => {
		if (intervalId) clearInterval(intervalId);
	});
	function copyCmd(cmd: string) {
		navigator.clipboard.writeText(cmd);
	}

	$: statusDot = $routerStatus?.status === 'running'
		? 'bg-emerald-500'
		: $routerStatus?.status === 'stopped'
			? 'bg-red-500'
			: 'bg-amber-500';

	$: statusText = $routerStatus?.status === 'running'
		? 'Running'
		: $routerStatus?.status === 'stopped'
			? 'Stopped'
			: 'Unknown';

	$: installCmd = installTab === 'bash' ? "curl -fsSL https://install.1mcp.in | sh"
		: installTab === 'pwsh' ? "irm https://install.1mcp.in/windows | iex"
		: installTab === 'brew' ? "brew install SaiAvinashPatoju/tap/1mcp"
		: "winget install 1mcp.1mcp";
</script>

<div class="hidden lg:flex w-1/2 flex-col justify-between p-12 relative z-10 bg-[#08080c]">
	<!-- Ambient glow -->
	<div class="absolute top-0 left-0 w-full h-full pointer-events-none">
		<div class="absolute top-[15%] left-[10%] w-[400px] h-[400px] bg-orange-600/[0.04] rounded-full blur-[120px]"></div>
		<div class="absolute bottom-[10%] right-[5%] w-[300px] h-[300px] bg-orange-500/[0.03] rounded-full blur-[100px]"></div>
	</div>

	<!-- Top section -->
	<div class="relative">
		<!-- Logo with orbital rings -->
		<div class="relative w-20 h-20 flex items-center justify-center mb-8">
			<div class="absolute inset-0 rounded-full border border-orange-500/10 animate-[spin_10s_linear_infinite]"></div>
			<div class="absolute -inset-4 rounded-full border border-white/[0.03] animate-[spin_16s_linear_infinite_reverse]"></div>
			<img src="/1mcp.png" alt="1mcp" class="w-16 h-16" />
		</div>

		<h1 class="text-4xl font-bold text-white mb-1">1mcp.in</h1>
		<p class="text-orange-500 text-lg font-medium mb-1">The Unified MCP Router</p>
		<p class="text-white/40 text-sm mb-10">Connect your tools. Route with mach1.</p>

		<!-- Feature list -->
		<div class="space-y-5">
			<div class="flex items-start gap-3">
				<div class="w-8 h-8 rounded-lg bg-white/[0.03] border border-white/[0.06] flex items-center justify-center flex-shrink-0 mt-0.5">
					<svg class="w-4 h-4 text-orange-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><path d="M2 12h20"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg>
				</div>
				<div>
					<p class="text-sm font-medium text-white/80">Unified Routing</p>
					<p class="text-xs text-white/35 mt-0.5">Route all MCP calls through a single, secure process.</p>
				</div>
			</div>
			<div class="flex items-start gap-3">
				<div class="w-8 h-8 rounded-lg bg-white/[0.03] border border-white/[0.06] flex items-center justify-center flex-shrink-0 mt-0.5">
					<svg class="w-4 h-4 text-orange-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
				</div>
				<div>
					<p class="text-sm font-medium text-white/80">Secure & Local</p>
					<p class="text-xs text-white/35 mt-0.5">Your data stays local. You are in control.</p>
				</div>
			</div>
			<div class="flex items-start gap-3">
				<div class="w-8 h-8 rounded-lg bg-white/[0.03] border border-white/[0.06] flex items-center justify-center flex-shrink-0 mt-0.5">
					<svg class="w-4 h-4 text-orange-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 2L2 7l10 5 10-5-10-5z"/><path d="M2 17l10 5 10-5"/><path d="M2 12l10 5 10-5"/></svg>
				</div>
				<div>
					<p class="text-sm font-medium text-white/80">Extensible</p>
					<p class="text-xs text-white/35 mt-0.5">Add and manage MCP servers and clients seamlessly.</p>
				</div>
			</div>
		</div>
	</div>

	<!-- Install section -->
	<div class="relative">
		<div class="flex items-center gap-2 mb-2">
			<p class="text-xs text-white/50 font-medium">Quick Install</p>
			<span class="text-[10px] px-1.5 py-0.5 rounded bg-orange-500/10 text-orange-500/70 border border-orange-500/10">v0.3.3</span>
		</div>
		<div class="flex gap-1 mb-2">
			<button class="text-[11px] px-2.5 py-1 rounded {installTab === 'bash' ? 'bg-white/[0.08] text-white/70' : 'text-white/30 hover:text-white/50'}" on:click={() => installTab = 'bash'}>macOS/Linux</button>
			<button class="text-[11px] px-2.5 py-1 rounded {installTab === 'pwsh' ? 'bg-white/[0.08] text-white/70' : 'text-white/30 hover:text-white/50'}" on:click={() => installTab = 'pwsh'}>Windows</button>
			<button class="text-[11px] px-2.5 py-1 rounded {installTab === 'brew' ? 'bg-white/[0.08] text-white/70' : 'text-white/30 hover:text-white/50'}" on:click={() => installTab = 'brew'}>Homebrew</button>
			<button class="text-[11px] px-2.5 py-1 rounded {installTab === 'winget' ? 'bg-white/[0.08] text-white/70' : 'text-white/30 hover:text-white/50'}" on:click={() => installTab = 'winget'}>Winget</button>
		</div>
		<div class="flex items-center justify-between px-3 py-2 rounded-lg bg-black/40 border border-white/[0.06]">
			<code class="text-[11px] text-white/60">{installCmd}</code>
			<button class="text-[11px] text-orange-500/70 hover:text-orange-400 shrink-0 ml-2" on:click={() => copyCmd(installCmd)}>Copy</button>
		</div>
	</div>

	<!-- Bottom section -->
	<div class="relative space-y-6">
		<p class="text-xs text-white/25 leading-relaxed">
			By continuing, you agree to the <a href="/terms" class="text-orange-500/70 hover:text-orange-400 transition-colors">Terms of Service</a> and <a href="https://1mcp.in/privacy" target="_blank" rel="noopener" class="text-orange-500/70 hover:text-orange-400 transition-colors">Privacy Policy</a>.
		</p>

		<!-- Status bar -->
		<div class="flex items-center justify-between px-4 py-3 rounded-xl bg-white/[0.02] border border-white/[0.04]">
			<div class="flex items-center gap-3">
				<div class="flex items-center gap-2">
					<div class="w-2 h-2 rounded-full {statusDot}"></div>
					<span class="text-xs text-white/50">mach1 Router</span>
				</div>
				<span class="text-[11px] text-white/25">{$routerStatus?.version ?? 'v1.0.0'}</span>
				<span class="text-[11px] text-white/25">{statusText}</span>
			</div>
			<div class="flex items-center gap-3">
				<div class="flex items-center gap-1.5">
					<svg class="w-3 h-3 text-white/20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
					<span class="text-[11px] text-white/25">Local Mode</span>
				</div>
				<a href="https://1mcp.in/help" target="_blank" rel="noopener" class="text-[11px] text-white/25 hover:text-white/50 transition-colors">Need help?</a>
			</div>
		</div>
	</div>
</div>
