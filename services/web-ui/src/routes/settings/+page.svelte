<script lang="ts">
	import { onMount } from 'svelte';
	import {
		user,
		authLoading,
		updateProfile,
		changePassword,
	} from '$lib/auth';
	import { browser } from '$app/environment';
	import { toast } from '$lib/toast';
	import {
		appPreferences,
		systemInfo,
		settingsSaved,
		settingsLoading,
		fetchAppPreferences,
		fetchSystemInfo,
		saveAppPreferences,
		resetRouterConfig,
		clearLocalData,
		copyDiagnostics,
		routerStatus,
		fetchRouterStatus,
	} from '$lib/stores';

	let topTab: 'settings' | 'account' = 'settings';
	let settingsTab:
		| 'general'
		| 'router'
		| 'mcp-servers'
		| 'clients'
		| 'security'
		| 'marketplace'
		| 'updates'
		| 'advanced' = 'general';

	// Local mutable copies for form binding
	let prefs = { ...$appPreferences };
	$: prefs = { ...$appPreferences };

	// Account form state
	let displayName = $user?.name ?? '';
	let emailAddress = $user?.email ?? '';
	let currentPassword = '';
	let newPassword = '';
	let confirmPassword = '';
	let accountSaved = false;
	let accountError = '';
	let passwordSaved = false;
	let passwordError = '';

	$: if ($user) {
		displayName = $user.name;
		emailAddress = $user.email;
	}

	$: initials =
		displayName
			.split(' ')
			.map((part) => part[0])
			.join('')
			.toUpperCase()
			.slice(0, 2) || '??';

	let resetConfirm = false;
	let uninstallConfirm = false;
	let clearDataConfirm = false;
	let diagnosticsCopied = false;
	let logLevelChanged = false;

	onMount(() => {
		fetchAppPreferences();
		fetchSystemInfo();
		fetchRouterStatus();
	});

	async function handleSaveSettings() {
		$settingsLoading = true;
		try {
			await saveAppPreferences(prefs);
			$settingsSaved = true;
			setTimeout(() => $settingsSaved = false, 2000);
		} catch (error) {
			console.error(error);
			toast.error('Failed to save settings');
		} finally {
			$settingsLoading = false;
		}
	}

	async function handleSaveAccount() {
		accountError = '';
		try {
			await updateProfile(displayName.trim(), emailAddress.trim());
			accountSaved = true;
			setTimeout(() => (accountSaved = false), 2000);
		} catch (error) {
			accountError =
				error instanceof Error ? error.message : 'Could not save account changes';
		}
	}

	async function handlePasswordSave() {
		passwordError = '';
		if (!currentPassword || !newPassword || !confirmPassword) {
			passwordError = 'Fill in all password fields.';
			return;
		}
		if (newPassword.length < 8) {
			passwordError = 'New password must be at least 8 characters.';
			return;
		}
		if (newPassword !== confirmPassword) {
			passwordError = 'New password and confirm password must match.';
			return;
		}
		try {
			await changePassword(currentPassword, newPassword);
			currentPassword = '';
			newPassword = '';
			confirmPassword = '';
			passwordSaved = true;
			setTimeout(() => (passwordSaved = false), 2000);
		} catch (error) {
			passwordError =
				error instanceof Error ? error.message : 'Could not update password';
		}
	}

	async function handleResetRouter() {
		if (!resetConfirm) {
			resetConfirm = true;
			return;
		}
		try {
			await resetRouterConfig();
			resetConfirm = false;
		} catch {
			resetConfirm = false;
			toast.error('Failed to reset router');
		}
	}

	async function handleUninstall() {
		if (!uninstallConfirm) {
			uninstallConfirm = true;
			return;
		}
		try {
			if (browser && '__TAURI_INTERNALS__' in window) {
				await import('@tauri-apps/api/core').then(m => m.invoke('uninstall_app'));
			} else {
				toast.info('Uninstall available in desktop app');
			}
		} catch {
			// ignore
		}
		uninstallConfirm = false;
	}

	async function handleClearData() {
		if (!clearDataConfirm) {
			clearDataConfirm = true;
			return;
		}
		try {
			await clearLocalData();
			clearDataConfirm = false;
		} catch {
			clearDataConfirm = false;
			toast.error('Failed to clear data');
		}
	}

	async function handleCopyDiagnostics() {
		try {
			const text = await copyDiagnostics();
			await navigator.clipboard.writeText(text);
			diagnosticsCopied = true;
			setTimeout(() => (diagnosticsCopied = false), 2000);
		} catch {
			// ignore
		}
	}

	function formatUptime(seconds: number): string {
		const h = Math.floor(seconds / 3600);
		const m = Math.floor((seconds % 3600) / 60);
		const s = seconds % 60;
		return `${h}h ${m}m ${s}s`;
	}

	function formatDate(dateStr: string): string {
		const d = new Date(dateStr);
		return d.toLocaleDateString('en-US', {
			month: 'short',
			day: 'numeric',
			year: 'numeric',
		});
	}

	const SETTINGS_TABS = [
		{ id: 'general', label: 'General', icon: 'general' },
		{ id: 'router', label: 'Router', icon: 'router' },
		{ id: 'mcp-servers', label: 'MCP Servers', icon: 'mcp' },
		{ id: 'clients', label: 'Clients', icon: 'clients' },
		{ id: 'security', label: 'Security', icon: 'security' },
		{ id: 'marketplace', label: 'Marketplace', icon: 'marketplace' },
		{ id: 'updates', label: 'Updates', icon: 'updates' },
		{ id: 'advanced', label: 'Advanced', icon: 'advanced' },
	] as const;

	function settingsIcon(name: string): string {
		switch (name) {
			case 'general':
				return `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"/></svg>`;
			case 'router':
				return `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="2" y="2" width="20" height="8" rx="2" ry="2"/><rect x="2" y="14" width="20" height="8" rx="2" ry="2"/><line x1="6" y1="6" x2="6.01" y2="6"/><line x1="6" y1="18" x2="6.01" y2="18"/></svg>`;
			case 'mcp':
				return `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/><polyline points="3.27 6.96 12 12.01 20.73 6.96"/><line x1="12" y1="22.08" x2="12" y2="12"/></svg>`;
			case 'clients':
				return `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>`;
			case 'security':
				return `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg>`;
			case 'marketplace':
				return `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M6 2L3 6v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2V6l-3-4z"/><line x1="3" y1="6" x2="21" y2="6"/><path d="M16 10a4 4 0 0 1-8 0"/></svg>`;
			case 'updates':
				return `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 12a9 9 0 0 0-9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/><path d="M3 3v5h5"/><path d="M3 12a9 9 0 0 0 9 9 9.75 9.75 0 0 0 6.74-2.74L21 16"/><path d="M16 16h5v5"/></svg>`;
			case 'advanced':
				return `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"/></svg>`;
			default:
				return '';
		}
	}
</script>

<div class="flex flex-col h-full">
	<!-- Header -->
	<div class="px-8 pt-8 pb-4">
		<h1 class="text-2xl font-bold text-white/95">Settings & Account</h1>
		<p class="text-sm text-white/40 mt-1">Manage your account, preferences, and system configuration.</p>
	</div>

	<!-- Top Tabs -->
	<div class="px-8 border-b border-white/[0.04]">
		<div class="flex gap-6">
			<button
				on:click={() => (topTab = 'settings')}
				class="pb-3 text-sm font-medium transition-colors relative {topTab === 'settings' ? 'text-orange-400' : 'text-white/40 hover:text-white/60'}"
			>
				Settings
				{#if topTab === 'settings'}
					<div class="absolute bottom-0 left-0 right-0 h-0.5 bg-orange-500 rounded-t-full"></div>
				{/if}
			</button>
			<button
				on:click={() => (topTab = 'account')}
				class="pb-3 text-sm font-medium transition-colors relative {topTab === 'account' ? 'text-orange-400' : 'text-white/40 hover:text-white/60'}"
			>
				Account
				{#if topTab === 'account'}
					<div class="absolute bottom-0 left-0 right-0 h-0.5 bg-orange-500 rounded-t-full"></div>
				{/if}
			</button>
		</div>
	</div>

	{#if topTab === 'settings'}
		<div class="flex flex-1 overflow-hidden">
			<!-- Settings Left Sidebar -->
			<div class="w-56 border-r border-white/[0.04] py-4 flex flex-col">
				{#each SETTINGS_TABS as tab}
					<button
						on:click={() => (settingsTab = tab.id)}
						class="flex items-center gap-2.5 px-5 py-2 text-sm transition-colors text-left w-full {settingsTab === tab.id ? 'text-orange-400 font-medium' : 'text-white/40 hover:text-white/60'}"
					>
						<span class="flex-shrink-0">{@html settingsIcon(tab.icon)}</span>
						{tab.label}
					</button>
				{/each}
			</div>

			<!-- Main Content -->
			<div class="flex-1 overflow-y-auto px-8 py-6">
				{#if settingsTab === 'general'}
					<!-- General Settings -->
					<div class="space-y-8 max-w-2xl">
						<!-- Startup -->
						<div>
							<h3 class="text-sm font-semibold text-white/90 mb-1">General</h3>
							<p class="text-xs text-white/30 mb-4">Core preferences and startup behavior.</p>

							<div class="space-y-4">
								<div class="flex items-center justify-between">
									<div>
										<p class="text-sm text-white/80">Start mach1 on login</p>
										<p class="text-xs text-white/30 mt-0.5">Automatically start the mach1 router when your system starts.</p>
									</div>
									<button
										aria-label="Start mach1 on login"
										on:click={() => (prefs = { ...prefs, start_on_login: !prefs.start_on_login })}
										class="relative w-10 h-5 rounded-full transition-colors flex-shrink-0"
										style="background: {prefs.start_on_login ? '#f97316' : '#2a2a3a'}"
									>
										<span
											class="absolute top-0.5 w-4 h-4 bg-white rounded-full transition-all shadow-sm"
											style="left: {prefs.start_on_login ? '1.375rem' : '0.125rem'}"
										></span>
									</button>
								</div>

								<div class="flex items-center justify-between">
									<div>
										<p class="text-sm text-white/80">Minimize to system tray</p>
										<p class="text-xs text-white/30 mt-0.5">Keep 1mcp.in running in the background.</p>
									</div>
									<button
										aria-label="Minimize to system tray"
										on:click={() => (prefs = { ...prefs, minimize_to_tray: !prefs.minimize_to_tray })}
										class="relative w-10 h-5 rounded-full transition-colors flex-shrink-0"
										style="background: {prefs.minimize_to_tray ? '#f97316' : '#2a2a3a'}"
									>
										<span
											class="absolute top-0.5 w-4 h-4 bg-white rounded-full transition-all shadow-sm"
											style="left: {prefs.minimize_to_tray ? '1.375rem' : '0.125rem'}"
										></span>
									</button>
								</div>

								<div class="flex items-center justify-between">
									<div>
										<p class="text-sm text-white/80">Theme</p>
										<p class="text-xs text-white/30 mt-0.5">Choose your preferred theme.</p>
									</div>
									<select
										bind:value={prefs.theme}
										class="bg-white/[0.04] border border-white/[0.06] rounded-lg px-3 py-1.5 text-sm text-white/80 focus:outline-none focus:border-orange-500/40"
									>
										<option value="dark">Dark</option>
										<option value="light">Light</option>
										<option value="system">System</option>
									</select>
								</div>

								<div class="flex items-center justify-between">
									<div>
										<p class="text-sm text-white/80">Language</p>
										<p class="text-xs text-white/30 mt-0.5">Select application language.</p>
									</div>
									<select
										bind:value={prefs.language}
										class="bg-white/[0.04] border border-white/[0.06] rounded-lg px-3 py-1.5 text-sm text-white/80 focus:outline-none focus:border-orange-500/40"
									>
										<option value="System Default">System Default</option>
										<option value="English">English</option>
									</select>
								</div>
							</div>
						</div>

						<div class="border-t border-white/[0.06]"></div>

						<!-- Data & Storage -->
						<div>
							<h3 class="text-sm font-semibold text-white/90 mb-1">Data & Storage</h3>
							<p class="text-xs text-white/30 mb-4">Configure local storage and data management.</p>

							<div class="space-y-4">
								<div>
									<p class="text-sm text-white/80 mb-1">Data directory</p>
									<p class="text-xs text-white/30 mb-2">Location for registry, configs, and logs.</p>
									<div class="flex items-center gap-2">
										<div class="flex-1 bg-white/[0.03] border border-white/[0.06] rounded-lg px-3 py-2 text-sm text-white/60 font-mono">
											{$systemInfo?.data_directory ?? '~/.1mcp'}
										</div>
										<button
											aria-label="Copy data directory path"
											on:click={() => navigator.clipboard.writeText($systemInfo?.data_directory ?? '~/.1mcp')}
											class="p-2 rounded-lg bg-white/[0.04] border border-white/[0.06] text-white/40 hover:text-white/60 transition-colors"
										>
											<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>
										</button>
									</div>
								</div>

								<div class="flex items-center justify-between">
									<div>
										<p class="text-sm text-white/80">Clear local data</p>
										<p class="text-xs text-white/30 mt-0.5">Remove all local data, configs, and cached files.</p>
									</div>
									<button
										on:click={handleClearData}
										class="px-4 py-1.5 rounded-lg text-xs font-medium border border-red-500/30 text-red-400 hover:bg-red-500/10 transition-colors"
									>
										{clearDataConfirm ? 'Confirm' : 'Clear Data'}
									</button>
								</div>
							</div>
						</div>

						<div class="border-t border-white/[0.06]"></div>

						<!-- Telemetry -->
						<div>
							<h3 class="text-sm font-semibold text-white/90 mb-1">Telemetry</h3>
							<p class="text-xs text-white/30 mb-4">Help improve 1mcp.in by sharing anonymous usage data.</p>

							<div class="flex items-center justify-between">
								<div>
									<p class="text-sm text-white/80">Enable telemetry</p>
									<p class="text-xs text-white/30 mt-0.5">Send anonymous usage and crash reports.</p>
								</div>
								<button
									aria-label="Enable telemetry"
									on:click={() => (prefs = { ...prefs, telemetry_enabled: !prefs.telemetry_enabled })}
									class="relative w-10 h-5 rounded-full transition-colors flex-shrink-0"
									style="background: {prefs.telemetry_enabled ? '#f97316' : '#2a2a3a'}"
								>
									<span
										class="absolute top-0.5 w-4 h-4 bg-white rounded-full transition-all shadow-sm"
										style="left: {prefs.telemetry_enabled ? '1.375rem' : '0.125rem'}"
									></span>
								</button>
							</div>
						</div>

						<!-- Save Button -->
						<div class="pt-4">
							<button
								on:click={handleSaveSettings}
								disabled={$settingsLoading}
								class="px-6 py-2.5 rounded-lg text-sm font-medium transition-all disabled:opacity-60 {$settingsSaved ? 'bg-emerald-600 text-white' : 'bg-orange-500 text-white hover:bg-orange-600'}"
							>
								{#if $settingsLoading}
									Saving...
								{:else if $settingsSaved}
									✓ Saved
								{:else}
									Save Changes
								{/if}
							</button>
						</div>
					</div>
				{:else if settingsTab === 'router'}
					<div class="max-w-2xl">
						<h3 class="text-sm font-semibold text-white/90 mb-1">Router</h3>
						<p class="text-xs text-white/30 mb-6">Configure mach1 router behavior and transport settings.</p>
						<div class="bg-white/[0.02] border border-white/[0.06] rounded-xl p-8 text-center">
							<p class="text-sm text-white/30">Router settings will be available in a future update.</p>
						</div>
					</div>
				{:else if settingsTab === 'mcp-servers'}
					<div class="max-w-2xl">
						<h3 class="text-sm font-semibold text-white/90 mb-1">MCP Servers</h3>
						<p class="text-xs text-white/30 mb-6">Manage default MCP server behaviors and timeouts.</p>
						<div class="bg-white/[0.02] border border-white/[0.06] rounded-xl p-8 text-center">
							<p class="text-sm text-white/30">MCP server settings will be available in a future update.</p>
						</div>
					</div>
				{:else if settingsTab === 'clients'}
					<div class="max-w-2xl">
						<h3 class="text-sm font-semibold text-white/90 mb-1">Clients</h3>
						<p class="text-xs text-white/30 mb-6">Configure default client connection behaviors.</p>
						<div class="bg-white/[0.02] border border-white/[0.06] rounded-xl p-8 text-center">
							<p class="text-sm text-white/30">Client settings will be available in a future update.</p>
						</div>
					</div>
				{:else if settingsTab === 'security'}
					<div class="max-w-2xl">
						<h3 class="text-sm font-semibold text-white/90 mb-1">Security</h3>
						<p class="text-xs text-white/30 mb-6">Manage security policies and approval gates.</p>
						<div class="bg-white/[0.02] border border-white/[0.06] rounded-xl p-8 text-center">
							<p class="text-sm text-white/30">Security settings will be available in a future update.</p>
						</div>
					</div>
				{:else if settingsTab === 'marketplace'}
					<div class="max-w-2xl">
						<h3 class="text-sm font-semibold text-white/90 mb-1">Marketplace</h3>
						<p class="text-xs text-white/30 mb-6">Configure marketplace discovery and update checks.</p>
						<div class="bg-white/[0.02] border border-white/[0.06] rounded-xl p-8 text-center">
							<p class="text-sm text-white/30">Marketplace settings will be available in a future update.</p>
						</div>
					</div>
				{:else if settingsTab === 'updates'}
					<div class="max-w-2xl">
						<h3 class="text-sm font-semibold text-white/90 mb-1">Updates</h3>
						<p class="text-xs text-white/30 mb-6">Manage automatic updates and release channels.</p>
						<div class="bg-white/[0.02] border border-white/[0.06] rounded-xl p-8 text-center">
							<p class="text-sm text-white/30">Update settings will be available in a future update.</p>
						</div>
					</div>
				{:else if settingsTab === 'advanced'}
					<div class="max-w-2xl">
						<h3 class="text-sm font-semibold text-white/90 mb-1">Advanced</h3>
						<p class="text-xs text-white/30 mb-6">Advanced configuration options for power users.</p>
						<div class="bg-white/[0.02] border border-white/[0.06] rounded-xl p-8 text-center">
							<p class="text-sm text-white/30">Advanced settings will be available in a future update.</p>
						</div>
					</div>
				{/if}
			</div>

			<!-- Right Sidebar -->
			<div class="w-80 border-l border-white/[0.04] p-6 space-y-6 overflow-y-auto">
				<!-- Account Card -->
				<div class="bg-white/[0.02] border border-white/[0.06] rounded-xl p-5">
					<div class="flex items-center justify-between mb-4">
						<h3 class="text-sm font-semibold text-white/90">Account</h3>
						<button
							on:click={() => (topTab = 'account')}
							class="text-xs text-white/40 hover:text-white/60 transition-colors px-2 py-1 rounded border border-white/[0.06]"
						>
							Manage
						</button>
					</div>
					<div class="flex items-center gap-3 mb-4">
						<div class="w-12 h-12 rounded-full bg-orange-500 flex items-center justify-center text-sm font-bold text-white flex-shrink-0">
							{initials}
						</div>
						<div class="min-w-0">
							<p class="text-sm font-medium text-white/90 truncate">{$user?.name ?? 'Guest'}</p>
							<p class="text-xs text-white/30 truncate">{$user?.email ?? 'No email'}</p>
						</div>
						<span class="text-[10px] px-2 py-0.5 rounded-full bg-blue-500/10 text-blue-400 border border-blue-500/20 flex-shrink-0">Local Account</span>
					</div>
					<div class="space-y-2 text-xs">
						<div class="flex justify-between">
							<span class="text-white/30">Account Type</span>
							<span class="text-white/60">Local (Open Source)</span>
						</div>
						<div class="flex justify-between">
							<span class="text-white/30">Member Since</span>
							<span class="text-white/60">May 2, 2025</span>
						</div>
						<div class="flex justify-between">
							<span class="text-white/30">Team</span>
							<span class="text-white/60">Not a member</span>
						</div>
					</div>
					<button on:click={() => toast.info('Team Pro coming soon — join waitlist at 1mcp.in')} class="mt-4 w-full py-2 rounded-lg text-xs font-medium border border-orange-500/30 text-orange-400 hover:bg-orange-500/10 transition-colors">
						Join Team Pro
					</button>
				</div>

				<!-- System Information -->
				<div class="bg-white/[0.02] border border-white/[0.06] rounded-xl p-5">
					<h3 class="text-sm font-semibold text-white/90 mb-4">System Information</h3>
					<div class="space-y-2.5 text-xs">
						<div class="flex justify-between">
							<span class="text-white/30">Platform</span>
							<span class="text-white/60">{$systemInfo?.platform ?? 'Linux x86_64'}</span>
						</div>
						<div class="flex justify-between">
							<span class="text-white/30">Version</span>
							<span class="text-white/60">{$systemInfo?.version ?? 'v1.0.0'}</span>
						</div>
						<div class="flex justify-between items-center">
							<span class="text-white/30">Router Status</span>
							<span class="flex items-center gap-1.5">
								<span class="w-1.5 h-1.5 rounded-full {$routerStatus?.status === 'running' ? 'bg-emerald-500' : 'bg-red-500'}"></span>
								<span class="text-emerald-400">{$routerStatus?.status === 'running' ? 'Running' : 'Stopped'}</span>
							</span>
						</div>
						<div class="flex justify-between">
							<span class="text-white/30">Transport</span>
							<span class="text-white/60">{$systemInfo?.transport ?? 'stdio'}</span>
						</div>
						<div class="flex justify-between">
							<span class="text-white/30">Uptime</span>
							<span class="text-white/60">{formatUptime($systemInfo?.uptime_seconds ?? $routerStatus?.uptime_seconds ?? 0)}</span>
						</div>
						<div class="flex justify-between items-center">
							<span class="text-white/30">Metrics Endpoint</span>
							<span class="flex items-center gap-1 text-white/60">
								{$systemInfo?.metrics_endpoint ?? '127.0.0.1:3031/metrics'}
								<button aria-label="Open metrics endpoint" on:click={() => window.open('http://' + ($systemInfo?.metrics_endpoint ?? '127.0.0.1:3031/metrics'), '_blank')} class="text-white/20 hover:text-white/40 transition-colors">
									<svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/><polyline points="15 3 21 3 21 9"/><line x1="10" y1="14" x2="21" y2="3"/></svg>
								</button>
							</span>
						</div>
						<div class="flex justify-between items-center">
							<span class="text-white/30">Log Level</span>
							<select
								bind:value={prefs.log_level}
								on:change={() => { logLevelChanged = true; handleSaveSettings(); }}
								class="bg-white/[0.04] border border-white/[0.06] rounded px-2 py-0.5 text-xs text-white/80 focus:outline-none focus:border-orange-500/40"
							>
								<option value="debug">Debug</option>
								<option value="info">Info</option>
								<option value="warn">Warn</option>
								<option value="error">Error</option>
							</select>
						</div>
					</div>
					<button
						on:click={handleCopyDiagnostics}
						class="mt-4 w-full py-2 rounded-lg text-xs font-medium border border-white/[0.08] text-white/50 hover:text-white/70 hover:bg-white/[0.03] transition-colors flex items-center justify-center gap-1.5"
					>
						<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>
						{diagnosticsCopied ? 'Copied!' : 'Copy Diagnostics'}
					</button>
				</div>

				<!-- Danger Zone -->
				<div class="bg-white/[0.02] border border-white/[0.06] rounded-xl p-5">
					<h3 class="text-sm font-semibold text-orange-400/80 mb-4">Danger Zone</h3>
					<div class="space-y-4">
						<div class="flex items-center justify-between">
							<div>
								<p class="text-sm text-white/80">Reset router configuration</p>
								<p class="text-xs text-white/30 mt-0.5">Reset all router settings to default.</p>
							</div>
							<button
								on:click={handleResetRouter}
								class="px-3 py-1.5 rounded-lg text-xs font-medium border border-red-500/30 text-red-400 hover:bg-red-500/10 transition-colors flex-shrink-0"
							>
								{resetConfirm ? 'Confirm' : 'Reset Router'}
							</button>
						</div>
						<div class="flex items-center justify-between">
							<div>
								<p class="text-sm text-white/80">Uninstall 1mcp.in</p>
								<p class="text-xs text-white/30 mt-0.5">Remove 1mcp.in and all related data from this system.</p>
							</div>
							<button
								on:click={handleUninstall}
								class="px-3 py-1.5 rounded-lg text-xs font-medium border border-red-500/30 text-red-400 hover:bg-red-500/10 transition-colors flex-shrink-0"
							>
								{uninstallConfirm ? 'Confirm' : 'Uninstall'}
							</button>
						</div>
					</div>
				</div>
			</div>
		</div>
	{:else}
		<!-- Account Tab -->
		<div class="flex flex-1 overflow-hidden">
			<div class="flex-1 overflow-y-auto px-8 py-6">
				<div class="max-w-2xl space-y-6">
					<!-- Profile Card -->
					<div class="bg-white/[0.02] border border-white/[0.06] rounded-xl p-6">
						<div class="flex items-center gap-4 mb-6">
							<div class="w-16 h-16 rounded-full bg-orange-500 flex items-center justify-center text-xl font-bold text-white flex-shrink-0 select-none">
								{initials}
							</div>
							<div>
								<p class="text-base font-semibold text-white/90">{displayName}</p>
								<p class="text-sm text-white/30">{emailAddress}</p>
								<span class="inline-block mt-2 text-xs px-2 py-0.5 rounded-full bg-blue-500/10 text-blue-400 border border-blue-500/20">
									Local Account
								</span>
							</div>
						</div>
						<div class="space-y-4">
							<div>
								<label class="block text-xs font-medium text-white/30 mb-1.5" for="acc-display-name">Display Name</label>
								<input id="acc-display-name" bind:value={displayName} class="w-full bg-white/[0.03] border border-white/[0.06] rounded-lg px-3 py-2 text-sm text-white/80 focus:outline-none focus:border-orange-500/40 transition-colors" />
							</div>
							<div>
								<label class="block text-xs font-medium text-white/30 mb-1.5" for="acc-email">Email Address</label>
								<input id="acc-email" type="email" bind:value={emailAddress} class="w-full bg-white/[0.03] border border-white/[0.06] rounded-lg px-3 py-2 text-sm text-white/80 focus:outline-none focus:border-orange-500/40 transition-colors" />
							</div>
						</div>
						{#if accountError}
							<p class="mt-4 text-xs text-red-400 bg-red-900/10 border border-red-900/40 rounded-lg px-3 py-2">{accountError}</p>
						{/if}
						<button
							disabled={$authLoading}
							on:click={handleSaveAccount}
							class="mt-5 px-6 py-2.5 rounded-lg text-sm font-medium transition-all disabled:opacity-60 {accountSaved ? 'bg-emerald-600 text-white' : 'bg-orange-500 text-white hover:bg-orange-600'}"
						>
							{#if $authLoading}
								Saving...
							{:else if accountSaved}
								✓ Saved
							{:else}
								Save Profile
							{/if}
						</button>
					</div>

					<!-- Password -->
					<div class="bg-white/[0.02] border border-white/[0.06] rounded-xl p-6">
						<h3 class="text-sm font-semibold text-white/90 mb-4">Password</h3>
						<div class="space-y-4">
							<div>
								<label class="block text-xs font-medium text-white/30 mb-1.5" for="acc-current-password">Current Password</label>
								<input id="acc-current-password" type="password" bind:value={currentPassword} class="w-full bg-white/[0.03] border border-white/[0.06] rounded-lg px-3 py-2 text-sm text-white/80 focus:outline-none focus:border-orange-500/40 transition-colors" />
							</div>
							<div>
								<label class="block text-xs font-medium text-white/30 mb-1.5" for="acc-new-password">New Password</label>
								<input id="acc-new-password" type="password" bind:value={newPassword} class="w-full bg-white/[0.03] border border-white/[0.06] rounded-lg px-3 py-2 text-sm text-white/80 focus:outline-none focus:border-orange-500/40 transition-colors" />
							</div>
							<div>
								<label class="block text-xs font-medium text-white/30 mb-1.5" for="acc-confirm-password">Confirm Password</label>
								<input id="acc-confirm-password" type="password" bind:value={confirmPassword} class="w-full bg-white/[0.03] border border-white/[0.06] rounded-lg px-3 py-2 text-sm text-white/80 focus:outline-none focus:border-orange-500/40 transition-colors" />
							</div>
						</div>
						<p class="mt-3 text-xs text-white/30">Passwords are stored as bcrypt hashes and never returned to the UI.</p>
						{#if passwordError}
							<p class="mt-4 text-xs text-red-400 bg-red-900/10 border border-red-900/40 rounded-lg px-3 py-2">{passwordError}</p>
						{/if}
						<button
							disabled={$authLoading}
							on:click={handlePasswordSave}
							class="mt-5 px-6 py-2.5 rounded-lg text-sm font-medium transition-all disabled:opacity-60 {passwordSaved ? 'bg-emerald-600 text-white' : 'bg-orange-500 text-white hover:bg-orange-600'}"
						>
							{#if $authLoading}
								Updating...
							{:else if passwordSaved}
								✓ Password Updated
							{:else}
								Update Password
							{/if}
						</button>
					</div>
				</div>
			</div>
		</div>
	{/if}
</div>
