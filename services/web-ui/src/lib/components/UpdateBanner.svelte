<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { invoke } from '@tauri-apps/api/core';
  import { listen } from '@tauri-apps/api/event';

  type Phase = 'idle' | 'downloading' | 'ready' | 'error';

  let phase: Phase = 'idle';
  let version = '';
  let errorMsg = '';
  let restarting = false;

  const unlisten: Array<() => void> = [];

  onMount(async () => {
    unlisten.push(
      await listen<string>('update-downloading', (e) => {
        version = e.payload;
        phase = 'downloading';
      })
    );
    unlisten.push(
      await listen<string>('update-ready', (e) => {
        version = e.payload;
        phase = 'ready';
      })
    );
    unlisten.push(
      await listen<string>('update-error', (e) => {
        errorMsg = e.payload;
        phase = 'error';
        // auto-dismiss error after 8 s
        setTimeout(() => { if (phase === 'error') phase = 'idle'; }, 8000);
      })
    );
    unlisten.push(
      await listen('update-none', () => {
        // nothing to do
      })
    );
  });

  onDestroy(() => unlisten.forEach((fn) => fn()));

  async function restart() {
    restarting = true;
    await invoke('restart_app');
  }

  async function checkNow() {
    phase = 'idle';
    await invoke('check_update');
  }
</script>

{#if phase !== 'idle'}
  <div
    class="fixed bottom-4 right-4 z-50 w-80 rounded-xl border border-white/10 shadow-2xl overflow-hidden"
    style="background: #13131f; backdrop-filter: blur(16px);"
  >
    <!-- Coloured top stripe -->
    <div
      class="h-1 w-full"
      style="background: {phase === 'ready' ? '#7c3aed' : phase === 'error' ? '#ef4444' : '#2563eb'};"
    ></div>

    <div class="p-4">
      {#if phase === 'downloading'}
        <div class="flex items-center gap-3">
          <!-- Spinner -->
          <svg class="w-5 h-5 animate-spin text-blue-400 flex-shrink-0" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v4a4 4 0 00-4 4H4z"></path>
          </svg>
          <div>
            <p class="text-sm font-semibold text-white">Downloading v{version}</p>
            <p class="text-xs text-white/50 mt-0.5">Installing in background…</p>
          </div>
        </div>
        <!-- Indeterminate progress bar -->
        <div class="mt-3 h-1 rounded-full overflow-hidden" style="background:#1e1e2e;">
          <div class="h-full rounded-full animate-pulse" style="background:#2563eb; width:60%;"></div>
        </div>

      {:else if phase === 'ready'}
        <div class="flex items-start gap-3">
          <!-- Check icon -->
          <div class="w-8 h-8 rounded-full flex items-center justify-center flex-shrink-0" style="background:#7c3aed22;">
            <svg class="w-4 h-4 text-violet-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/>
            </svg>
          </div>
          <div class="flex-1 min-w-0">
            <p class="text-sm font-semibold text-white">v{version} ready</p>
            <p class="text-xs text-white/50 mt-0.5">Restart to apply the update</p>
          </div>
          <!-- Dismiss -->
          <button
            on:click={() => (phase = 'idle')}
            class="text-white/30 hover:text-white/70 transition-colors text-lg leading-none flex-shrink-0"
            aria-label="Dismiss"
          >×</button>
        </div>
        <button
          on:click={restart}
          disabled={restarting}
          class="mt-3 w-full py-2 rounded-lg text-sm font-semibold transition-all"
          style="background: linear-gradient(135deg,#7c3aed,#6d28d9); color:#fff; opacity:{restarting ? 0.6 : 1};"
        >
          {restarting ? 'Restarting…' : '↺ Restart Now'}
        </button>

      {:else if phase === 'error'}
        <div class="flex items-start gap-3">
          <svg class="w-5 h-5 text-red-400 flex-shrink-0 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z"/>
          </svg>
          <div class="flex-1 min-w-0">
            <p class="text-sm font-semibold text-white">Update failed</p>
            <p class="text-xs text-white/50 mt-0.5 truncate">{errorMsg}</p>
          </div>
          <button
            on:click={() => (phase = 'idle')}
            class="text-white/30 hover:text-white/70 transition-colors text-lg leading-none flex-shrink-0"
            aria-label="Dismiss"
          >×</button>
        </div>
        <button
          on:click={checkNow}
          class="mt-3 w-full py-2 rounded-lg text-sm font-semibold transition-all"
          style="background:#1e1e2e; color:#fff;"
        >Try Again</button>
      {/if}
    </div>
  </div>
{/if}
