<script lang="ts">
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import { signUp, authLoading, isAuthenticated } from '$lib/auth';
	import { toast } from '$lib/toast';
	import AuthLeftPanel from '$lib/components/AuthLeftPanel.svelte';

	let name = '';
	let email = '';
	let password = '';
	let confirmPassword = '';
	let showPassword = false;
	let error = '';

	onMount(() => {
		const unsub = isAuthenticated.subscribe((v) => {
			if (v) goto('/dashboard');
		});
		return unsub;
	});

	async function handleSubmit() {
		error = '';
		if (!name.trim()) {
			error = 'Name is required';
			return;
		}
		if (!email.trim()) {
			error = 'Email is required';
			return;
		}
		if (!password) {
			error = 'Password is required';
			return;
		}
		if (password.length < 8) {
			error = 'Password must be at least 8 characters';
			return;
		}
		if (password !== confirmPassword) {
			error = 'Passwords do not match';
			return;
		}
		try {
			await signUp(name.trim(), email.trim(), password);
			goto('/dashboard');
		} catch (e: any) {
			error = e?.message ?? 'Sign up failed';
		}
	}
</script>

<svelte:head>
	<title>1mcp.in — Sign Up</title>
</svelte:head>

<div class="min-h-screen bg-[#08080c] flex relative overflow-hidden">
	<AuthLeftPanel />

	<!-- RIGHT SIGNUP SECTION -->
	<div class="w-full lg:w-1/2 flex items-center justify-center p-6 relative z-10">
		<div class="w-full max-w-[420px]">
			<div class="bg-[#0f0f14]/80 border border-white/[0.06] rounded-2xl p-8 shadow-2xl backdrop-blur-xl">
				<h2 class="text-xl font-bold text-white/95">Create your account</h2>
				<p class="text-sm text-white/35 mt-1">Sign up to get started with 1mcp.in</p>

				<form on:submit|preventDefault={handleSubmit} class="mt-6 space-y-4">
					<div>
						<label class="block text-xs font-medium text-white/40 mb-1.5" for="signup-name">Name</label>
						<div class="relative">
							<svg class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-white/20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>
							<input
								id="signup-name"
								type="text"
								bind:value={name}
								placeholder="Your full name"
								class="w-full bg-black/40 border border-white/[0.06] rounded-lg px-10 py-2.5 text-sm text-white/80 placeholder-white/15 focus:outline-none focus:border-orange-500/50 focus:ring-1 focus:ring-orange-500/20 transition-colors"
							/>
						</div>
					</div>

					<div>
						<label class="block text-xs font-medium text-white/40 mb-1.5" for="signup-email">Email</label>
						<div class="relative">
							<svg class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-white/20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="2" y="4" width="20" height="16" rx="2"/><path d="m22 7-8.97 5.7a1.94 1.94 0 0 1-2.06 0L2 7"/></svg>
							<input
								id="signup-email"
								type="email"
								bind:value={email}
								placeholder="you@example.com"
								required
								class="w-full bg-black/40 border border-white/[0.06] rounded-lg px-10 py-2.5 text-sm text-white/80 placeholder-white/15 focus:outline-none focus:border-orange-500/50 focus:ring-1 focus:ring-orange-500/20 transition-colors"
							/>
						</div>
					</div>

					<div>
						<label class="block text-xs font-medium text-white/40 mb-1.5" for="signup-password">Password</label>
						<div class="relative">
							<svg class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-white/20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
							<input
								id="signup-password"
								type={showPassword ? 'text' : 'password'}
								bind:value={password}
								placeholder="••••••••"
								minlength="8"
								required
								class="w-full bg-black/40 border border-white/[0.06] rounded-lg px-10 py-2.5 text-sm text-white/80 placeholder-white/15 focus:outline-none focus:border-orange-500/50 focus:ring-1 focus:ring-orange-500/20 transition-colors pr-10"
							/>
							<button
								type="button"
								on:click={() => (showPassword = !showPassword)}
								class="absolute right-3 top-1/2 -translate-y-1/2 text-white/20 hover:text-white/60 transition-colors"
							>
								{#if showPassword}
									<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M9.88 9.88a3 3 0 1 0 4.24 4.24"/><path d="M10.73 5.08A10.43 10.43 0 0 1 12 5c7 0 10 7 10 7a13.16 13.16 0 0 1-1.67 2.68"/><path d="M6.61 6.61A13.526 13.526 0 0 0 2 12s3 7 10 7a9.74 9.74 0 0 0 5.39-1.61"/><line x1="2" y1="2" x2="22" y2="22"/></svg>
								{:else}
									<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M2 12s3-7 10-7 10 7 10 7-3 7-10 7-10-7-10-7Z"/><circle cx="12" cy="12" r="3"/></svg>
								{/if}
							</button>
						</div>
					</div>

					<div>
						<label class="block text-xs font-medium text-white/40 mb-1.5" for="signup-confirm">Confirm Password</label>
						<div class="relative">
							<svg class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-white/20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
							<input
								id="signup-confirm"
								type="password"
								bind:value={confirmPassword}
								placeholder="••••••••"
								minlength="8"
								required
								class="w-full bg-black/40 border border-white/[0.06] rounded-lg px-10 py-2.5 text-sm text-white/80 placeholder-white/15 focus:outline-none focus:border-orange-500/50 focus:ring-1 focus:ring-orange-500/20 transition-colors"
							/>
						</div>
					</div>

					{#if error}
						<p class="text-xs text-red-400 bg-red-900/10 border border-red-900/40 rounded-lg px-3 py-2">{error}</p>
					{/if}

					<button
						type="submit"
						disabled={$authLoading}
						class="w-full py-2.5 rounded-lg bg-orange-500 hover:bg-orange-600 text-white text-sm font-semibold transition-all disabled:opacity-60 shadow-lg shadow-orange-500/20"
					>
						{#if $authLoading}
							<span class="flex items-center justify-center gap-2">
								<span class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin"></span>
								Creating account…
							</span>
						{:else}
							Sign Up
						{/if}
					</button>
				</form>

				<p class="text-center text-xs text-white/25 mt-5">
					Already have an account? <a href="/" class="text-orange-500 hover:text-orange-400 transition-colors">Sign in</a>
				</p>
			</div>
		</div>
	</div>
</div>
