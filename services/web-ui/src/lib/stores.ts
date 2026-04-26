import { writable, derived, get } from 'svelte/store';
import type { InstalledMcp, MarketplaceMcp, ClientApp } from './types';

// ─── Installed MCPs ─────────────────────────────────────
export const installed = writable<InstalledMcp[]>([
	{
		id: 'github',
		name: 'GitHub',
		version: '0.6.2',
		runtime: 'node',
		enabled: true,
		command: 'npx -y @modelcontextprotocol/server-github',
		description: 'Search code, read issues/PRs, and create issues on GitHub via the GitHub API.',
		patProvider: 'github'
	},
	{
		id: 'memory',
		name: 'Knowledge Graph Memory',
		version: '0.6.0',
		runtime: 'node',
		enabled: true,
		command: 'npx -y @modelcontextprotocol/server-memory',
		description: 'Persistent knowledge graph the agent can query and update across sessions.'
	}
]);

// ─── Marketplace ────────────────────────────────────────
export const marketplace = writable<MarketplaceMcp[]>([
	{
		id: 'github', name: 'GitHub',
		shortDescription: 'Search code, read issues/PRs, and create issues on GitHub via the GitHub API.',
		version: '0.6.2', runtime: 'node', author: 'Anthropic',
		tags: ['github', 'git', 'issues', 'official'],
		rating: 4.9, reviewCount: 634, downloads: 92800,
		verificationStatus: 'verified', publishedAt: '2024-11-05', installed: true, patProvider: 'github'
	},
	{
		id: 'memory', name: 'Knowledge Graph Memory',
		shortDescription: 'Persistent knowledge graph the agent can query and update across sessions.',
		version: '0.6.0', runtime: 'node', author: 'Anthropic',
		tags: ['memory', 'knowledge-graph', 'official'],
		rating: 4.9, reviewCount: 521, downloads: 78300,
		verificationStatus: 'verified', publishedAt: '2024-10-15', installed: true
	},
	{
		id: 'filesystem', name: 'Filesystem',
		shortDescription: 'Read, write, and search files within allow-listed directories.',
		version: '0.6.2', runtime: 'node', author: 'Anthropic',
		tags: ['filesystem', 'files', 'official'],
		rating: 4.8, reviewCount: 312, downloads: 45200,
		verificationStatus: 'verified', publishedAt: '2024-11-01', installed: false
	},
	{
		id: 'fetch', name: 'Fetch',
		shortDescription: 'Fetch URLs and convert HTML to markdown for agent consumption.',
		version: '0.6.0', runtime: 'python', author: 'Anthropic',
		tags: ['http', 'fetch', 'web', 'official'],
		rating: 4.7, reviewCount: 188, downloads: 31000,
		verificationStatus: 'verified', publishedAt: '2024-10-20', installed: false
	},
	{
		id: 'git', name: 'Git',
		shortDescription: 'Read-only git repository inspection (log, diff, blame).',
		version: '0.6.0', runtime: 'python', author: 'Anthropic',
		tags: ['git', 'vcs', 'official'],
		rating: 4.6, reviewCount: 143, downloads: 22100,
		verificationStatus: 'verified', publishedAt: '2024-10-25', installed: false
	},
	{
		id: 'postgres', name: 'PostgreSQL',
		shortDescription: 'Query and inspect PostgreSQL databases with safe read-only access.',
		version: '1.0.0', runtime: 'node', author: 'db-tools',
		tags: ['database', 'postgres', 'sql'],
		rating: 4.5, reviewCount: 119, downloads: 17600,
		verificationStatus: 'verified', publishedAt: '2025-02-10', installed: false
	},
	{
		id: 'slack', name: 'Slack',
		shortDescription: 'Read and send Slack messages, manage channels and workspace users.',
		version: '1.2.0', runtime: 'node', author: 'community',
		tags: ['slack', 'messaging', 'communication'],
		rating: 4.2, reviewCount: 87, downloads: 9400,
		verificationStatus: 'unverified', publishedAt: '2025-01-12', installed: false
	},
	{
		id: 'jira', name: 'Jira',
		shortDescription: 'Create and manage Jira issues, sprints, and project boards.',
		version: '0.9.1', runtime: 'python', author: 'atlassian-community',
		tags: ['jira', 'project-management', 'atlassian'],
		rating: 3.8, reviewCount: 52, downloads: 6100,
		verificationStatus: 'pending', publishedAt: '2025-03-01', installed: false
	},
	{
		id: 'linear', name: 'Linear',
		shortDescription: 'Manage Linear issues, cycles, and projects from your AI assistant.',
		version: '0.3.0', runtime: 'node', author: 'linear-community',
		tags: ['linear', 'project-management', 'issues'],
		rating: 4.1, reviewCount: 38, downloads: 4200,
		verificationStatus: 'unverified', publishedAt: '2025-04-01', installed: false,
		patProvider: 'linear'
	}
]);

// ─── Client Apps ────────────────────────────────────────
export const clients = writable<ClientApp[]>([
	{
		id: 'vscode',
		name: 'VS Code',
		icon: '💻',
		description: 'GitHub Copilot, Roo Code, Continue — via mcp.json',
		connected: false,
		connectCommand: 'connect vscode'
	},
	{
		id: 'cursor',
		name: 'Cursor',
		icon: '⚡',
		description: 'Cursor AI IDE — auto-configure mcpServers',
		connected: false,
		connectCommand: 'connect cursor'
	},
	{
		id: 'claude',
		name: 'Claude Desktop',
		icon: '🤖',
		description: 'Anthropic Claude Desktop — auto-configure mcpServers',
		connected: false,
		connectCommand: 'connect claude'
	},
	{
		id: 'claudecode',
		name: 'Claude Code',
		icon: '🖥️',
		description: 'Claude Code CLI — terminal-based AI agent',
		connected: false,
		connectCommand: 'connect claudecode'
	}
]);

// ─── User counter (base 1000 + real) ────────────────────
export const userCount = writable(1000);

let counterInterval: ReturnType<typeof setInterval> | null = null;

export function startUserCounter() {
	if (counterInterval) return;
	counterInterval = setInterval(() => {
		userCount.update((n) => n + Math.floor(Math.random() * 3));
	}, 4000);
}

export function stopUserCounter() {
	if (counterInterval) {
		clearInterval(counterInterval);
		counterInterval = null;
	}
}

// ─── Derived stats ──────────────────────────────────────
export const installedCount = derived(installed, ($i) => $i.length);
export const runningCount = derived(installed, ($i) => $i.filter((m) => m.enabled).length);

// ─── Actions ────────────────────────────────────────────
export function toggleMcp(id: string) {
	installed.update((list) =>
		list.map((m) => (m.id === id ? { ...m, enabled: !m.enabled } : m))
	);
}

export function uninstallMcp(id: string) {
	installed.update((list) => list.filter((m) => m.id !== id));
	marketplace.update((list) =>
		list.map((m) => (m.id === id ? { ...m, installed: false } : m))
	);
}

export function installMcp(id: string) {
	const mkt = get(marketplace);
	const mcp = mkt.find((m) => m.id === id);
	const inst = get(installed);
	if (!mcp || inst.some((m) => m.id === id)) return;

	installed.update((list) => [
		...list,
		{
			id: mcp.id,
			name: mcp.name,
			version: mcp.version,
			runtime: mcp.runtime,
			enabled: true,
			command: `npx -y @modelcontextprotocol/server-${mcp.id}`,
			description: mcp.shortDescription,
			patProvider: mcp.patProvider
		}
	]);
	marketplace.update((list) =>
		list.map((m) => (m.id === id ? { ...m, installed: true } : m))
	);
}

export function connectClient(id: string) {
	clients.update((list) =>
		list.map((c) => (c.id === id ? { ...c, connected: true } : c))
	);
}

export function disconnectClient(id: string) {
	clients.update((list) =>
		list.map((c) => (c.id === id ? { ...c, connected: false } : c))
	);
}
