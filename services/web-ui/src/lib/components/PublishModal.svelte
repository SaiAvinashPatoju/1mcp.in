<script lang="ts">
	import type { Runtime } from '$lib/types';

	export let onClose: () => void;
	export let onPublish: (data: { name: string; description: string; version: string; runtime: Runtime; tags: string[]; verificationStatus: 'verified' | 'unverified'; fileName: string }) => void;

	const STEPS = ['Upload', 'Details', 'Verification', 'Confirm'];
	let step = 0;
	let file: File | null = null;
	let dragging = false;
	let name = '';
	let description = '';
	let version = '1.0.0';
	let runtime: Runtime = 'node';
	let tagInput = '';
	let tags: string[] = [];
	let verificationStatus: 'verified' | 'unverified' = 'unverified';
	let submitted = false;
	let fileInput: HTMLInputElement;

	function handleDrop(e: DragEvent) {
		e.preventDefault();
		dragging = false;
		const f = e.dataTransfer?.files[0];
		if (f && (f.name.endsWith('.tar.gz') || f.name.endsWith('.tgz'))) file = f;
	}

	function addTag() {
		const t = tagInput.trim().toLowerCase().replace(/\s+/g, '-');
		if (t && !tags.includes(t)) tags = [...tags, t];
		tagInput = '';
	}

	function canNext(): boolean {
		if (step === 0) return !!file;
		if (step === 1) return name.trim() !== '' && description.trim() !== '' && version.trim() !== '';
		return true;
	}

	function handleSubmit() {
		submitted = true;
		setTimeout(() => {
			onPublish({ name, description, version, runtime, tags, verificationStatus, fileName: file!.name });
			onClose();
		}, 1600);
	}
</script>

<!-- svelte-ignore a11y-click-events-have-key-events -->
<!-- svelte-ignore a11y-no-static-element-interactions -->
<div class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm" on:click|self={onClose}>
	<div class="w-full max-w-lg bg-[#12121a] border border-white/[0.06] rounded-2xl shadow-2xl">
		<div class="flex items-center justify-between px-5 py-4 border-b border-white/[0.06]">
			<h2 class="text-sm font-semibold text-white/90">Publish MCP Server</h2>
			<button on:click={onClose} class="w-8 h-8 flex items-center justify-center rounded-lg text-white/30 hover:text-white/80 hover:bg-white/[0.06] transition-colors">
				<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
			</button>
		</div>

		<!-- Steps -->
		<div class="flex items-center px-5 pt-5 pb-1 gap-0">
			{#each STEPS as s, i}
				<div class="flex items-center {i < STEPS.length - 1 ? 'flex-1' : ''}">
					<div class="flex items-center gap-1.5 text-xs font-medium {i <= step ? 'text-violet-400' : 'text-white/30'}">
						<div class="w-5 h-5 rounded-full flex items-center justify-center text-xs transition-colors
							{i < step ? 'bg-violet-600 text-white' : i === step ? 'border border-violet-500 text-violet-400' : 'border border-white/10 text-white/30'}"
						>
							{#if i < step}✓{:else}{i + 1}{/if}
						</div>
						{s}
					</div>
					{#if i < STEPS.length - 1}
						<div class="flex-1 h-px mx-2 {i < step ? 'bg-violet-600' : 'bg-white/[0.06]'}"></div>
					{/if}
				</div>
			{/each}
		</div>

		<div class="px-5 py-5 min-h-[16rem]">
			{#if step === 0}
				<p class="text-xs text-white/30 mb-4">Upload your packaged MCP server as a <code class="text-violet-400">.tar.gz</code> archive.</p>
				<!-- svelte-ignore a11y-no-static-element-interactions -->
				<div
					on:click={() => fileInput?.click()}
					on:dragover|preventDefault={() => (dragging = true)}
					on:dragleave={() => (dragging = false)}
					on:drop={handleDrop}
					class="border-2 border-dashed rounded-xl p-10 flex flex-col items-center gap-3 cursor-pointer transition-all
						{dragging ? 'border-violet-500 bg-violet-900/10' : file ? 'border-emerald-600 bg-emerald-900/10' : 'border-white/[0.08] hover:border-white/[0.16]'}"
				>
					<input bind:this={fileInput} type="file" accept=".tar.gz,.tgz" class="hidden" on:change={(e) => { const f = e.currentTarget.files?.[0]; if (f) file = f; }} />
					{#if file}
						<span class="text-2xl">✓</span>
						<p class="text-sm text-emerald-400 font-medium">{file.name}</p>
						<p class="text-xs text-white/30">{(file.size / 1024).toFixed(1)} KB · click to replace</p>
					{:else}
						<svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" class="text-white/30"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="17 8 12 3 7 8"/><line x1="12" y1="3" x2="12" y2="15"/></svg>
						<p class="text-sm text-white/70">Drop your <strong>.tar.gz</strong> here</p>
						<p class="text-xs text-white/30">or click to browse</p>
					{/if}
				</div>
			{:else if step === 1}
				<div class="space-y-4">
					<div>
						<label class="block text-xs font-medium text-white/30 mb-1.5">Name *</label>
						<input bind:value={name} placeholder="My MCP Server" class="w-full bg-black/40 border border-white/[0.06] rounded-lg px-3 py-2 text-sm text-white/80 placeholder-white/20 focus:outline-none focus:border-violet-500 transition-colors" />
					</div>
					<div>
						<label class="block text-xs font-medium text-white/30 mb-1.5">Description *</label>
						<textarea bind:value={description} placeholder="What does your MCP server do?" rows="3" class="w-full bg-black/40 border border-white/[0.06] rounded-lg px-3 py-2 text-sm text-white/80 placeholder-white/20 focus:outline-none focus:border-violet-500 transition-colors resize-none"></textarea>
					</div>
					<div class="flex gap-3">
						<div class="flex-1">
							<label class="block text-xs font-medium text-white/30 mb-1.5">Version *</label>
							<input bind:value={version} placeholder="1.0.0" class="w-full bg-black/40 border border-white/[0.06] rounded-lg px-3 py-2 text-sm text-white/80 placeholder-white/20 focus:outline-none focus:border-violet-500 transition-colors" />
						</div>
						<div class="flex-1">
							<label class="block text-xs font-medium text-white/30 mb-1.5">Runtime *</label>
							<select bind:value={runtime} class="w-full bg-black/40 border border-white/[0.06] rounded-lg px-3 py-2 text-sm text-white/80 focus:outline-none focus:border-violet-500 transition-colors">
								<option value="node">Node.js</option>
								<option value="python">Python</option>
								<option value="go">Go</option>
								<option value="binary">Binary</option>
							</select>
						</div>
					</div>
					<div>
						<label class="block text-xs font-medium text-white/30 mb-1.5">Tags</label>
						<div class="flex gap-2 mb-2">
							<input bind:value={tagInput} on:keydown={(e) => { if (e.key === 'Enter') { e.preventDefault(); addTag(); } }} placeholder="e.g. database, sql" class="flex-1 bg-black/40 border border-white/[0.06] rounded-lg px-3 py-2 text-sm text-white/80 placeholder-white/20 focus:outline-none focus:border-violet-500 transition-colors" />
							<button on:click={addTag} class="px-3 py-2 rounded-lg bg-white/[0.04] text-white/40 hover:text-white/80 text-xs border border-white/[0.06]">Add</button>
						</div>
						<div class="flex flex-wrap gap-1.5">
							{#each tags as tag}
								<span class="flex items-center gap-1 text-xs px-2 py-0.5 rounded bg-violet-900/30 text-violet-400 border border-violet-800/50">
									#{tag}
									<button on:click={() => (tags = tags.filter((t) => t !== tag))} class="hover:text-white">×</button>
								</span>
							{/each}
						</div>
					</div>
				</div>
			{:else if step === 2}
				<div class="space-y-3">
					<p class="text-xs text-white/30 mb-4">Choose how your MCP will be listed.</p>
					<button on:click={() => (verificationStatus = 'unverified')} class="w-full p-4 rounded-xl border text-left transition-all {verificationStatus === 'unverified' ? 'border-yellow-600 bg-yellow-900/10' : 'border-white/[0.06] hover:border-white/[0.12]'}">
						<div class="flex items-center gap-3 mb-2">
							<span class="text-yellow-400">⚠</span>
							<span class="text-sm font-semibold text-white/90">Community (Unverified)</span>
							{#if verificationStatus === 'unverified'}<span class="ml-auto text-yellow-400 text-xs">✓</span>{/if}
						</div>
						<p class="text-xs text-white/40 leading-relaxed">Listed immediately with a community badge.</p>
					</button>
					<button on:click={() => (verificationStatus = 'verified')} class="w-full p-4 rounded-xl border text-left transition-all {verificationStatus === 'verified' ? 'border-violet-600 bg-violet-900/10' : 'border-white/[0.06] hover:border-white/[0.12]'}">
						<div class="flex items-center gap-3 mb-2">
							<span class="text-emerald-400">🛡</span>
							<span class="text-sm font-semibold text-white/90">Verified</span>
							{#if verificationStatus === 'verified'}<span class="ml-auto text-emerald-400 text-xs">✓</span>{/if}
						</div>
						<p class="text-xs text-white/40 leading-relaxed">Queued for automated security scan (malware, OWASP, data-exfiltration). Verified MCPs rank higher. Review: 24–72h.</p>
					</button>
				</div>
			{:else if !submitted}
				<p class="text-xs text-white/30 mb-5">Review your submission.</p>
				<div class="space-y-3 bg-black/30 border border-white/[0.06] rounded-xl p-4">
					{#each [
						{ label: 'Package', value: file?.name },
						{ label: 'Name', value: name },
						{ label: 'Version', value: `v${version}` },
						{ label: 'Runtime', value: runtime },
						{ label: 'Tags', value: tags.length > 0 ? tags.map((t) => `#${t}`).join(' ') : '(none)' },
						{ label: 'Listing', value: verificationStatus === 'verified' ? '✦ Submit for verification' : '⚠ Community (unverified)' }
					] as row}
						<div class="flex items-start justify-between gap-4">
							<span class="text-xs text-white/30 flex-shrink-0">{row.label}</span>
							<span class="text-xs text-white/80 text-right">{row.value}</span>
						</div>
					{/each}
				</div>
			{:else}
				<div class="flex flex-col items-center justify-center h-48 gap-3">
					<div class="w-12 h-12 rounded-full bg-emerald-900/30 border border-emerald-600 flex items-center justify-center text-emerald-400 text-xl">✓</div>
					<p class="text-sm font-semibold text-white/90">{verificationStatus === 'verified' ? 'Submitted for Review!' : 'Published!'}</p>
					<p class="text-xs text-white/40 text-center max-w-xs">{verificationStatus === 'verified' ? "Queued for security review. We'll notify you." : 'Your MCP is now live in the marketplace.'}</p>
				</div>
			{/if}
		</div>

		{#if !submitted}
			<div class="flex items-center justify-between px-5 py-4 border-t border-white/[0.06]">
				<button on:click={() => (step > 0 ? (step -= 1) : onClose())} class="text-sm px-4 py-1.5 rounded-lg text-white/40 hover:text-white/80 hover:bg-white/[0.06] transition-colors">
					{step === 0 ? 'Cancel' : '← Back'}
				</button>
				{#if step < STEPS.length - 1}
					<button on:click={() => canNext() && (step += 1)} disabled={!canNext()} class="text-sm px-4 py-1.5 rounded-lg bg-violet-600 text-white hover:bg-violet-700 transition-colors font-medium disabled:opacity-40 disabled:cursor-not-allowed">
						Next →
					</button>
				{:else}
					<button on:click={handleSubmit} class="text-sm px-4 py-1.5 rounded-lg bg-violet-600 text-white hover:bg-violet-700 transition-colors font-medium">
						Publish
					</button>
				{/if}
			</div>
		{/if}
	</div>
</div>
