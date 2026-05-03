<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import { toast } from '$lib/toast';
	import { setMcpEnv, startMCP } from '$lib/stores';

	export let mcpId: string;
	export let mcpName: string;
	export let requiredEnv: string[] = [];
	export let show = false;

	const dispatch = createEventDispatcher();

	let envValues: Record<string, string> = {};
	let loading = false;
	let visible: Record<string, boolean> = {};

	function getLabel(key: string): string {
		return key
			.replace(/_/g, ' ')
			.replace(/\b\w/g, (c) => c.toUpperCase())
			.replace(/Id$/i, 'ID');
	}

	function isSecret(key: string): boolean {
		return (
			key.includes('TOKEN') ||
			key.includes('KEY') ||
			key.includes('SECRET') ||
			key.includes('PASSWORD')
		);
	}

	async function handleSave() {
		const missing = requiredEnv.filter((k) => !envValues[k]?.trim());
		if (missing.length > 0) {
			toast.error(`Missing: ${missing.join(', ')}`);
			return;
		}

		loading = true;
		try {
			await setMcpEnv(mcpId, envValues);
			await startMCP(mcpId);
			toast.success(`${mcpName} configured and started!`);
			dispatch('complete');
			show = false;
		} catch (err) {
			toast.error(`Failed: ${err instanceof Error ? err.message : 'Unknown error'}`);
		} finally {
			loading = false;
		}
	}

	function handleSkip() {
		dispatch('skip');
		show = false;
	}
</script>

{#if show}
	<div class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
		<div class="bg-gray-900 border border-white/10 rounded-2xl p-6 w-full max-w-md mx-4 shadow-2xl">
			<!-- Header -->
			<div class="flex items-center gap-3 mb-4">
				<div class="w-10 h-10 rounded-xl bg-orange-500/10 border border-orange-500/20 flex items-center justify-center">
					<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="text-orange-400">
						<rect x="3" y="11" width="18" height="11" rx="2" ry="2"/>
						<path d="M7 11V7a5 5 0 0 1 10 0v4"/>
					</svg>
				</div>
				<div>
					<h3 class="text-sm font-bold text-white/90">Configure {mcpName}</h3>
					<p class="text-[11px] text-white/40">Enter required credentials to connect</p>
				</div>
			</div>

			<!-- Env var inputs -->
			<div class="space-y-3 mb-6">
				{#each requiredEnv as key}
					<div>
						<label class="block text-[11px] text-white/50 mb-1.5 font-medium">{getLabel(key)}</label>
						<div class="relative">
							<input
								bind:value={envValues[key]}
								placeholder={`Enter ${getLabel(key)}...`}
								type={isSecret(key) && !visible[key] ? 'password' : 'text'}
								class="w-full bg-white/[0.03] border border-white/[0.08] rounded-lg px-3 py-2 text-xs text-white/80 placeholder-white/20 focus:outline-none focus:border-orange-500/40 focus:ring-1 focus:ring-orange-500/20 transition-all"
							/>
							{#if isSecret(key)}
								<button
									on:click={() => (visible[key] = !visible[key])}
									class="absolute right-2 top-1/2 -translate-y-1/2 text-white/30 hover:text-white/60 transition-colors"
									aria-label={visible[key] ? 'Hide' : 'Show'}
								>
									{#if visible[key]}
										<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24"/><line x1="1" y1="1" x2="23" y2="23"/></svg>
									{:else}
										<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/></svg>
									{/if}
								</button>
							{/if}
						</div>
					</div>
				{/each}
			</div>

			<!-- Actions -->
			<div class="flex gap-2">
				<button
					on:click={handleSkip}
					class="flex-1 py-2 rounded-lg bg-white/[0.03] border border-white/[0.06] text-xs text-white/50 hover:text-white/80 hover:bg-white/[0.06] transition-all"
				>
					Skip for now
				</button>
				<button
					on:click={handleSave}
					disabled={loading}
					class="flex-1 py-2 rounded-lg bg-orange-500 text-white text-xs font-medium hover:bg-orange-600 transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2"
				>
					{#if loading}
						<svg class="animate-spin" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 12a9 9 0 11-6.219-8.56"/></svg>
					{/if}
					Save & Start
				</button>
			</div>
		</div>
	</div>
{/if}
