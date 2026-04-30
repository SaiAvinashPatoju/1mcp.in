<script lang="ts">
	import { goto } from '$app/navigation';
	import { signIn, signUp, authLoading, isAuthenticated } from '$lib/auth';
	import { toast } from '$lib/toast';
	import { onMount } from 'svelte';

	let mode: 'signin' | 'signup' = 'signin';
	let name = '';
	let email = '';
	let password = '';
	let showPassword = false;
	let error = '';
	let remember = true;
	const authModes = ['signin', 'signup'] as const;

	onMount(() => {
		const unsub = isAuthenticated.subscribe((v) => {
			if (v) goto('/dashboard');
		});
		return unsub;
	});

	async function handleSubmit() {
		error = '';
		try {
			if (mode === 'signup') {
				if (!name.trim()) { error = 'Name is required'; return; }
				await signUp(name.trim(), email, password);
			} else {
				await signIn(email, password, remember);
			}
		} catch (e: any) {
			error = e?.message ?? 'Something went wrong';
		}
	}

	// Floating icon positions (percentages) for the decorative background
	const floatingIcons = [
		{ x: 8, y: 18, icon: 'vscode', label: 'VS Code' },
		{ x: 4, y: 42, icon: 'claude', label: 'Claude' },
		{ x: 6, y: 65, icon: 'cursor', label: 'Cursor' },
		{ x: 10, y: 85, icon: 'windsurf', label: 'Windsurf' },
		{ x: 88, y: 22, icon: 'github', label: 'GitHub' },
		{ x: 92, y: 40, icon: 'notion', label: 'Notion' },
		{ x: 90, y: 58, icon: 'openai', label: 'OpenAI' },
		{ x: 86, y: 78, icon: 'vscode2', label: 'VS Code' },
	];
</script>

<svelte:head>
	<title>1mcp.in — One router. Every AI client.</title>
</svelte:head>

<div class="min-h-screen bg-[#08080c] flex relative overflow-hidden">
	<!-- Ambient background glows -->
	<div class="absolute top-0 left-0 w-full h-full pointer-events-none">
		<div class="absolute top-[20%] left-[10%] w-[500px] h-[500px] bg-orange-600/[0.04] rounded-full blur-[120px]"></div>
		<div class="absolute bottom-[10%] right-[5%] w-[400px] h-[400px] bg-orange-500/[0.03] rounded-full blur-[100px]"></div>
	</div>

	<!-- LEFT HERO SECTION -->
	<div class="hidden lg:flex w-1/2 flex-col justify-between p-12 relative z-10">
		<!-- Top: Logo -->
		<div class="flex items-center gap-2">
			<div class="flex items-center justify-center text-orange-500">
				<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
					<polyline points="16 18 22 12 16 6"></polyline>
					<polyline points="8 6 2 12 8 18"></polyline>
				</svg>
			</div>
			<span class="text-lg font-bold text-white/90 tracking-tight">1mcp.in</span>
		</div>

		<!-- Middle: Headline + Description -->
		<div class="max-w-md">
			<!-- Public Beta Badge -->
			<div class="inline-flex items-center gap-2 px-3 py-1.5 rounded-full bg-orange-500/10 border border-orange-500/20 mb-8">
				<span class="w-1.5 h-1.5 rounded-full bg-orange-500 animate-pulse"></span>
				<span class="text-xs font-medium text-orange-400/90">Now in Public Beta</span>
			</div>

			<h1 class="text-5xl font-bold text-white leading-[1.1] tracking-tight mb-2">
				One <span class="text-orange-500">router.</span>
			</h1>
			<h1 class="text-5xl font-bold leading-[1.1] tracking-tight mb-6">
				<span class="text-white">Every</span> <span class="text-orange-500">AI client.</span>
			</h1>

			<p class="text-sm text-white/40 leading-relaxed mb-8 max-w-sm">
				1mcp replaces every manual MCP setup with a single local router. Install once. Use from VS Code, Cursor, Claude, Codex, and more.
			</p>

			<div class="flex items-center gap-3">
				<button on:click={() => window.open('https://1mcp.in/download', '_blank')} class="inline-flex items-center gap-2 px-5 py-2.5 rounded-lg bg-orange-500 hover:bg-orange-600 text-white text-sm font-semibold transition-all shadow-lg shadow-orange-500/20">
					<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
					Install now
				</button>
				<a href="https://github.com/SaiAvinashPatoju/1mcp.in" target="_blank" rel="noopener" class="inline-flex items-center gap-2 px-5 py-2.5 rounded-lg bg-white/[0.03] border border-white/[0.08] hover:bg-white/[0.06] hover:border-white/[0.12] text-white/70 hover:text-white text-sm font-medium transition-all">
					View on GitHub
					<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><line x1="5" y1="12" x2="19" y2="12"/><polyline points="12 5 19 12 12 19"/></svg>
				</a>
			</div>
		</div>

		<!-- Bottom: Spacer for balance -->
		<div></div>

		<!-- Decorative dotted lines connecting floating icons -->
		<svg class="absolute inset-0 w-full h-full pointer-events-none" style="z-index: -1;">
			<defs>
				<pattern id="dotPattern" width="8" height="8" patternUnits="userSpaceOnUse">
					<circle cx="1" cy="1" r="0.5" fill="rgba(255,255,255,0.03)" />
				</pattern>
			</defs>
			<!-- Grid background -->
			<rect width="100%" height="100%" fill="url(#dotPattern)" />
			<!-- Connection lines -->
			<line x1="15%" y1="25%" x2="85%" y2="30%" stroke="rgba(249,115,22,0.08)" stroke-width="1" stroke-dasharray="4 4" />
			<line x1="10%" y1="50%" x2="88%" y2="45%" stroke="rgba(249,115,22,0.06)" stroke-width="1" stroke-dasharray="4 4" />
			<line x1="12%" y1="75%" x2="90%" y2="70%" stroke="rgba(249,115,22,0.08)" stroke-width="1" stroke-dasharray="4 4" />
		</svg>

		<!-- Floating AI client icons -->
		{#each floatingIcons as item}
			<div
				class="absolute flex items-center justify-center w-10 h-10 rounded-xl bg-white/[0.03] border border-white/[0.06] backdrop-blur-sm"
				style="left: {item.x}%; top: {item.y}%;"
			>
				{#if item.icon === 'vscode'}
					<svg class="w-5 h-5 text-blue-400" viewBox="0 0 24 24" fill="currentColor"><path d="M17.583.063a1.5 1.5 0 0 1 1.342.893l.063.177 3 9.5a1.5 1.5 0 0 1-.134 1.258l-.1.158-7.5 10.5a1.5 1.5 0 0 1-2.015.334l-.146-.1L7.36 19.06l-3.824 2.87a1.5 1.5 0 0 1-2.33-1.297l.007-.156V3.523a1.5 1.5 0 0 1 2.423-1.184l.097.084 3.824 2.87 3.633-2.68a1.5 1.5 0 0 1 1.023-.353l.18.008.17.024.17.047z"/></svg>
				{:else if item.icon === 'claude'}
					<svg class="w-5 h-5 text-amber-300" viewBox="0 0 24 24" fill="currentColor"><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm0 18c-4.41 0-8-3.59-8-8s3.59-8 8-8 8 3.59 8 8-3.59 8-8 8z"/><circle cx="12" cy="12" r="3"/></svg>
				{:else if item.icon === 'cursor'}
					<svg class="w-5 h-5 text-white" viewBox="0 0 24 24" fill="currentColor"><path d="M5.5 3.21V20.8c0 .45.54.67.85.35l4.86-4.86a.5.5 0 0 1 .35-.15h6.87a.5.5 0 0 0 .35-.85L6.35 2.85a.5.5 0 0 0-.85.36z"/></svg>
				{:else if item.icon === 'windsurf'}
					<svg class="w-5 h-5 text-cyan-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M2 12h4l2-6 4 12 4-8 2 2h4"/></svg>
				{:else if item.icon === 'github'}
					<svg class="w-5 h-5 text-white" viewBox="0 0 24 24" fill="currentColor"><path d="M12 2C6.477 2 2 6.484 2 12.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.531 1.032 1.531 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0 1 12 6.844c.85.004 1.705.115 2.504.337 1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.202 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.309.678.92.678 1.855 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.02 10.02 0 0 0 22 12.017C22 6.484 17.522 2 12 2z"/></svg>
				{:else if item.icon === 'notion'}
					<svg class="w-5 h-5 text-white" viewBox="0 0 24 24" fill="currentColor"><path d="M4.459 4.208c.746.606 1.026.56 2.428.466l13.215-.793c.28 0 .047-.28-.046-.326L17.86 2.02c-.42-.326-.98-.7-2.055-.607L3.01 2.745c-.466.047-.56.28-.374.466zm.793 3.08v13.904c0 .747.373 1.027 1.214.98l14.523-.84c.841-.046.935-.56.935-1.167V6.354c0-.606-.233-.933-.748-.887l-15.177.887c-.56.047-.747.327-.747.934zm14.337.745c.093.42 0 .84-.42.888l-.7.14v10.264c-.608.327-1.168.514-1.635.514-.748 0-.935-.234-1.495-.933l-4.577-7.186v6.952l1.448.327s0 .84-1.168.84l-3.222.186c-.093-.186 0-.653.327-.746l.84-.233V9.854L7.822 9.76c-.094-.42.14-1.026.793-1.073l3.456-.233 4.764 7.279v-6.44l-1.215-.14c-.093-.514.28-.887.747-.933zM1.936 1.035l13.31-.98c1.634-.14 2.055-.047 3.082.7l4.249 2.986c.7.513.934.653.934 1.213v16.378c0 1.026-.373 1.634-1.68 1.726l-15.458.934c-.98.047-1.448-.093-1.962-.747l-3.129-4.06c-.56-.747-.793-1.306-.793-1.96V2.667c0-.84.374-1.493 1.447-1.632z"/></svg>
				{:else if item.icon === 'openai'}
					<svg class="w-5 h-5 text-emerald-400" viewBox="0 0 24 24" fill="currentColor"><path d="M22.282 9.821a5.985 5.985 0 0 0-.516-4.91 6.046 6.046 0 0 0-6.51-2.9A6.065 6.065 0 0 0 4.981 4.18a5.985 5.985 0 0 0-3.998 2.9 6.046 6.046 0 0 0 .743 7.097 5.98 5.98 0 0 0 .51 4.911 6.051 6.051 0 0 0 6.515 2.9A5.985 5.985 0 0 0 13.26 24a6.056 6.056 0 0 0 5.772-4.206 5.99 5.99 0 0 0 3.997-2.9 6.056 6.056 0 0 0-.747-7.073zM13.26 22.43a4.476 4.476 0 0 1-2.876-1.04l.141-.081 4.779-2.758a.795.795 0 0 0 .392-.681v-6.737l2.02 1.168a.071.071 0 0 1 .038.052v5.583a4.504 4.504 0 0 1-4.494 4.494zM3.6 18.304a4.47 4.47 0 0 1-.535-3.014l.142.085 4.783 2.759a.771.771 0 0 0 .78 0l5.843-3.369v2.332a.08.08 0 0 1-.033.062L9.74 19.95a4.5 4.5 0 0 1-6.14-1.646zM2.34 7.896a4.485 4.485 0 0 1 2.366-1.973V11.6a.766.766 0 0 0 .388.676l5.815 3.355-2.02 1.168a.076.076 0 0 1-.071 0l-4.83-2.786A4.504 4.504 0 0 1 2.34 7.896zm16.597 3.855-5.833-3.387L15.119 7.2a.076.076 0 0 1 .071 0l4.83 2.791a4.494 4.494 0 0 1-.676 8.105v-5.678a.79.79 0 0 0-.407-.667zm2.01-3.023-.141-.085-4.774-2.782a.776.776 0 0 0-.785 0L9.409 9.23V6.897a.066.066 0 0 1 .028-.061l4.83-2.787a4.5 4.5 0 0 1 6.68 4.66zm-12.64 4.135-2.02-1.164a.08.08 0 0 1-.038-.057V6.075a4.5 4.5 0 0 1 7.375-3.453l-.142.08L8.704 5.46a.795.795 0 0 0-.393.681zm1.097-2.365 2.602-1.5 2.607 1.5v2.999l-2.597 1.5-2.607-1.5z"/></svg>
				{:else}
					<svg class="w-5 h-5 text-blue-400" viewBox="0 0 24 24" fill="currentColor"><path d="M17.583.063a1.5 1.5 0 0 1 1.342.893l.063.177 3 9.5a1.5 1.5 0 0 1-.134 1.258l-.1.158-7.5 10.5a1.5 1.5 0 0 1-2.015.334l-.146-.1L7.36 19.06l-3.824 2.87a1.5 1.5 0 0 1-2.33-1.297l.007-.156V3.523a1.5 1.5 0 0 1 2.423-1.184l.097.084 3.824 2.87 3.633-2.68a1.5 1.5 0 0 1 1.023-.353l.18.008.17.024.17.047z"/></svg>
				{/if}
			</div>
		{/each}

		<!-- Center glowing dot where lines converge -->
		<div class="absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 pointer-events-none">
			<div class="w-3 h-3 rounded-full bg-orange-500 shadow-[0_0_20px_rgba(249,115,22,0.5)]"></div>
		</div>
	</div>

	<!-- RIGHT LOGIN SECTION -->
	<div class="w-full lg:w-1/2 flex items-center justify-center p-6 relative z-10">
		<div class="w-full max-w-[420px]">
			<!-- Login Card -->
			<div class="bg-[#0f0f14]/80 border border-white/[0.06] rounded-2xl p-8 shadow-2xl backdrop-blur-xl">
				<!-- Logo + Title -->
				<div class="flex flex-col items-center gap-3 mb-8">
					<div class="w-14 h-14 rounded-2xl bg-orange-500 flex items-center justify-center shadow-lg shadow-orange-500/30">
						<span class="text-xl font-black text-white">1M</span>
					</div>
					<div class="text-center">
						<h2 class="text-xl font-bold text-white/95">Welcome back</h2>
						<p class="text-sm text-white/35 mt-1">Sign in to access your MCP gateway</p>
					</div>
				</div>

				<!-- Sign In / Sign Up Toggle -->
				<div class="flex bg-black/40 rounded-lg p-1 mb-6 border border-white/[0.04]">
					{#each authModes as m}
						<button
							on:click={() => { mode = m; error = ''; }}
							class="flex-1 py-2 text-sm rounded-md transition-all font-medium
								{mode === m ? 'bg-orange-500 text-white shadow-sm' : 'text-white/30 hover:text-white/60'}"
						>
							{m === 'signin' ? 'Sign In' : 'Sign Up'}
						</button>
					{/each}
				</div>

				<!-- Form -->
				<form on:submit|preventDefault={handleSubmit} class="space-y-4">
					{#if mode === 'signup'}
						<div>
							<label class="block text-xs font-medium text-white/40 mb-1.5" for="name-input">Name</label>
							<input
								id="name-input"
								type="text"
								bind:value={name}
								placeholder="Your full name"
								class="w-full bg-black/40 border border-white/[0.06] rounded-lg px-3.5 py-2.5 text-sm text-white/80 placeholder-white/15 focus:outline-none focus:border-orange-500/50 focus:ring-1 focus:ring-orange-500/20 transition-colors"
							/>
						</div>
					{/if}

					<div>
						<label class="block text-xs font-medium text-white/40 mb-1.5" for="email-input">Email</label>
						<input
							id="email-input"
							type="email"
							bind:value={email}
							placeholder="you@example.com"
							required
							class="w-full bg-black/40 border border-white/[0.06] rounded-lg px-3.5 py-2.5 text-sm text-white/80 placeholder-white/15 focus:outline-none focus:border-orange-500/50 focus:ring-1 focus:ring-orange-500/20 transition-colors"
						/>
					</div>

					<div>
						<label class="block text-xs font-medium text-white/40 mb-1.5" for="password-input">Password</label>
						<div class="relative">
							<input
								id="password-input"
								type={showPassword ? 'text' : 'password'}
								bind:value={password}
								placeholder="••••••••"
								required
								minlength="8"
								class="w-full bg-black/40 border border-white/[0.06] rounded-lg px-3.5 py-2.5 text-sm text-white/80 placeholder-white/15 focus:outline-none focus:border-orange-500/50 focus:ring-1 focus:ring-orange-500/20 transition-colors pr-10"
							/>
							<button
								type="button"
								on:click={() => (showPassword = !showPassword)}
								class="absolute right-3 top-1/2 -translate-y-1/2 text-white/20 hover:text-white/60 transition-colors"
							>
								{#if showPassword}
									<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M9.88 9.88a3 3 0 1 0 4.24 4.24"/><path d="M10.73 5.08A10.43 10.43 0 0 1 12 5c7 0 10 7 10 7a13.16 13.16 0 0 1-1.67 2.68"/><path d="M6.61 6.61A13.526 13.526 0 0 0 2 12s3 7 10 7a9.74 9.74 0 0 0 5.39-1.61"/><line x1="2" y1="2" x2="22" y2="22"/></svg>
								{:else}
									<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M2 12s3-7 10-7 10 7 10 7-3 7-10 7-10-7-10-7Z"/><circle cx="12" cy="12" r="3"/></svg>
								{/if}
							</button>
						</div>
					</div>

					{#if mode === 'signin'}
						<div class="flex items-center justify-between">
							<label class="flex items-center gap-2 cursor-pointer select-none">
								<input type="checkbox" bind:checked={remember} class="w-3.5 h-3.5 rounded border border-white/[0.1] bg-black/40 accent-orange-500" />
								<span class="text-xs text-white/40">Remember me</span>
							</label>
							<a href="/forgot-password" class="text-xs text-orange-500/80 hover:text-orange-400 transition-colors">Forgot password?</a>
						</div>
					{/if}

					{#if error}
						<p class="text-xs text-red-400 bg-red-900/10 border border-red-900/40 rounded-lg px-3 py-2">{error}</p>
					{/if}

					<button
						type="submit"
						disabled={$authLoading}
						class="w-full py-2.5 rounded-lg bg-orange-500 hover:bg-orange-600 text-white text-sm font-semibold transition-all disabled:opacity-60 disabled:cursor-not-allowed shadow-lg shadow-orange-500/20 hover:shadow-orange-500/30 mt-2"
					>
						{#if $authLoading}
							<span class="flex items-center justify-center gap-2">
								<span class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin"></span>
								{mode === 'signin' ? 'Signing in…' : 'Creating account…'}
							</span>
						{:else}
							{mode === 'signin' ? 'Sign In' : 'Create Account'}
						{/if}
					</button>
				</form>

				<!-- Divider -->
				<div class="relative my-6">
					<div class="absolute inset-0 flex items-center">
						<div class="w-full border-t border-white/[0.06]"></div>
					</div>
					<div class="relative flex justify-center text-xs">
						<span class="px-3 bg-[#0f0f14] text-white/25">or continue with</span>
					</div>
				</div>

				<!-- Social Login Buttons -->
				<div class="grid grid-cols-3 gap-3">
					<button on:click={() => toast.info('GitHub login coming soon')} aria-label="Sign in with GitHub" class="flex items-center justify-center gap-2 py-2.5 rounded-lg bg-white/[0.03] border border-white/[0.06] hover:bg-white/[0.06] hover:border-white/[0.1] transition-all group">
						<svg class="w-5 h-5 text-white/60 group-hover:text-white transition-colors" viewBox="0 0 24 24" fill="currentColor"><path d="M12 2C6.477 2 2 6.484 2 12.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.531 1.032 1.531 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0 1 12 6.844c.85.004 1.705.115 2.504.337 1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.202 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.309.678.92.678 1.855 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.02 10.02 0 0 0 22 12.017C22 6.484 17.522 2 12 2z"/></svg>
					</button>
					<button on:click={() => toast.info('Google login coming soon')} aria-label="Sign in with Google" class="flex items-center justify-center gap-2 py-2.5 rounded-lg bg-white/[0.03] border border-white/[0.06] hover:bg-white/[0.06] hover:border-white/[0.1] transition-all group">
						<svg class="w-5 h-5" viewBox="0 0 24 24" fill="currentColor"><path d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92a5.06 5.06 0 0 1-2.2 3.32v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.1z" fill="#4285F4"/><path d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z" fill="#34A853"/><path d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z" fill="#FBBC05"/><path d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z" fill="#EA4335"/></svg>
					</button>
					<button on:click={() => toast.info('Discord login coming soon')} aria-label="Sign in with Discord" class="flex items-center justify-center gap-2 py-2.5 rounded-lg bg-white/[0.03] border border-white/[0.06] hover:bg-white/[0.06] hover:border-white/[0.1] transition-all group">
						<svg class="w-5 h-5 text-[#5865F2] group-hover:brightness-110 transition-all" viewBox="0 0 24 24" fill="currentColor"><path d="M20.317 4.37a19.791 19.791 0 0 0-4.885-1.515.074.074 0 0 0-.079.037c-.21.375-.444.864-.608 1.25a18.27 18.27 0 0 0-5.487 0 12.64 12.64 0 0 0-.617-1.25.077.077 0 0 0-.079-.037A19.736 19.736 0 0 0 3.677 4.37a.07.07 0 0 0-.032.027C.533 9.046-.32 13.58.099 18.057a.082.082 0 0 0 .031.057 19.9 19.9 0 0 0 5.993 3.03.078.078 0 0 0 .084-.028c.462-.63.874-1.295 1.226-1.994a.076.076 0 0 0-.041-.106 13.107 13.107 0 0 1-1.872-.892.077.077 0 0 1-.008-.128 10.2 10.2 0 0 0 .372-.292.074.074 0 0 1 .077-.01c3.928 1.793 8.18 1.793 12.062 0a.074.074 0 0 1 .078.01c.12.098.246.198.373.292a.077.077 0 0 1-.006.127 12.299 12.299 0 0 1-1.873.892.077.077 0 0 0-.041.107c.36.698.772 1.362 1.225 1.993a.076.076 0 0 0 .084.028 19.839 19.839 0 0 0 6.002-3.03.077.077 0 0 0 .032-.054c.5-5.177-.838-9.674-3.549-13.66a.061.061 0 0 0-.031-.03zM8.02 15.33c-1.183 0-2.157-1.085-2.157-2.419 0-1.333.956-2.419 2.157-2.419 1.21 0 2.176 1.096 2.157 2.42 0 1.333-.956 2.418-2.157 2.418zm7.975 0c-1.183 0-2.157-1.085-2.157-2.419 0-1.333.955-2.419 2.157-2.419 1.21 0 2.176 1.096 2.157 2.42 0 1.333-.946 2.418-2.157 2.418z"/></svg>
					</button>
				</div>
			</div>

			<!-- Terms footer -->
			<p class="text-center text-xs text-white/20 mt-5">
				By continuing, you agree to the 1mcp.in <a href="/terms" class="text-orange-500/70 hover:text-orange-400 transition-colors">Terms of Service</a>.
			</p>
		</div>
	</div>
</div>

<style>
	/* Smooth animation for the beta dot */
	@keyframes pulse {
		0%, 100% { opacity: 1; }
		50% { opacity: 0.5; }
	}
	.animate-pulse {
		animation: pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite;
	}
</style>
