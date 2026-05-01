<script lang="ts">
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import { forgotPassword, authLoading, isAuthenticated } from '$lib/auth';
	import AuthLeftPanel from '$lib/components/AuthLeftPanel.svelte';

	let email = '';
	let sent = false;
	let error = '';

	onMount(() => {
		const unsub = isAuthenticated.subscribe((v) => {
			if (v) goto('/dashboard');
		});
		return unsub;
	});

	async function handleSubmit() {
		error = '';
		if (!email.trim()) {
			error = 'Email is required';
			return;
		}
		try {
			await forgotPassword(email.trim());
			sent = true;
		} catch (e: any) {
			error = e?.message ?? 'Something went wrong';
		}
	}
</script>

<svelte:head>
	<title>1mcp.in — Reset Password</title>
</svelte:head>

<div class="min-h-screen bg-[#08080c] flex relative overflow-hidden">
	<AuthLeftPanel />

	<!-- RIGHT RESET SECTION -->
	<div class="w-full lg:w-1/2 flex items-center justify-center p-6 relative z-10">
		<div class="w-full max-w-[420px]">
			<div class="bg-[#0f0f14]/80 border border-white/[0.06] rounded-2xl p-8 shadow-2xl backdrop-blur-xl">
				<h2 class="text-xl font-bold text-white/95">Reset your password</h2>
				<p class="text-sm text-white/35 mt-1">Enter your email to receive a reset link</p>

				{#if sent}
					<div class="mt-6 rounded-xl bg-emerald-600/10 border border-emerald-500/20 p-4">
						<p class="text-sm text-emerald-300">If this email is registered, a reset link has been sent.</p>
					</div>
					<a href="/" class="mt-4 block w-full py-2.5 rounded-lg bg-orange-500 hover:bg-orange-600 text-white text-sm font-semibold text-center transition-colors">Back to Login</a>
				{:else}
					<form on:submit|preventDefault={handleSubmit} class="mt-6 space-y-4">
						<div>
							<label class="block text-xs font-medium text-white/40 mb-1.5" for="reset-email">Email</label>
							<div class="relative">
								<svg class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-white/20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="2" y="4" width="20" height="16" rx="2"/><path d="m22 7-8.97 5.7a1.94 1.94 0 0 1-2.06 0L2 7"/></svg>
								<input
									id="reset-email"
									type="email"
									bind:value={email}
									placeholder="you@example.com"
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
									Sending…
								</span>
							{:else}
								Send Reset Link
							{/if}
						</button>
					</form>

					<p class="text-center text-xs text-white/25 mt-5">
						<a href="/" class="text-orange-500 hover:text-orange-400 transition-colors">Back to Login</a>
					</p>
				{/if}
			</div>
		</div>
	</div>
</div>
