import type { RouterStatus, SystemUsage, ActivityItem, McpServerDetail, ClientApp, MarketplaceMcp, Skill, InstalledMcp, User } from '../../src/lib/types';

export const seedUser: User = {
	id: 'user-1',
	name: 'Test User',
	email: 'test@example.com',
};

export const seedRouterStatus: RouterStatus = {
	status: 'running',
	version: 'v1.0.0',
	transport: 'stdio',
	uptime_seconds: 86400,
	port: 3000,
	metrics_endpoint: '3031/metrics',
};

export const seedSystemUsage: SystemUsage = {
	cpu_percent: 23,
	memory_percent: 45,
	disk_percent: 62,
	cpu_history: [20, 25, 22, 23, 21, 24, 23],
	memory_history: [40, 42, 44, 43, 45, 44, 45],
	disk_history: [60, 61, 62, 62, 62, 62, 62],
};

export const seedActivityLog: ActivityItem[] = [
	{ id: 'a1', type: 'router_started', message: 'mach1 router started', timestamp: new Date(Date.now() - 60000).toISOString(), icon: '▶' },
	{ id: 'a2', type: 'mcp_started', message: 'GitHub MCP started', timestamp: new Date(Date.now() - 120000).toISOString(), icon: '📦' },
	{ id: 'a3', type: 'client_connected', message: 'VS Code connected', timestamp: new Date(Date.now() - 300000).toISOString(), icon: '🔗' },
];

export const seedMcpServers: McpServerDetail[] = [
	{
		id: 'mach1', name: 'Mach1 Router', description: 'Semantic router for 1mcp.in',
		version: '1.0.0', runtime: 'binary', status: 'running', status_detail: 'PID 21340',
		lifecycle: 'Manual', trust: 'internal', author: '1mcp.in', last_used_at: null,
		tools_count: 5, installed_at: new Date().toISOString(),
		process: { pid: 21340, memory_mb: 64.2, cpu_percent: 0.3, uptime_seconds: 86400, restarts: 0 },
	},
	{
		id: 'github', name: 'GitHub', description: 'Search code, read issues/PRs, create issues.',
		version: '0.6.2', runtime: 'node', status: 'running', status_detail: 'PID 21500',
		lifecycle: 'Auto (lazy)', trust: 'anthropic-official', author: 'Anthropic',
		idle_timeout: '15 minutes', last_used_at: new Date(Date.now() - 300000).toISOString(),
		tools_count: 37, installed_at: new Date(Date.now() - 86400000).toISOString(),
		process: { pid: 21500, memory_mb: 128.5, cpu_percent: 1.2, uptime_seconds: 43200, restarts: 1 },
	},
	{
		id: 'memory', name: 'Knowledge Graph Memory', description: 'Persistent knowledge graph.',
		version: '0.6.0', runtime: 'node', status: 'running', status_detail: 'PID 21501',
		lifecycle: 'Auto (lazy)', trust: 'anthropic-official', author: 'Anthropic',
		idle_timeout: '15 minutes', last_used_at: new Date(Date.now() - 600000).toISOString(),
		tools_count: 15, installed_at: new Date(Date.now() - 86400000).toISOString(),
		process: { pid: 21501, memory_mb: 92.1, cpu_percent: 0.8, uptime_seconds: 43200, restarts: 0 },
	},
	{
		id: 'filesystem', name: 'Filesystem', description: 'Read, write, search files.',
		version: '0.6.2', runtime: 'node', status: 'sleeping', status_detail: 'Idle',
		lifecycle: 'Auto (lazy)', trust: 'anthropic-official', author: 'Anthropic',
		idle_timeout: '15 minutes', last_used_at: new Date(Date.now() - 7200000).toISOString(),
		tools_count: 10, installed_at: new Date(Date.now() - 172800000).toISOString(),
	},
	{
		id: 'postgres', name: 'PostgreSQL', description: 'Query and inspect PostgreSQL databases.',
		version: '1.0.0', runtime: 'node', status: 'error', status_detail: 'Connection refused',
		lifecycle: 'Auto (lazy)', trust: '1mcp.in-verified', author: 'db-tools',
		idle_timeout: '15 minutes', last_used_at: new Date(Date.now() - 86400000).toISOString(),
		tools_count: 12, installed_at: new Date(Date.now() - 259200000).toISOString(),
	},
];

export const seedClients: ClientApp[] = [
	{ id: 'vscode', name: 'VS Code', icon: '<>', description: 'GitHub Copilot, Roo Code, Continue via mcp.json', connected: true, connectCommand: 'connect vscode' },
	{ id: 'cursor', name: 'Cursor', icon: '⌘', description: 'Cursor AI IDE via ~/.cursor/mcp.json', connected: true, connectCommand: 'connect cursor' },
	{ id: 'claude', name: 'Claude Desktop', icon: '🤖', description: 'Anthropic Claude Desktop via claude_desktop_config.json', connected: false, connectCommand: 'connect claude' },
	{ id: 'claudecode', name: 'Claude Code', icon: '⌨', description: 'Claude Code CLI via ~/.claude.json', connected: false, connectCommand: 'connect claudecode' },
	{ id: 'windsurf', name: 'Windsurf', icon: '🌊', description: 'Windsurf Cascade IDE via ~/.codeium/mcp_config.json', connected: false, connectCommand: 'connect windsurf' },
	{ id: 'codex', name: 'Codex', icon: '🧠', description: 'Codex AI Assistant via ~/.codex/config.toml', connected: false, connectCommand: 'connect codex' },
	{ id: 'antigravity', name: 'Antigravity', icon: '🚀', description: 'Antigravity agent integration', connected: false, connectCommand: 'connect antigravity' },
	{ id: 'opencode', name: 'OpenCode', icon: '{ }', description: 'OpenCode IDE via ~/.config/opencode/opencode.json', connected: false, connectCommand: 'connect opencode' },
];

export const seedMarketplace: MarketplaceMcp[] = [
	{ id: 'github', name: 'GitHub', shortDescription: 'Search code, read issues/PRs, create issues.', version: '0.6.2', runtime: 'node', author: 'Anthropic', tags: ['github', 'git', 'issues', 'official'], rating: 4.9, reviewCount: 634, downloads: 92800, verificationStatus: 'anthropic-official', publishedAt: '2024-11-05', installed: true },
	{ id: 'memory', name: 'Knowledge Graph Memory', shortDescription: 'Persistent knowledge graph.', version: '0.6.0', runtime: 'node', author: 'Anthropic', tags: ['memory', 'knowledge-graph', 'official'], rating: 4.9, reviewCount: 521, downloads: 78300, verificationStatus: 'anthropic-official', publishedAt: '2024-10-15', installed: true },
	{ id: 'filesystem', name: 'Filesystem', shortDescription: 'Read, write, and search files.', version: '0.6.2', runtime: 'node', author: 'Anthropic', tags: ['filesystem', 'files', 'official'], rating: 4.8, reviewCount: 312, downloads: 45200, verificationStatus: 'anthropic-official', publishedAt: '2024-11-01', installed: false },
	{ id: 'fetch', name: 'Fetch', shortDescription: 'Fetch URLs and convert HTML to markdown.', version: '0.6.0', runtime: 'python', author: 'Anthropic', tags: ['http', 'fetch', 'web', 'official'], rating: 4.7, reviewCount: 188, downloads: 31000, verificationStatus: 'anthropic-official', publishedAt: '2024-10-20', installed: false },
	{ id: 'git', name: 'Git', shortDescription: 'Read-only git repository inspection.', version: '0.6.0', runtime: 'python', author: 'Anthropic', tags: ['git', 'vcs', 'official'], rating: 4.6, reviewCount: 143, downloads: 22100, verificationStatus: 'anthropic-official', publishedAt: '2024-10-25', installed: false },
	{ id: 'postgres', name: 'PostgreSQL', shortDescription: 'Query and inspect PostgreSQL databases.', version: '1.0.0', runtime: 'node', author: 'db-tools', tags: ['database', 'postgres', 'sql'], rating: 4.5, reviewCount: 119, downloads: 17600, verificationStatus: '1mcp.in-verified', publishedAt: '2025-02-10', installed: false },
	{ id: 'slack', name: 'Slack', shortDescription: 'Read and send Slack messages.', version: '1.2.0', runtime: 'node', author: 'community', tags: ['slack', 'messaging', 'communication'], rating: 4.2, reviewCount: 87, downloads: 9400, verificationStatus: 'community', publishedAt: '2025-01-12', installed: false },
	{ id: 'jira', name: 'Jira', shortDescription: 'Create and manage Jira issues.', version: '0.9.1', runtime: 'python', author: 'atlassian-community', tags: ['jira', 'project-management', 'atlassian'], rating: 3.8, reviewCount: 52, downloads: 6100, verificationStatus: 'community', publishedAt: '2025-03-01', installed: false },
];

export const seedSkills: Skill[] = [
	{ id: 'frontend-dev', name: 'Frontend Developer', description: 'GitHub, filesystem, and memory for frontend workflows', icon: '🎨', mcp_ids: ['github', 'filesystem', 'memory'], client_ids: [], installed: true, enabled: true, created_at: 0 },
	{ id: 'backend-dev', name: 'Backend Developer', description: 'GitHub, Postgres, and fetch for backend and API work', icon: '⚙️', mcp_ids: ['github', 'postgres', 'fetch'], client_ids: [], installed: false, enabled: true, created_at: 0 },
	{ id: 'devops', name: 'DevOps', description: 'GitHub, fetch, and memory for infrastructure and deployment', icon: '🛠️', mcp_ids: ['github', 'fetch', 'memory'], client_ids: [], installed: false, enabled: true, created_at: 0 },
	{ id: 'writer', name: 'Writer', description: 'Fetch, memory, and filesystem for research and content creation', icon: '✍️', mcp_ids: ['fetch', 'memory', 'filesystem'], client_ids: [], installed: false, enabled: true, created_at: 0 },
	{ id: 'full-stack', name: 'Full Stack', description: 'GitHub, filesystem, memory, and fetch for end-to-end development', icon: '🚀', mcp_ids: ['github', 'filesystem', 'memory', 'fetch'], client_ids: [], installed: false, enabled: true, created_at: 0 },
];

export const seedInstalled: InstalledMcp[] = [
	{ id: 'mach1', name: 'Mach1 Router', version: '1.0.0', runtime: 'binary', enabled: true, command: 'mach1', args: [], description: 'Semantic router for 1mcp.in.' },
	{ id: 'github', name: 'GitHub', version: '0.6.2', runtime: 'node', enabled: true, command: 'npx', args: ['-y', '@modelcontextprotocol/server-github'], description: 'Search code, read issues/PRs.', patProvider: 'github' },
	{ id: 'memory', name: 'Knowledge Graph Memory', version: '0.6.0', runtime: 'node', enabled: true, command: 'npx', args: ['-y', '@modelcontextprotocol/server-memory'], description: 'Persistent knowledge graph.' },
];
