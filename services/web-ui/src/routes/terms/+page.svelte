<script lang="ts">
    import { goto } from '$app/navigation';
</script>

<div class="max-w-3xl mx-auto px-6 py-12">
    <button on:click={() => goto('/')} class="text-xs text-white/30 hover:text-white/50 mb-6 transition-colors">&larr; Back</button>
    <h1 class="text-2xl font-bold text-white/90 mb-6">Terms of Service</h1>
    <div class="space-y-4 text-sm text-white/50 leading-relaxed">
        <p>1mcp.in ("we", "our", "us") provides a local-first MCP hub and router software.</p>
        <p>By using this software, you agree to these terms. The software is provided under the MIT License as open source.</p>
        <h2 class="text-lg font-semibold text-white/70 mt-6">Use License</h2>
        <p>Permission is granted to use, copy, modify, and distribute the software subject to the MIT License terms.</p>
        <h2 class="text-lg font-semibold text-white/70 mt-6">No Warranty</h2>
        <p>The software is provided "as is" without warranty of any kind. The authors are not liable for any damages.</p>
        <h2 class="text-lg font-semibold text-white/70 mt-6">Contact</h2>
        <p>For questions, open an issue at github.com/SaiAvinashPatoju/1mcp.in.</p>
    </div>
</div>
