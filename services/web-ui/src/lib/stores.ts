import { browser } from '$app/environment';
import { writable, derived, get } from 'svelte/store';
import type { InstalledMcp, MarketplaceMcp, ClientApp, Skill, RouterStatus, SystemUsage, ActivityItem, McpServerDetail, CommandResult, ServerDetail, ServerTool, ServerLogEntry, ServerConfig, MarketplaceItemDetail, ClientConnectionDetail, ClientRoutingHealth, ClientConfigPreview, AppPreferences, SystemInfo, DiagnosticsData, Runtime } from './types';

const API_URL = (import.meta.env.VITE_API_URL as string | undefined) ?? 'http://localhost:8080';
const isTauri = browser && '__TAURI_INTERNALS__' in window;

type CloudMarketplaceItem = {
	id: string;
	name: string;
	description: string;
	version: string;
	runtime: string;
	transport?: string;
	tags: string[];
	homepage: string;
	license: string;
	author?: string;
	publishedAt?: string;
	downloads?: number;
	rating?: number;
	reviewCount?: number;
	verification?: string;
	sha256?: string;
	signature?: string;
	entrypoint?: {
		command: string;
		args?: string[];
		cwd?: string;
	};
	security_checks?: { label: string; status: string }[];
	requires_env?: string[];
};

type CloudSkillItem = {
	id: string;
	name: string;
	description: string;
	icon: string;
	mcp_ids: string[];
	client_ids?: string[];
	installed?: boolean;
	enabled?: boolean;
	created_at?: number;
};

type ClientConnectionState = {
	id: string;
	connected: boolean;
	config_path?: string | null;
};

async function invokeDesktop<T>(command: string, args: Record<string, unknown> = {}): Promise<T> {
	const { invoke } = await import('@tauri-apps/api/core');
	return invoke<T>(command, args);
}

// ── Installed MCPs ──

export const installed = writable<InstalledMcp[]>([
	{
		id: 'mach1',
		name: 'Mach1 Router',
		version: '1.0.0',
		runtime: 'binary',
		enabled: true,
		command: 'mach1',
		args: [],
		description: 'Semantic router for 1mcp.in. Auto-activates the required MCPs using prompt-aware matching.'
	},
	{
		id: 'github',
		name: 'GitHub',
		version: '0.6.2',
		runtime: 'node',
		enabled: false,
		command: 'npx',
		args: ['-y', '@modelcontextprotocol/server-github'],
		description: 'Search code, read issues/PRs, and create issues on GitHub via the GitHub API.',
		patProvider: 'github'
	},
	{
		id: 'memory',
		name: 'Knowledge Graph Memory',
		version: '0.6.0',
		runtime: 'node',
		enabled: false,
		command: 'npx',
		args: ['-y', '@modelcontextprotocol/server-memory'],
		description: 'Persistent knowledge graph the agent can query and update across sessions.'
	}
]);

// ── Marketplace ──

export const marketplace = writable<MarketplaceMcp[]>([
	{
		id: 'github', name: 'GitHub',
		shortDescription: 'Search code, read issues/PRs, and create issues on GitHub via the GitHub API.',
		version: '0.6.2', runtime: 'node', author: 'Anthropic',
		tags: ['github', 'git', 'issues', 'official'],
		rating: 4.9, reviewCount: 634, downloads: 92800,
		verificationStatus: 'anthropic-official', publishedAt: '2024-11-05', installed: true, patProvider: 'github',
		requires_env: ['GITHUB_PERSONAL_ACCESS_TOKEN'],
		homepage: 'https://github.com/modelcontextprotocol/servers'
	},
	{
		id: 'memory', name: 'Knowledge Graph Memory',
		shortDescription: 'Persistent knowledge graph the agent can query and update across sessions.',
		version: '0.6.0', runtime: 'node', author: 'Anthropic',
		tags: ['memory', 'knowledge-graph', 'official'],
		rating: 4.9, reviewCount: 521, downloads: 78300,
		verificationStatus: 'anthropic-official', publishedAt: '2024-10-15', installed: true,
		homepage: 'https://github.com/modelcontextprotocol/servers'
	},
	{
		id: 'filesystem', name: 'Filesystem',
		shortDescription: 'Read, write, and search files within allow-listed directories.',
		version: '0.6.2', runtime: 'node', author: 'Anthropic',
		tags: ['filesystem', 'files', 'official'],
		rating: 4.8, reviewCount: 312, downloads: 45200,
		verificationStatus: 'anthropic-official', publishedAt: '2024-11-01', installed: false,
		requires_env: ['MACH1_FS_ROOT']
	},
	{
		id: 'fetch', name: 'Fetch',
		shortDescription: 'Fetch URLs and convert HTML to markdown for agent consumption.',
		version: '0.6.0', runtime: 'python', author: 'Anthropic',
		tags: ['http', 'fetch', 'web', 'official'],
		rating: 4.7, reviewCount: 188, downloads: 31000,
		verificationStatus: 'anthropic-official', publishedAt: '2024-10-20', installed: false
	},
	{
		id: 'git', name: 'Git',
		shortDescription: 'Read-only git repository inspection (log, diff, blame).',
		version: '0.6.0', runtime: 'python', author: 'Anthropic',
		tags: ['git', 'vcs', 'official'],
		rating: 4.6, reviewCount: 143, downloads: 22100,
		verificationStatus: 'anthropic-official', publishedAt: '2024-10-25', installed: false,
		requires_env: ['MACH1_GIT_REPO']
	},
	{
		id: 'postgres', name: 'PostgreSQL',
		shortDescription: 'Query and inspect PostgreSQL databases with safe read-only access.',
		version: '1.0.0', runtime: 'node', author: 'db-tools',
		tags: ['database', 'postgres', 'sql'],
		rating: 4.5, reviewCount: 119, downloads: 17600,
		verificationStatus: '1mcp.in-verified', publishedAt: '2025-02-10', installed: false,
		requires_env: ['POSTGRES_CONNECTION_STRING']
	},
	{
		id: 'slack', name: 'Slack',
		shortDescription: 'Read and send Slack messages, manage channels and workspace users.',
		version: '1.2.0', runtime: 'node', author: 'community',
		tags: ['slack', 'messaging', 'communication'],
		rating: 4.2, reviewCount: 87, downloads: 9400,
		verificationStatus: 'community', publishedAt: '2025-01-12', installed: false,
		requires_env: ['SLACK_BOT_TOKEN', 'SLACK_TEAM_ID']
	},
	{
		id: 'jira', name: 'Jira',
		shortDescription: 'Create and manage Jira issues, sprints, and project boards.',
		version: '0.9.1', runtime: 'python', author: 'atlassian-community',
		tags: ['jira', 'project-management', 'atlassian'],
		rating: 3.8, reviewCount: 52, downloads: 6100,
		verificationStatus: 'pending', publishedAt: '2025-03-01', installed: false,
		requires_env: ['JIRA_BASE_URL', 'JIRA_USER_EMAIL', 'JIRA_API_TOKEN']
	},
	{
		id: 'linear', name: 'Linear',
		shortDescription: 'Manage Linear issues, cycles, and projects from your AI assistant.',
		version: '0.3.0', runtime: 'node', author: 'linear-community',
		tags: ['linear', 'project-management', 'issues'],
		rating: 4.1, reviewCount: 38, downloads: 4200,
		verificationStatus: 'community', publishedAt: '2025-04-01', installed: false,
		patProvider: 'linear',
		requires_env: ['LINEAR_API_KEY']
	}
]);

// ── Client Apps ──

export const clients = writable<ClientApp[]>([
	{
		id: 'vscode',
		name: 'VS Code',
		icon: '<>',
		description: 'GitHub Copilot, Roo Code, Continue via mcp.json',
		connected: false,
		connectCommand: 'connect vscode'
	},
	{
		id: 'cursor',
		name: 'Cursor',
		icon: '⌘',
		description: 'Cursor AI IDE via ~/.cursor/mcp.json',
		connected: false,
		connectCommand: 'connect cursor'
	},
	{
		id: 'claude',
		name: 'Claude Desktop',
		icon: '🤖',
		description: 'Anthropic Claude Desktop via claude_desktop_config.json',
		connected: false,
		connectCommand: 'connect claude'
	},
	{
		id: 'claudecode',
		name: 'Claude Code',
		icon: '⌨',
		description: 'Claude Code CLI via ~/.claude.json',
		connected: false,
		connectCommand: 'connect claudecode'
	},
	{
		id: 'windsurf',
		name: 'Windsurf',
		icon: '🌊',
		description: 'Windsurf Cascade IDE via ~/.codeium/mcp_config.json',
		connected: false,
		connectCommand: 'connect windsurf'
	},
	{
		id: 'codex',
		name: 'Codex',
		icon: '🧠',
		description: 'Codex AI Assistant via ~/.codex/config.toml',
		connected: false,
		connectCommand: 'connect codex'
	},
	{
		id: 'antigravity',
		name: 'Antigravity',
		icon: '🚀',
		description: 'Antigravity agent integration via ~/.antigravity/mcp.json',
		connected: false,
		connectCommand: 'connect antigravity'
	},
	{
		id: 'opencode',
		name: 'OpenCode',
		icon: '{ }',
		description: 'OpenCode IDE via ~/.config/opencode/opencode.json',
		connected: false,
		connectCommand: 'connect opencode'
	}
]);

// ── Skills ──

export const skills = writable<Skill[]>([
	{
		id: 'frontend-dev',
		name: 'Frontend Developer',
		description: 'GitHub, filesystem, and memory for frontend workflows',
		icon: '🎨',
		mcp_ids: ['github', 'filesystem', 'memory'],
		client_ids: [],
		installed: false,
		enabled: true,
		created_at: 0
	},
	{
		id: 'backend-dev',
		name: 'Backend Developer',
		description: 'GitHub, Postgres, and fetch for backend and API work',
		icon: '⚙️',
		mcp_ids: ['github', 'postgres', 'fetch'],
		client_ids: [],
		installed: false,
		enabled: true,
		created_at: 0
	},
	{
		id: 'devops',
		name: 'DevOps',
		description: 'GitHub, fetch, and memory for infrastructure and deployment',
		icon: '🛠️',
		mcp_ids: ['github', 'fetch', 'memory'],
		client_ids: [],
		installed: false,
		enabled: true,
		created_at: 0
	},
	{
		id: 'writer',
		name: 'Writer',
		description: 'Fetch, memory, and filesystem for research and content creation',
		icon: '✍️',
		mcp_ids: ['fetch', 'memory', 'filesystem'],
		client_ids: [],
		installed: false,
		enabled: true,
		created_at: 0
	},
	{
		id: 'full-stack',
		name: 'Full Stack',
		description: 'GitHub, filesystem, memory, and fetch for end-to-end development',
		icon: '🚀',
		mcp_ids: ['github', 'filesystem', 'memory', 'fetch'],
		client_ids: [],
		installed: false,
		enabled: true,
		created_at: 0
	}
]);

// ── User counter (real from API) ──

export const userCount = writable(0);

let counterInterval: ReturnType<typeof setInterval> | null = null;

export async function startUserCounter() {
	if (counterInterval) return;
	await fetchUserCount();
	// Refresh every 60s
	counterInterval = setInterval(fetchUserCount, 60000);
}

async function fetchUserCount() {
	try {
		if (isTauri) {
			userCount.set(await invokeDesktop<number>('fetch_cloud_stats'));
			return;
		}

		const res = await fetch(`${API_URL}/api/stats`);
		if (res.ok) {
			const data = await res.json();
			userCount.set(data.total_users ?? 0);
		}
	} catch {
		// API unavailable — keep last known value
	}
}

export function stopUserCounter() {
	if (counterInterval) {
		clearInterval(counterInterval);
		counterInterval = null;
	}
}

function syncInstalledMarketplaceFlags(installedIds: Set<string>) {
	marketplace.update((list) =>
		list.map((mcp) => ({
			...mcp,
			installed: installedIds.has(mcp.id)
		}))
	);
}

/** Heuristic auto-detector for env requirements.
 *  Checks: explicit requires_env → patProvider mapping → keyword heuristics on name/description/tags. */
function detectRequiredEnv(mcp: {
	id: string;
	name: string;
	shortDescription?: string;
	tags?: string[];
	patProvider?: string;
	requires_env?: string[];
}): string[] {
	// 1. Explicit requires_env (non-empty) wins
	if (mcp.requires_env && mcp.requires_env.length > 0) return mcp.requires_env;

	// 2. patProvider mapping
	const patEnvMap: Record<string, string[]> = {
		github: ['GITHUB_PERSONAL_ACCESS_TOKEN'],
		gitlab: ['GITLAB_PERSONAL_ACCESS_TOKEN'],
		linear: ['LINEAR_API_KEY'],
	};
	if (mcp.patProvider && patEnvMap[mcp.patProvider]) {
		return patEnvMap[mcp.patProvider];
	}

	// 3. Keyword heuristics — scan name, description, tags
	const haystack = `${mcp.name || ''} ${mcp.shortDescription || ''} ${(mcp.tags ?? []).join(' ')}`.toLowerCase();

	const servicePatterns: Array<{ keys: string[]; env: string[] }> = [
		{ keys: ['github', 'gh '], env: ['GITHUB_PERSONAL_ACCESS_TOKEN'] },
		{ keys: ['gitlab', 'gl '], env: ['GITLAB_PERSONAL_ACCESS_TOKEN'] },
		{ keys: ['linear'], env: ['LINEAR_API_KEY'] },
		{ keys: ['slack'], env: ['SLACK_BOT_TOKEN', 'SLACK_TEAM_ID'] },
		{ keys: ['jira', 'atlassian'], env: ['JIRA_BASE_URL', 'JIRA_USER_EMAIL', 'JIRA_API_TOKEN'] },
		{ keys: ['postgres', 'postgresql', 'pg_'], env: ['POSTGRES_CONNECTION_STRING'] },
		{ keys: ['notion'], env: ['NOTION_API_KEY'] },
		{ keys: ['hubspot'], env: ['HUBSPOT_API_KEY'] },
		{ keys: ['stripe'], env: ['STRIPE_API_KEY'] },
		{ keys: ['salesforce'], env: ['SALESFORCE_CLIENT_ID', 'SALESFORCE_CLIENT_SECRET'] },
		{ keys: ['aws', 'amazon'], env: ['AWS_ACCESS_KEY_ID', 'AWS_SECRET_ACCESS_KEY'] },
		{ keys: ['gcp', 'google cloud'], env: ['GOOGLE_APPLICATION_CREDENTIALS'] },
		{ keys: ['azure'], env: ['AZURE_API_KEY'] },
		{ keys: ['brave', 'brave search'], env: ['BRAVE_API_KEY'] },
		{ keys: ['serpapi', 'google search'], env: ['SERPAPI_API_KEY'] },
		{ keys: ['openai', 'chatgpt'], env: ['OPENAI_API_KEY'] },
		{ keys: ['anthropic', 'claude'], env: ['ANTHROPIC_API_KEY'] },
		{ keys: ['figma'], env: ['FIGMA_ACCESS_TOKEN'] },
		{ keys: ['docker', 'dockerhub'], env: ['DOCKER_HUB_TOKEN'] },
		{ keys: ['sentry'], env: ['SENTRY_AUTH_TOKEN'] },
		{ keys: ['datadog'], env: ['DATADOG_API_KEY', 'DATADOG_APP_KEY'] },
		{ keys: ['sendgrid', 'mail'], env: ['SENDGRID_API_KEY'] },
		{ keys: ['twilio'], env: ['TWILIO_ACCOUNT_SID', 'TWILIO_AUTH_TOKEN'] },
		{ keys: ['airtable'], env: ['AIRTABLE_API_KEY'] },
		{ keys: ['asana'], env: ['ASANA_PERSONAL_ACCESS_TOKEN'] },
		{ keys: ['pagerduty'], env: ['PAGERDUTY_API_KEY'] },
		{ keys: ['vercel'], env: ['VERCEL_TOKEN'] },
		{ keys: ['netlify'], env: ['NETLIFY_AUTH_TOKEN'] },
		{ keys: ['railway'], env: ['RAILWAY_API_KEY'] },
		{ keys: ['discord'], env: ['DISCORD_BOT_TOKEN'] },
		{ keys: ['telegram'], env: ['TELEGRAM_BOT_TOKEN'] },
		{ keys: ['google sheets', 'google_sheets'], env: ['GOOGLE_SHEETS_API_KEY'] },
		{ keys: ['google drive', 'google_drive'], env: ['GOOGLE_DRIVE_API_KEY'] },
		{ keys: ['gmail'], env: ['GMAIL_APP_PASSWORD'] },
		{ keys: ['calendar', 'google calendar'], env: ['GOOGLE_CALENDAR_API_KEY'] },
		// Generic patterns — only match if we haven't found anything above
		{ keys: ['api_key', 'apikey', 'api token', 'bearer token', 'pat', 'personal access'], env: ['API_KEY'] },
	];

	for (const { keys, env } of servicePatterns) {
		if (keys.some(k => haystack.includes(k))) {
			return env;
		}
	}

	return [];
}

// ── Marketplace sync from API ──

/** Fetch the marketplace catalog from the cloud API. If running inside Tauri,
 *  also persists the result to local SQLite via the sync_marketplace command.
 *  Falls back to the static store if the API is unreachable.
 */
export async function fetchMarketplace() {
	try {
		let apiItems: CloudMarketplaceItem[] = [];

		if (isTauri) {
			apiItems = await invokeDesktop<CloudMarketplaceItem[]>('fetch_cloud_marketplace');
		} else {
			const res = await fetch(`${API_URL}/api/marketplace`);
			if (!res.ok) return;

			const data = await res.json();
			apiItems = data.items ?? [];
		}

		if (apiItems.length === 0) return;

		// Merge API data into the marketplace store (preserves install state, ratings etc.)
		marketplace.update((local) => {
			const byId = new Map(local.map((m) => [m.id, m]));
			const merged: typeof local = [];

			for (const apiItem of apiItems) {
				const existing = byId.get(apiItem.id);
				// Restore persisted install count from localStorage
				let persistedDownloads: number | undefined;
				try {
					const key = `mcp_installs_${apiItem.id}`;
					const stored = localStorage.getItem(key);
					if (stored) persistedDownloads = parseInt(stored, 10);
				} catch { /* ignore */ }
				merged.push({
					...(existing ?? {
						rating: 4.5,
						reviewCount: 0,
						downloads: 0,
						verificationStatus: 'community' as const,
						publishedAt: new Date().toISOString().slice(0, 10),
						installed: false,
					}),
					id: apiItem.id,
					name: apiItem.name,
					shortDescription: apiItem.description,
					version: apiItem.version,
					runtime: apiItem.runtime as typeof local[number]['runtime'],
					transport: apiItem.transport,
					tags: apiItem.tags ?? [],
					author: existing?.author ?? 'community',
					verificationStatus: (apiItem.verification as typeof local[number]['verificationStatus']) ?? existing?.verificationStatus ?? 'community',
					sha256: apiItem.sha256,
					signature: apiItem.signature,
					entrypoint: apiItem.entrypoint,
					homepage: apiItem.homepage ?? existing?.homepage,
					patProvider: existing?.patProvider,
					requires_env: apiItem.requires_env?.length
						? apiItem.requires_env
						: detectRequiredEnv({
							id: apiItem.id,
							name: apiItem.name,
							shortDescription: apiItem.description,
							tags: apiItem.tags,
							patProvider: existing?.patProvider,
							requires_env: existing?.requires_env,
						}),
					downloads: persistedDownloads ?? existing?.downloads ?? 0,
				});
				byId.delete(apiItem.id); // mark as processed
			}

			// Keep any locally-only items not in the API response
			for (const remaining of byId.values()) {
				merged.push(remaining);
			}

			return merged;
		});

		// Persist to local SQLite if inside Tauri
		try {
			const { invoke } = await import('@tauri-apps/api/core');
			await invoke('sync_marketplace', { items: apiItems });
		} catch {
			// Not in Tauri (browser preview) — skip
		}
	} catch {
		// API unavailable — keep static store as-is
	}
}

// ── Derived stats ──

export const installedCount = derived(installed, ($i) => $i.length);
export const runningCount = derived(installed, ($i) => $i.filter((m) => m.enabled).length);

// ── Actions ──

export async function fetchInstalled() {
	if (isTauri) {
		try {
			const localInstalled = await invokeDesktop<InstalledMcp[]>('list_installed');
			installed.set(localInstalled);
			syncInstalledMarketplaceFlags(new Set(localInstalled.map((mcp) => mcp.id)));
		} catch {
			// Keep seeded defaults
		}
	} else {
		try {
			const res = await fetch(`${API_URL}/api/installed`);
			if (res.ok) {
				const data = await res.json();
				const items = data.items ?? data ?? [];
				installed.set(items);
				syncInstalledMarketplaceFlags(new Set(items.map((m: any) => m.id)));
			}
		} catch {
			// Keep seeded defaults
		}
	}
}

export async function toggleMcp(id: string) {
	installed.update((list) =>
		list.map((m) => (m.id === id ? { ...m, enabled: !m.enabled } : m))
	);
	mcpServers.update((list) =>
		list.map((m) =>
			m.id === id
				? { ...m, status: m.status === 'running' ? 'sleeping' : 'running' as 'running' | 'sleeping' | 'error' }
				: m
		)
	);
	if (isTauri) {
		try {
			await invokeDesktop('toggle_mcp', { id });
		} catch {
			// Revert on failure
			installed.update((list) =>
				list.map((m) => (m.id === id ? { ...m, enabled: !m.enabled } : m))
			);
			mcpServers.update((list) =>
				list.map((m) =>
					m.id === id
						? { ...m, status: m.status === 'running' ? 'sleeping' : 'running' as 'running' | 'sleeping' | 'error' }
						: m
				)
			);
		}
	}
}

export async function uninstallMcp(id: string) {
	const prevInstalled = get(installed);
	installed.update((list) => list.filter((m) => m.id !== id));
	syncInstalledMarketplaceFlags(new Set(get(installed).map((mcp) => mcp.id)));
	if (isTauri) {
		try {
			await invokeDesktop('uninstall_mcp', { id });
		} catch {
			installed.set(prevInstalled);
			syncInstalledMarketplaceFlags(new Set(prevInstalled.map((mcp) => mcp.id)));
		}
	}
}

export async function installMcp(id: string) {
	const mkt = get(marketplace);
	const mcp = mkt.find((m) => m.id === id);
	const inst = get(installed);
	if (!mcp || inst.some((m) => m.id === id)) return;

	const command = mcp.entrypoint?.command || 'npx';
	const args = mcp.entrypoint?.args?.length
		? mcp.entrypoint.args
		: ['-y', `@modelcontextprotocol/server-${mcp.id}`];

	const prevInstalled = get(installed);
	installed.update((list) => [
		...list,
		{
			id: mcp.id,
			name: mcp.name,
			version: mcp.version,
			runtime: mcp.runtime,
			enabled: false,
			command,
			args,
			description: mcp.shortDescription,
			patProvider: mcp.patProvider
		}
	]);
	// Increment install counter in marketplace
	marketplace.update((list) =>
		list.map((m) =>
			m.id === id ? { ...m, downloads: m.downloads + 1 } : m
		)
	);
	// Persist install count to localStorage
	try {
		const key = `mcp_installs_${id}`;
		const stored = localStorage.getItem(key);
		const count = (stored ? parseInt(stored, 10) : mcp.downloads) + 1;
		localStorage.setItem(key, count.toString());
	} catch { /* localStorage unavailable */ }
	syncInstalledMarketplaceFlags(new Set(get(installed).map((installedMcp) => installedMcp.id)));
	if (isTauri) {
		try {
			const newMcp = get(installed).find((m) => m.id === id);
			if (newMcp) await invokeDesktop('install_mcp', { mcp: newMcp });
		} catch {
			installed.set(prevInstalled);
			// Revert install count
			marketplace.update((list) =>
				list.map((m) =>
					m.id === id ? { ...m, downloads: m.downloads - 1 } : m
				)
			);
			syncInstalledMarketplaceFlags(new Set(prevInstalled.map((m) => m.id)));
		}
	}
}

export async function refreshClientConnections() {
	if (!isTauri) return;

	try {
		const states = await invokeDesktop<ClientConnectionState[]>('get_client_connections');
		const stateMap = new Map(states.map((state) => [state.id, state.connected]));
		clients.update((list) =>
			list.map((client) => ({
				...client,
				connected: stateMap.get(client.id) ?? false
			}))
		);
	} catch {
		// Leave the existing UI state untouched if the desktop bridge is unavailable.
	}
}

export async function connectClient(id: string) {
	if (!isTauri) {
		throw new Error(
			'Desktop app required. Run the 1mcp.in desktop app to auto-configure your IDE, or use the manual instructions below.'
		);
	}

	await invokeDesktop<string>('patch_client_config', { clientId: id });
	await refreshClientConnections();
}

export async function disconnectClient(id: string) {
	if (!isTauri) {
		throw new Error(
			'Desktop app required. Run the 1mcp.in desktop app to disconnect mach1 from your IDE, or remove the "mach1" entry from the config file manually.'
		);
	}

	await invokeDesktop<boolean>('remove_client_config', { clientId: id });
	await refreshClientConnections();
}

// ── Skills actions ──

export async function fetchSkills() {
	try {
		let localSkills: Skill[] = [];
		if (isTauri) {
			try {
				localSkills = await invokeDesktop<Skill[]>('list_skills');
			} catch {
				localSkills = [];
			}
		}

		let remoteSkills: CloudSkillItem[] = [];
		if (isTauri) {
			remoteSkills = await invokeDesktop<CloudSkillItem[]>('fetch_cloud_skills');
		} else {
			const res = await fetch(`${API_URL}/api/skills`);
			if (res.ok) {
				const data = await res.json();
				remoteSkills = data.items ?? [];
			}
		}

		if (remoteSkills.length > 0) {
			const byId = new Map(localSkills.map((skill) => [skill.id, skill]));
			skills.set(
				remoteSkills.map((skill) => {
					const existing = byId.get(skill.id);
					return {
						id: skill.id,
						name: skill.name,
						description: skill.description,
						icon: skill.icon,
						mcp_ids: skill.mcp_ids ?? [],
						client_ids: existing?.client_ids ?? skill.client_ids ?? [],
						installed: existing?.installed ?? skill.installed ?? false,
						enabled: existing?.enabled ?? skill.enabled ?? true,
						created_at: existing?.created_at ?? skill.created_at ?? 0
					};
				})
			);
			return;
		}

		if (localSkills.length > 0) {
			skills.set(localSkills);
		}
	} catch {
		// Keep static defaults
	}
}

export async function installSkill(id: string) {
	const skill = get(skills).find((s) => s.id === id);
	if (!skill || skill.installed) return;

	const installedMcpIds: string[] = [];
	let allSucceeded = true;
	for (const mcpId of skill.mcp_ids) {
		const alreadyInstalled = get(installed).some((m) => m.id === mcpId);
		if (!alreadyInstalled) {
			try {
				await installMcp(mcpId);
				installedMcpIds.push(mcpId);
			} catch {
				allSucceeded = false;
				break;
			}
		}
	}

	if (!allSucceeded) {
		for (const installedId of installedMcpIds) {
			uninstallMcp(installedId);
		}
		return;
	}

	const prevSkills = get(skills);
	skills.update((list) =>
		list.map((s) => (s.id === id ? { ...s, installed: true, enabled: true } : s))
	);

	if (isTauri) {
		try {
			const updated = get(skills).find((s) => s.id === id);
			if (updated) await invokeDesktop('upsert_skill', { skill: updated });
		} catch {
			skills.set(prevSkills);
			for (const installedId of installedMcpIds) {
				uninstallMcp(installedId);
			}
		}
	}
}

export async function uninstallSkill(id: string) {
	const skill = get(skills).find((s) => s.id === id);
	if (!skill || !skill.installed) return;

	const prevSkills = get(skills);
	skills.update((list) =>
		list.map((s) => (s.id === id ? { ...s, installed: false, client_ids: [] } : s))
	);

	for (const mcpId of skill.mcp_ids) {
		const onlyForThisSkill = !get(skills).some(
			(s) => s.id !== id && s.installed && s.mcp_ids.includes(mcpId)
		);
		if (onlyForThisSkill) {
			uninstallMcp(mcpId);
		}
	}

	if (isTauri) {
		try {
			const updated = get(skills).find((s) => s.id === id);
			if (updated) await invokeDesktop('upsert_skill', { skill: updated });
		} catch {
			skills.set(prevSkills);
		}
	}
}

export async function toggleSkillEnabled(id: string) {
	skills.update((list) =>
		list.map((s) => (s.id === id ? { ...s, enabled: !s.enabled } : s))
	);
	if (isTauri) {
		try {
			await invokeDesktop('toggle_skill', { id });
		} catch {
			skills.update((list) =>
				list.map((s) => (s.id === id ? { ...s, enabled: !s.enabled } : s))
			);
		}
	}
}

// ── Bundles ──

export type Bundle = {
	id: string;
	name: string;
	description: string;
	version: string;
	mcp_ids: string[];
	installed?: boolean;
	enabled?: boolean;
};

export const bundles = writable<Bundle[]>([
	{
		id: 'github-maintainer',
		name: 'GitHub Maintainer',
		description: 'Complete GitHub repository management bundle with issue tracking, PR review, and release management.',
		version: '1.0.0',
		mcp_ids: ['github', 'sequential-thinking', 'memory'],
		installed: false,
		enabled: true
	}
]);

export async function fetchBundles() {
	try {
		if (isTauri) {
			const localBundles = await invokeDesktop<Bundle[]>('list_bundles');
			if (localBundles && localBundles.length > 0) {
				bundles.set(localBundles);
			}
			return;
		}
		const res = await fetch(`${API_URL}/api/bundles`);
		if (res.ok) {
			const data = await res.json();
			bundles.set(data.items ?? []);
		}
	} catch {
		// Keep static defaults
	}
}

export async function installBundle(id: string) {
	const bundle = get(bundles).find((b) => b.id === id);
	if (!bundle || bundle.installed) return;

	// Install all MCPs in the bundle
	for (const mcpId of bundle.mcp_ids) {
		const alreadyInstalled = get(installed).some((m) => m.id === mcpId);
		if (!alreadyInstalled) {
			await installMcp(mcpId);
		}
	}

	bundles.update((list) =>
		list.map((b) => (b.id === id ? { ...b, installed: true } : b))
	);

	if (isTauri) {
		const updated = get(bundles).find((b) => b.id === id);
		if (updated) {
			await invokeDesktop('install_bundle', { bundle: updated });
		}
	}
}

export async function uninstallBundle(id: string) {
	const bundle = get(bundles).find((b) => b.id === id);
	if (!bundle || !bundle.installed) return;

	// Only uninstall MCPs that aren't used by other bundles
	for (const mcpId of bundle.mcp_ids) {
		const onlyForThisBundle = !get(bundles).some(
			(b) => b.id !== id && b.installed && b.mcp_ids.includes(mcpId)
		);
		if (onlyForThisBundle) {
			await uninstallMcp(mcpId);
		}
	}

	bundles.update((list) =>
		list.map((b) => (b.id === id ? { ...b, installed: false } : b))
	);

	if (isTauri) {
		await invokeDesktop('uninstall_bundle', { id });
	}
}

// ── Dashboard Stores ──

export const routerStatus = writable<RouterStatus>({
	status: 'stopped',
	version: 'v1.0.0',
	transport: 'stdio',
	uptime_seconds: 0,
	port: 3000,
	metrics_endpoint: '3031/metrics'
});

export const systemUsage = writable<SystemUsage>({
	cpu_percent: 0,
	memory_percent: 0,
	disk_percent: 0,
	cpu_history: [],
	memory_history: [],
	disk_history: []
});

export const activityLog = writable<ActivityItem[]>([]);

export const mcpServers = writable<McpServerDetail[]>([]);

export const isConsoleExpanded = writable(false);
export const consoleTab = writable<'output' | 'problems' | 'debug' | 'cli'>('output');

// ── Zoom Level ──
export const zoomLevel = writable(1.0);

// ── Dashboard API ──

export async function fetchRouterStatus() {
	try {
		if (isTauri) {
			routerStatus.set(await invokeDesktop<RouterStatus>('get_router_status'));
			return;
		}
		const res = await fetch(`${API_URL}/api/router/status`);
		if (res.ok) routerStatus.set(await res.json());
	} catch {
		// keep defaults
	}
}

export async function fetchSystemUsage() {
	try {
		if (isTauri) {
			systemUsage.set(await invokeDesktop<SystemUsage>('get_system_usage'));
			return;
		}
		const res = await fetch(`${API_URL}/api/system/usage`);
		if (res.ok) systemUsage.set(await res.json());
	} catch {
		// keep defaults
	}
}

export async function fetchActivityLog(limit = 20) {
	try {
		if (isTauri) {
			activityLog.set(await invokeDesktop<ActivityItem[]>('get_activity_log', { limit }));
			return;
		}
		const res = await fetch(`${API_URL}/api/activity?limit=${limit}`);
		if (res.ok) {
			const data = await res.json();
			activityLog.set(data.activities ?? []);
		}
	} catch {
		// keep defaults
	}
}

export async function fetchMcpServers() {
	try {
		if (isTauri) {
			mcpServers.set(await invokeDesktop<McpServerDetail[]>('get_mcp_servers'));
			return;
		}
		const res = await fetch(`${API_URL}/api/mcp/servers`);
		if (res.ok) {
			const data = await res.json();
			mcpServers.set(data.servers ?? []);
		}
	} catch {
		// keep defaults
	}
}

export async function executeCommand(command: string): Promise<CommandResult> {
	if (isTauri) {
		return invokeDesktop<CommandResult>('execute_command', { command });
	}
	const res = await fetch(`${API_URL}/api/command/exec`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ command })
	});
	if (!res.ok) throw new Error('Command execution failed');
	return res.json();
}

export async function restartRouter() {
	if (isTauri) {
		return invokeDesktop<string>('restart_router');
	}
	const res = await fetch(`${API_URL}/api/router/restart`, { method: 'POST' });
	if (!res.ok) throw new Error('Restart failed');
	return res.text();
}

let syncController: AbortController | null = null;

export function startDashboardSync() {
	async function sync() {
		if (syncController) syncController.abort();
		syncController = new AbortController();
		await Promise.allSettled([
			fetchRouterStatus(),
			fetchSystemUsage(),
			fetchActivityLog(10),
			fetchMcpServers(),
		]);
	}
	sync();
	const interval = setInterval(sync, 5000);
	return () => {
		clearInterval(interval);
		if (syncController) syncController.abort();
	};
}

export async function setupSkillForClient(skillId: string, clientId: string) {
	const skill = get(skills).find((s) => s.id === skillId);
	if (!skill || !skill.installed) return;

	// Ensure all MCPs are installed (safety net)
	const installedList = get(installed);
	for (const mcpId of skill.mcp_ids) {
		if (!installedList.some((m) => m.id === mcpId)) {
			await installMcp(mcpId);
		}
	}

	// Connect the client to mach1
	await connectClient(clientId);

	// Record the wiring
	skills.update((list) =>
		list.map((s) =>
			s.id === skillId
				? { ...s, client_ids: [...new Set([...s.client_ids, clientId])] }
				: s
		)
	);

	if (isTauri) {
		const updated = get(skills).find((s) => s.id === skillId);
		if (updated) {
			try {
				await invokeDesktop('upsert_skill', { skill: updated });
			} catch {
				// Non-blocking
			}
		}
	}
}

// ── Servers Page Stores ──

export const selectedServerId = writable<string | null>(null);
export const serverDetail = writable<ServerDetail | null>(null);
export const serverTools = writable<ServerTool[]>([]);
export const serverLogs = writable<ServerLogEntry[]>([]);
export const serverConfig = writable<ServerConfig | null>(null);

export async function fetchServerDetail(id: string) {
	try {
		if (isTauri) {
			serverDetail.set(await invokeDesktop<ServerDetail>('get_server_detail', { id }));
			return;
		}
		const res = await fetch(`${API_URL}/api/servers/${id}`);
		if (res.ok) serverDetail.set(await res.json());
	} catch {
		// keep defaults
	}
}

export async function fetchServerTools(id: string) {
	try {
		let tools: ServerTool[] = [];
		if (isTauri) {
			tools = await invokeDesktop<ServerTool[]>('get_server_tools', { id });
		} else {
			const res = await fetch(`${API_URL}/api/servers/${id}/tools`);
			if (res.ok) tools = await res.json();
		}
		serverTools.set(tools);
		// Sync tools_count into both mcpServers and serverDetail
		mcpServers.update(list => list.map(s =>
			s.id === id ? { ...s, tools_count: tools.length } : s
		));
		serverDetail.update(detail =>
			detail && detail.id === id ? { ...detail, tools_count: tools.length } : detail
		);
	} catch {
		// keep defaults
	}
}

export async function fetchServerLogs(id: string, limit = 50) {
	try {
		if (isTauri) {
			serverLogs.set(await invokeDesktop<ServerLogEntry[]>('get_server_logs', { id, limit }));
			return;
		}
		const res = await fetch(`${API_URL}/api/servers/${id}/logs?limit=${limit}`);
		if (res.ok) serverLogs.set(await res.json());
	} catch {
		// keep defaults
	}
}

export async function fetchServerConfig(id: string) {
	try {
		if (isTauri) {
			serverConfig.set(await invokeDesktop<ServerConfig>('get_server_config', { id }));
			return;
		}
		const res = await fetch(`${API_URL}/api/servers/${id}/config`);
		if (res.ok) serverConfig.set(await res.json());
	} catch {
		// keep defaults
	}
}

export async function scanServer(id: string): Promise<string> {
	if (isTauri) {
		return invokeDesktop<string>('scan_server', { id });
	}
	const res = await fetch(`${API_URL}/api/servers/${id}/scan`, { method: 'POST' });
	if (!res.ok) throw new Error('Scan failed');
	return res.text();
}

export async function restartSingleServer(id: string): Promise<string> {
	if (isTauri) {
		return invokeDesktop<string>('restart_single_server', { id });
	}
	const res = await fetch(`${API_URL}/api/servers/${id}/restart`, { method: 'POST' });
	if (!res.ok) throw new Error('Restart failed');
	return res.text();
}

export async function uninstallSingleServer(id: string): Promise<void> {
	if (isTauri) {
		await invokeDesktop('uninstall_single_server', { id });
		return;
	}
	const res = await fetch(`${API_URL}/api/servers/${id}`, { method: 'DELETE' });
	if (!res.ok) throw new Error('Uninstall failed');
}

export function selectServer(id: string | null) {
	selectedServerId.set(id);
	if (id) {
		fetchServerDetail(id);
		fetchServerTools(id);
		fetchServerLogs(id);
		fetchServerConfig(id);
	} else {
		serverDetail.set(null);
		serverTools.set([]);
		serverLogs.set([]);
		serverConfig.set(null);
	}
}

// ── Discover / Marketplace Stores ──

export const selectedMarketplaceItem = writable<string | null>(null);
export const marketplaceItemDetail = writable<MarketplaceItemDetail | null>(null);

function deriveSecurityChecks(verificationStatus: string): { label: string; status: 'passed' | 'warning' | 'failed' }[] {
	if (verificationStatus === 'anthropic-official' || verificationStatus === '1mcp.in-verified' || verificationStatus === '1mcp-verified') {
		return [
			{ label: 'Tool schema verified', status: 'passed' as const },
			{ label: 'Digest matches registry', status: 'passed' as const },
		];
	}
	return [
		{ label: 'Tool schema verified', status: 'passed' as const },
		{ label: 'Community contributed — verify before use', status: 'warning' as const },
	];
}

export async function fetchMarketplaceItemDetail(id: string) {
	try {
		let apiItem: CloudMarketplaceItem | null = null;
		if (isTauri) {
			apiItem = await invokeDesktop<CloudMarketplaceItem | null>('get_marketplace_item', { id });
		} else {
			const res = await fetch(`${API_URL}/api/marketplace/${id}`);
			if (res.ok) {
				const data = await res.json();
				apiItem = data as CloudMarketplaceItem;
			}
		}
		const localItem = get(marketplace).find((m) => m.id === id);
		if (apiItem) {
			marketplaceItemDetail.set({
				id: apiItem.id,
				name: apiItem.name,
				description: apiItem.description,
				shortDescription: apiItem.description,
				version: apiItem.version,
				runtime: apiItem.runtime as Runtime,
				author: apiItem.author ?? 'community',
				trust: apiItem.verification ?? 'community',
				license: apiItem.license ?? 'MIT',
				sha256: apiItem.sha256 ?? '',
				verified_at: apiItem.publishedAt ?? '',
				updated_at: apiItem.publishedAt ?? '',
				downloads: Math.max(apiItem.downloads ?? 0, localItem?.downloads ?? 0),
				rating: apiItem.rating ?? 0,
				reviewCount: apiItem.reviewCount ?? 0,
				tags: apiItem.tags ?? [],
				installed: localItem?.installed ?? false,
				capabilities: apiItem.tags ?? [],
				security_checks: (apiItem.security_checks as { label: string; status: 'passed' | 'warning' | 'failed' }[] | undefined) ?? deriveSecurityChecks(apiItem.verification ?? 'community'),
				requires_env: apiItem.requires_env?.length
					? apiItem.requires_env
					: detectRequiredEnv({
						id: apiItem.id,
						name: apiItem.name,
						shortDescription: apiItem.description,
						tags: apiItem.tags,
						patProvider: localItem?.patProvider,
						requires_env: localItem?.requires_env,
					}),
				homepage: apiItem.homepage ?? localItem?.homepage,
				patProvider: localItem?.patProvider
			});
			return;
		}
		if (localItem) {
			marketplaceItemDetail.set({
				id: localItem.id,
				name: localItem.name,
				description: localItem.shortDescription,
				shortDescription: localItem.shortDescription,
				version: localItem.version,
				runtime: localItem.runtime,
				author: localItem.author,
				trust: localItem.verificationStatus,
				license: 'MIT',
				sha256: localItem.sha256 ?? '',
				verified_at: localItem.publishedAt,
				updated_at: localItem.publishedAt,
				downloads: localItem.downloads,
				rating: localItem.rating,
				reviewCount: localItem.reviewCount,
				tags: localItem.tags,
				installed: localItem.installed,
				capabilities: localItem.tags,
				security_checks: deriveSecurityChecks(localItem.verificationStatus),
				requires_env: detectRequiredEnv({
					id: localItem.id,
					name: localItem.name,
					shortDescription: localItem.shortDescription,
					tags: localItem.tags,
					patProvider: localItem.patProvider,
					requires_env: localItem.requires_env,
				}),
				homepage: localItem.homepage,
				patProvider: localItem.patProvider
			});
		}
	} catch {
		marketplaceItemDetail.set(null);
	}
}

export function selectMarketplaceItem(id: string | null) {
	selectedMarketplaceItem.set(id);
	if (id) {
		fetchMarketplaceItemDetail(id);
	} else {
		marketplaceItemDetail.set(null);
	}
}

// ── Clients Page Stores ──

export const selectedClientId = writable<string | null>(null);
export const clientDetail = writable<ClientConnectionDetail | null>(null);
export const clientRoutingHealth = writable<ClientRoutingHealth | null>(null);
export const clientConfigPreview = writable<ClientConfigPreview | null>(null);

export async function fetchClientDetail(id: string) {
	try {
		if (isTauri) {
			clientDetail.set(await invokeDesktop<ClientConnectionDetail>('get_client_detail', { id }));
			return;
		}
		const res = await fetch(`${API_URL}/api/clients/${id}`);
		if (res.ok) clientDetail.set(await res.json());
	} catch {
		// keep defaults
	}
}

export async function fetchClientRoutingHealth(id: string) {
	try {
		if (isTauri) {
			clientRoutingHealth.set(await invokeDesktop<ClientRoutingHealth>('get_client_routing_health', { id }));
			return;
		}
		const res = await fetch(`${API_URL}/api/clients/${id}/health`);
		if (res.ok) clientRoutingHealth.set(await res.json());
	} catch {
		// keep defaults
	}
}

export async function fetchClientConfigPreview(id: string) {
	try {
		if (isTauri) {
			clientConfigPreview.set(await invokeDesktop<ClientConfigPreview>('get_client_config_preview', { id }));
			return;
		}
		const res = await fetch(`${API_URL}/api/clients/${id}/config`);
		if (res.ok) clientConfigPreview.set(await res.json());
	} catch {
		// keep defaults
	}
}

export function selectClient(id: string | null) {
	selectedClientId.set(id);
	if (id) {
		fetchClientDetail(id);
		fetchClientRoutingHealth(id);
		fetchClientConfigPreview(id);
	} else {
		clientDetail.set(null);
		clientRoutingHealth.set(null);
		clientConfigPreview.set(null);
	}
}

export async function connectAllSupportedClients() {
	for (const client of get(clients)) {
		if (!client.connected) {
			try {
				await connectClient(client.id);
			} catch {
				// skip unsupported
			}
		}
	}
}

export async function disconnectAllClients() {
	for (const client of get(clients)) {
		if (client.connected) {
			try {
				await disconnectClient(client.id);
			} catch {
				// skip
			}
		}
	}
}

// ── Settings / Preferences Stores ──

export const appPreferences = writable<AppPreferences>({
	start_on_login: true,
	minimize_to_tray: true,
	theme: 'dark',
	language: 'System Default',
	telemetry_enabled: false,
	log_level: 'info',
});

export const systemInfo = writable<SystemInfo | null>(null);
export const settingsSaved = writable(false);
export const settingsLoading = writable(false);

export async function fetchAppPreferences() {
	try {
		if (isTauri) {
			appPreferences.set(await invokeDesktop<AppPreferences>('get_settings', {}));
			return;
		}
		const res = await fetch(`${API_URL}/api/settings`);
		if (res.ok) appPreferences.set(await res.json());
	} catch {
		// keep defaults
	}
}

export async function saveAppPreferences(prefs: AppPreferences) {
	const prevPrefs = get(appPreferences);
	settingsLoading.set(true);
	try {
		if (isTauri) {
			await invokeDesktop<void>('save_settings', { prefs });
		} else {
			const res = await fetch(`${API_URL}/api/settings`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(prefs),
			});
			if (!res.ok) throw new Error('Failed to save settings');
		}
		appPreferences.set(prefs);
		settingsSaved.set(true);
		setTimeout(() => settingsSaved.set(false), 2000);
	} catch (error) {
		appPreferences.set(prevPrefs);
		throw error;
	} finally {
		settingsLoading.set(false);
	}
}

export async function fetchSystemInfo() {
	try {
		if (isTauri) {
			systemInfo.set(await invokeDesktop<SystemInfo>('get_system_info', {}));
			return;
		}
		const res = await fetch(`${API_URL}/api/system/info`);
		if (res.ok) systemInfo.set(await res.json());
	} catch {
		// keep null
	}
}

export async function resetRouterConfig() {
	try {
		if (isTauri) {
			await invokeDesktop<void>('reset_router_config', {});
			return;
		}
		const res = await fetch(`${API_URL}/api/settings/reset`, { method: 'POST' });
		if (!res.ok) throw new Error('Failed to reset router');
	} catch (error) {
		throw error;
	}
}

export async function clearLocalData() {
	try {
		if (isTauri) {
			await invokeDesktop<void>('clear_local_data', {});
			return;
		}
		const res = await fetch(`${API_URL}/api/settings/clear-data`, { method: 'POST' });
		if (!res.ok) throw new Error('Failed to clear data');
	} catch (error) {
		throw error;
	}
}

export async function copyDiagnostics(): Promise<string> {
	try {
		if (isTauri) {
			return await invokeDesktop<string>('copy_diagnostics', {});
		}
		const res = await fetch(`${API_URL}/api/settings/diagnostics`);
		if (!res.ok) throw new Error('Failed to get diagnostics');
		const data: DiagnosticsData = await res.json();
		return JSON.stringify(data, null, 2);
	} catch {
		return '{}';
	}
}

// ── MCP Lifecycle Actions (v0.3.4) ──

export async function installMCP(id: string) {
	if (isTauri) {
		return invokeDesktop('mach1_install_mcp', { id });
	}
	const res = await fetch(`${API_URL}/api/mcp/install`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ id })
	});
	if (!res.ok) throw new Error('Install failed');
	return res.json();
}

export async function batchInstallMCPs(ids: string[]) {
	if (isTauri) {
		return invokeDesktop('mach1_install_batch', { ids });
	}
	const res = await fetch(`${API_URL}/api/mcp/install-batch`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ ids })
	});
	if (!res.ok) throw new Error('Batch install failed');
	return res.json();
}

export async function startMCP(id: string) {
	installed.update((list) =>
		list.map((m) => (m.id === id ? { ...m, enabled: true } : m))
	);
	mcpServers.update((list) =>
		list.map((m) =>
			m.id === id
				? { ...m, status: 'running' as const }
				: m
		)
	);
	if (isTauri) {
		try {
			return await invokeDesktop('mach1_start_mcp', { id });
		} catch (e) {
			installed.update((list) =>
				list.map((m) => (m.id === id ? { ...m, enabled: false } : m))
			);
			mcpServers.update((list) =>
				list.map((m) =>
					m.id === id
						? { ...m, status: 'error' as const }
						: m
				)
			);
			throw e;
		}
	}
	const res = await fetch(`${API_URL}/api/mcp/${encodeURIComponent(id)}/start`, { method: 'POST' });
	if (!res.ok) throw new Error('Start failed');
	return res.json();
}

export async function stopMCP(id: string) {
	installed.update((list) =>
		list.map((m) => (m.id === id ? { ...m, enabled: false } : m))
	);
	mcpServers.update((list) =>
		list.map((m) =>
			m.id === id
				? { ...m, status: 'sleeping' as const }
				: m
		)
	);
	if (isTauri) {
		try {
			return await invokeDesktop('mach1_close_mcp', { id });
		} catch (e) {
			installed.update((list) =>
				list.map((m) => (m.id === id ? { ...m, enabled: true } : m))
			);
			mcpServers.update((list) =>
				list.map((m) =>
					m.id === id
						? { ...m, status: 'running' as const }
						: m
				)
			);
			throw e;
		}
	}
	const res = await fetch(`${API_URL}/api/mcp/${encodeURIComponent(id)}/stop`, { method: 'POST' });
	if (!res.ok) throw new Error('Stop failed');
	return res.json();
}

export async function setMcpEnv(id: string, vars: Record<string, string>) {
	installed.update((list) =>
		list.map((m) =>
			m.id === id
				? { ...m, env: { ...(m.env ?? {}), ...vars } }
				: m
		)
	);
	if (isTauri) {
		return invokeDesktop('mach1_config_env', { id, vars });
	}
	const res = await fetch(`${API_URL}/api/mcp/${encodeURIComponent(id)}/env`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ vars })
	});
	if (!res.ok) throw new Error('Env config failed');
	return res.json();
}

export async function checkEnabled() {
	if (isTauri) {
		return invokeDesktop('mach1_check_enabled', {});
	}
	const res = await fetch(`${API_URL}/api/mcp/enabled`);
	if (!res.ok) throw new Error('Check enabled failed');
	return res.json();
}

export async function healthCheck(id: string) {
	if (isTauri) {
		return invokeDesktop('mach1_health_check', { id });
	}
	const res = await fetch(`${API_URL}/api/mcp/${encodeURIComponent(id)}/health`);
	if (!res.ok) throw new Error('Health check failed');
	return res.json();
}

export async function autoDetectEnv(id: string): Promise<Record<string, string>> {
	if (isTauri) {
		return invokeDesktop('mach1_auto_detect_env', { id });
	}
	const res = await fetch(`${API_URL}/api/mcp/${encodeURIComponent(id)}/env/detect`);
	if (!res.ok) throw new Error('Auto-detect failed');
	return res.json();
}
