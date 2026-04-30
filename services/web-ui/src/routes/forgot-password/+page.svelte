<script lang="ts">
    import { goto } from '$app/navigation';
    import { toast } from '$lib/toast';
    let email = '';
    let sent = false;
    async function handleSubmit() {
        if (!email) return;
        sent = true;
        toast.success('Password reset link sent if this email is registered');
    }
</script>

<div class="flex items-center justify-center min-h-screen px-4">
    <div class="w-full max-w-sm">
        <h1 class="text-2xl font-bold text-white/90 mb-2">Forgot Password</h1>
        <p class="text-sm text-white/40 mb-6">Enter your email to receive a reset link.</p>
        {#if sent}
            <div class="rounded-xl bg-emerald-600/10 border border-emerald-500/20 p-4 mb-6">
                <p class="text-sm text-emerald-300">Password reset link sent if this email is registered.</p>
            </div>
            <button on:click={() => goto('/')} class="w-full py-2.5 rounded-lg bg-orange-500 text-white text-sm font-medium hover:bg-orange-600 transition-colors">Back to Login</button>
        {:else}
            <form on:submit|preventDefault={handleSubmit} class="space-y-4">
                <input type="email" bind:value={email} placeholder="you@example.com" required class="w-full bg-white/[0.03] border border-white/[0.06] rounded-lg px-3 py-2 text-sm text-white/80 placeholder-white/20 focus:outline-none focus:border-orange-500/30" />
                <button type="submit" disabled={!email} class="w-full py-2.5 rounded-lg bg-orange-500 text-white text-sm font-medium hover:bg-orange-600 transition-colors disabled:opacity-50">Send Reset Link</button>
            </form>
            <button on:click={() => goto('/')} class="mt-4 w-full text-center text-xs text-white/30 hover:text-white/50 transition-colors">Back to Login</button>
        {/if}
    </div>
</div>
