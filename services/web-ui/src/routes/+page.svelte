<script lang="ts">
	import { goto } from '$app/navigation';
	import { signIn, signUp, authLoading, isAuthenticated } from '$lib/auth';
	import { onMount } from 'svelte';

	let mode: 'signin' | 'signup' = 'signin';
	let name = '';
	let email = '';
	let password = '';
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
		if (mode === 'signup') {
			if (!name.trim()) { error = 'Name is required'; return; }
			await signUp(name.trim(), email, password);
		} else {
			await signIn(email, password);
		}
	}
</script>

<div class="min-h-screen bg-[#0a0a0f] flex flex-col items-center justify-center p-4 relative overflow-hidden">
	<div class="absolute top-1/3 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[600px] h-[400px] bg-violet-600/5 rounded-full blur-3xl pointer-events-none"></div>

	<div class="w-full max-w-sm relative z-10">
		<div class="flex flex-col items-center gap-3 mb-8">
			<div class="w-14 h-14 rounded-2xl bg-gradient-to-br from-violet-600 to-violet-800 flex items-center justify-center shadow-lg shadow-violet-900/50">
				<span class="text-xl font-black text-white">M1</span>
			</div>
			<div class="text-center">
				<h1 class="text-2xl font-bold text-white/95">Mach1</h1>
				<p class="text-sm text-white/30 mt-1">Your universal MCP gateway</p>
			</div>
		</div>

		<div class="bg-white/[0.03] border border-white/[0.06] rounded-2xl p-6 shadow-2xl backdrop-blur-sm">
			<div class="flex bg-black/30 rounded-lg p-1 mb-6 border border-white/[0.04]">
				{#each ['signin', 'signup'] as m}
					<button
						on:click={() => { mode = m; error = ''; }}
						class="flex-1 py-1.5 text-sm rounded-md transition-all font-medium
							{mode === m ? 'bg-violet-600 text-white shadow-sm' : 'text-white/30 hover:text-white/60'}"
					>
						{m === 'signin' ? 'Sign In' : 'Sign Up'}
					</button>
				{/each}
			</div>

			<form on:submit|preventDefault={handleSubmit} class="space-y-4">
				{#if mode === 'signup'}
					<div>
						<label class="block text-xs font-medium text-white/30 mb-1.5" for="name-input">Name</label>
						<input id="name-input" type="text" bind:value={name} placeholder="Your full name" class="w-full bg-black/40 border border-white/[0.06] rounded-lg px-3 py-2 text-sm text-white/80 placeholder-white/15 focus:outline-none focus:border-violet-500 transition-colors" />
					</div>
				{/if}

				<div>
					<label class="block text-xs font-medium text-white/30 mb-1.5" for="email-input">Email</label>
					<input id="email-input" type="email" bind:value={email} placeholder="you@example.com" required class="w-full bg-black/40 border border-white/[0.06] rounded-lg px-3 py-2 text-sm text-white/80 placeholder-white/15 focus:outline-none focus:border-violet-500 transition-colors" />
				</div>

				<div>
					<label class="block text-xs font-medium text-white/30 mb-1.5" for="password-input">Password</label>
					<div class="relative">
						<input id="password-input" type={showPassword ? 'text' : 'password'} bind:value={password} placeholder="••••••••" required minlength="8" class="w-full bg-black/40 border border-white/[0.06] rounded-lg px-3 py-2 text-sm text-white/80 placeholder-white/15 focus:outline-none focus:border-violet-500 transition-colors pr-10" />
						<button type="button" on:click={() => (showPassword = !showPassword)} class="absolute right-2.5 top-1/2 -translate-y-1/2 text-white/20 hover:text-white/60">
							{showPassword ? '🙈' : '👁'}
						</button>
					</div>
				</div>

				{#if error}
					<p class="text-xs text-red-400 bg-red-900/10 border border-red-900/40 rounded-lg px-3 py-2">{error}</p>
				{/if}

				<button type="submit" disabled={$authLoading} class="w-full py-2 rounded-lg bg-violet-600 text-white text-sm font-medium hover:bg-violet-700 transition-colors disabled:opacity-60 disabled:cursor-not-allowed mt-2">
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
		</div>

		<p class="text-center text-xs text-white/15 mt-4">By continuing, you agree to the Mach1 Terms of Service.</p>
	</div>
</div>
