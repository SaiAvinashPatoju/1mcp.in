<script lang="ts">
	import { user } from '$lib/auth';

	let saved = false;
	let notifyUpdates = true;
	let notifySecurity = true;
	let notifyReviews = false;

	$: initials = $user?.name?.split(' ').map((p) => p[0]).join('').toUpperCase().slice(0, 2) ?? '??';

	function handleSave() {
		saved = true;
		setTimeout(() => (saved = false), 2000);
	}
</script>

<div class="p-8 max-w-2xl mx-auto">
	<h1 class="text-xl font-bold text-white/95 mb-8">Account</h1>

	<!-- Profile -->
	<div class="bg-white/[0.03] border border-white/[0.06] rounded-2xl p-6 mb-5 backdrop-blur-sm">
		<div class="flex items-center gap-4 mb-6">
			<div class="w-16 h-16 rounded-full bg-violet-900/40 flex items-center justify-center text-xl font-bold text-violet-400 select-none flex-shrink-0">
				{initials}
			</div>
			<div>
				<p class="text-base font-semibold text-white/90">{$user?.name}</p>
				<p class="text-sm text-white/30">{$user?.email}</p>
				<span class="inline-block mt-2 text-xs px-2 py-0.5 rounded-full bg-violet-900/30 text-violet-400 border border-violet-800/50">Free Plan</span>
			</div>
		</div>
		<div class="space-y-4">
			<div>
				<label class="block text-xs font-medium text-white/30 mb-1.5" for="display-name">Display Name</label>
				<input id="display-name" value={$user?.name ?? ''} class="w-full bg-black/40 border border-white/[0.06] rounded-lg px-3 py-2 text-sm text-white/80 focus:outline-none focus:border-violet-500 transition-colors" />
			</div>
			<div>
				<label class="block text-xs font-medium text-white/30 mb-1.5" for="email-addr">Email Address</label>
				<input id="email-addr" type="email" value={$user?.email ?? ''} class="w-full bg-black/40 border border-white/[0.06] rounded-lg px-3 py-2 text-sm text-white/80 focus:outline-none focus:border-violet-500 transition-colors" />
			</div>
		</div>
	</div>

	<!-- Notifications -->
	<div class="bg-white/[0.03] border border-white/[0.06] rounded-2xl p-6 mb-5 backdrop-blur-sm">
		<div class="flex items-center gap-2 mb-5">
			<span>🔔</span>
			<h2 class="text-sm font-semibold text-white/90">Notifications</h2>
		</div>
		{#each [
			{ label: 'MCP updates & new versions', sub: 'Get notified when installed servers release updates', value: notifyUpdates, toggle: () => (notifyUpdates = !notifyUpdates) },
			{ label: 'Security alerts', sub: 'Warnings about vulnerable or flagged packages', value: notifySecurity, toggle: () => (notifySecurity = !notifySecurity) },
			{ label: 'Review replies', sub: 'When someone replies to your marketplace reviews', value: notifyReviews, toggle: () => (notifyReviews = !notifyReviews) }
		] as item}
			<div class="flex items-center justify-between gap-4 py-3">
				<div>
					<p class="text-sm text-white/80">{item.label}</p>
					<p class="text-xs text-white/30 mt-0.5">{item.sub}</p>
				</div>
				<button on:click={item.toggle} class="relative w-10 h-5 rounded-full transition-colors flex-shrink-0" style="background: {item.value ? '#7c3aed' : '#2a2a3a'}">
					<span class="absolute top-0.5 w-4 h-4 bg-white rounded-full transition-all shadow-sm" style="left: {item.value ? '1.375rem' : '0.125rem'}"></span>
				</button>
			</div>
		{/each}
	</div>

	<!-- Security -->
	<div class="bg-white/[0.03] border border-white/[0.06] rounded-2xl p-6 mb-6 backdrop-blur-sm">
		<div class="flex items-center gap-2 mb-5">
			<span>🛡</span>
			<h2 class="text-sm font-semibold text-white/90">Security</h2>
		</div>
		{#each [
			{ label: 'Change Password', desc: 'Update your account password' },
			{ label: 'Two-Factor Authentication', desc: 'Add an extra layer of security' },
			{ label: 'Active Sessions', desc: 'View and revoke signed-in devices' }
		] as item}
			<button class="w-full flex items-center justify-between py-3.5 border-b border-white/[0.04] last:border-b-0 group text-left">
				<div>
					<p class="text-sm text-white/80 group-hover:text-violet-400 transition-colors">{item.label}</p>
					<p class="text-xs text-white/30 mt-0.5">{item.desc}</p>
				</div>
				<span class="text-white/20 group-hover:text-violet-400 transition-colors">›</span>
			</button>
		{/each}
	</div>

	<button on:click={handleSave} class="w-full py-2.5 rounded-xl text-sm font-medium transition-all {saved ? 'bg-emerald-700 text-white' : 'bg-violet-600 text-white hover:bg-violet-700'}">
		{saved ? '✓ Saved' : 'Save Changes'}
	</button>
</div>
