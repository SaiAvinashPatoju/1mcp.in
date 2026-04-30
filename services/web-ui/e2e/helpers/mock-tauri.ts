import type { Page } from '@playwright/test';

export async function setupMockTauri(page: Page) {
	await page.addInitScript(() => {
		const handlers: Record<string, (args?: Record<string, unknown>) => unknown> = {
			auth_me: () => ({ id: 'mock-user', name: 'Test User', email: 'test@example.com' }),
			auth_login: () => ({ token: 'mock-token', user: { id: 'mock-user', name: 'Test User', email: 'test@example.com' } }),
			auth_register: () => ({ token: 'mock-token', user: { id: 'mock-user', name: 'Test User', email: 'test@example.com' } }),
			auth_update_profile: () => ({ id: 'mock-user', name: 'Test User', email: 'test@example.com' }),
			auth_change_password: () => undefined,
			get_router_status: () => ({ status: 'running', version: 'v1.0.0', transport: 'stdio', uptime_seconds: 86400, port: 3000, metrics_endpoint: '3031/metrics' }),
			get_system_usage: () => ({ cpu_percent: 23, memory_percent: 45, disk_percent: 62, cpu_history: [20, 25, 22, 23], memory_history: [40, 42, 44, 43], disk_history: [60, 61, 62, 62] }),
			get_activity_log: () => [
				{ id: 'a1', type: 'router_started', message: 'mach1 router started', timestamp: new Date(Date.now() - 60000).toISOString(), icon: '\u25B6' },
				{ id: 'a2', type: 'mcp_started', message: 'GitHub MCP started', timestamp: new Date(Date.now() - 120000).toISOString(), icon: '\uD83D\uDCE6' },
			],
			get_mcp_servers: () => [
				{ id: 'mach1', name: 'Mach1 Router', description: 'Semantic router', version: '1.0.0', runtime: 'binary', status: 'running', status_detail: 'PID 21340', lifecycle: 'Manual', trust: 'internal', author: '1mcp.in', last_used_at: null, tools_count: 5, installed_at: new Date().toISOString(), process: { pid: 21340, memory_mb: 64.2, cpu_percent: 0.3, uptime_seconds: 86400, restarts: 0 } },
				{ id: 'github', name: 'GitHub', description: 'Search code', version: '0.6.2', runtime: 'node', status: 'running', status_detail: 'PID 21500', lifecycle: 'Auto (lazy)', trust: 'anthropic-official', author: 'Anthropic', idle_timeout: '15 minutes', last_used_at: new Date(Date.now() - 300000).toISOString(), tools_count: 37, installed_at: new Date().toISOString(), process: { pid: 21500, memory_mb: 128.5, cpu_percent: 1.2, uptime_seconds: 43200, restarts: 1 } },
				{ id: 'memory', name: 'Knowledge Graph Memory', description: 'Persistent knowledge graph', version: '0.6.0', runtime: 'node', status: 'running', status_detail: 'PID 21501', lifecycle: 'Auto (lazy)', trust: 'anthropic-official', author: 'Anthropic', idle_timeout: '15 minutes', last_used_at: new Date(Date.now() - 600000).toISOString(), tools_count: 15, installed_at: new Date().toISOString(), process: { pid: 21501, memory_mb: 92.1, cpu_percent: 0.8, uptime_seconds: 43200, restarts: 0 } },
				{ id: 'filesystem', name: 'Filesystem', description: 'Read, write, search files', version: '0.6.2', runtime: 'node', status: 'sleeping', status_detail: 'Idle', lifecycle: 'Auto (lazy)', trust: 'anthropic-official', author: 'Anthropic', idle_timeout: '15 minutes', last_used_at: new Date(Date.now() - 7200000).toISOString(), tools_count: 10, installed_at: new Date().toISOString() },
				{ id: 'postgres', name: 'PostgreSQL', description: 'Query databases', version: '1.0.0', runtime: 'node', status: 'error', status_detail: 'Connection refused', lifecycle: 'Auto (lazy)', trust: '1mcp.in-verified', author: 'db-tools', idle_timeout: '15 minutes', last_used_at: new Date(Date.now() - 86400000).toISOString(), tools_count: 12, installed_at: new Date().toISOString() },
			],
			get_client_connections: () => [
				{ id: 'vscode', connected: true }, { id: 'cursor', connected: true },
				{ id: 'claude', connected: false }, { id: 'claudecode', connected: false },
				{ id: 'windsurf', connected: false }, { id: 'codex', connected: false },
				{ id: 'antigravity', connected: false }, { id: 'opencode', connected: false },
			],
			get_client_detail: () => ({ id: 'vscode', name: 'VS Code', subtitle: 'via mcp.json', status: 'connected', transport: 'stdio', config_path: '~/.vscode/mcp.json', last_handshake: '2m ago', router_binding: 'mach1', process_id: '12345' }),
			get_client_routing_health: () => ({ requests: 42, active_tools: ['github__list_prs'], latency_avg_ms: 145, errors: 0, period: '5m' }),
			get_client_config_preview: () => ({ path: '~/.vscode/mcp.json', content: JSON.stringify({ 'mcpServers': { 'mach1': { command: 'mach1' } } }) }),
			list_installed: () => [
				{ id: 'mach1', name: 'Mach1 Router', version: '1.0.0', runtime: 'binary', enabled: true, command: 'mach1', args: [], description: 'Semantic router' },
				{ id: 'github', name: 'GitHub', version: '0.6.2', runtime: 'node', enabled: true, command: 'npx', args: ['-y', '@modelcontextprotocol/server-github'], description: 'Search code, read issues/PRs.', patProvider: 'github' },
			],
			get_settings: () => ({ start_on_login: true, minimize_to_tray: true, theme: 'dark', language: 'System Default', telemetry_enabled: false, log_level: 'info' }),
			get_system_info: () => ({ platform: 'Linux x86_64', version: 'v1.0.0', router_status: 'running', transport: 'stdio', uptime_seconds: 86400, metrics_endpoint: '127.0.0.1:3031/metrics', data_directory: '~/.1mcp' }),
			execute_command: () => ({ output: 'ok', error: '' }),
			restart_router: () => 'ok',
			scan_server: () => 'ok',
			restart_single_server: () => 'ok',
			uninstall_single_server: () => undefined,
			patch_client_config: () => 'ok',
			remove_client_config: () => true,
			install_mcp: () => undefined,
			uninstall_mcp: () => undefined,
			toggle_mcp: () => undefined,
			upsert_skill: () => undefined,
			toggle_skill: () => undefined,
			reset_router_config: () => undefined,
			clear_local_data: () => undefined,
			save_settings: () => undefined,
			copy_diagnostics: () => '{}',
			list_skills: () => [],
			fetch_cloud_marketplace: () => [],
			fetch_cloud_skills: () => [],
			fetch_cloud_stats: () => 1247,
			get_marketplace_item: () => null,
			get_server_config: () => ({ command: 'npx', args: ['-y', '@modelcontextprotocol/server-github'], cwd: '/home/user', env: [{ key: 'GITHUB_TOKEN', value: 'ghp_****', secret: true }] }),
			get_server_tools: () => [
				{ name: 'github__list_issues', description: 'List GitHub issues for a repository', inputSchema: { type: 'object', properties: { owner: { type: 'string' }, repo: { type: 'string' } } } },
				{ name: 'github__create_issue', description: 'Create a GitHub issue', inputSchema: { type: 'object', properties: {} } },
			],
			get_server_detail: (args) => {
				const servers = [
					{ id: 'mach1', name: 'Mach1 Router', description: 'Semantic router', version: '1.0.0', runtime: 'binary', status: 'running', status_detail: 'PID 21340', lifecycle: 'Manual', trust: 'internal', author: '1mcp.in', last_used_at: null, tools_count: 5, installed_at: new Date().toISOString(), process: { pid: 21340, memory_mb: 64.2, cpu_percent: 0.3, uptime_seconds: 86400, restarts: 0 } },
					{ id: 'github', name: 'GitHub', description: 'Search code', version: '0.6.2', runtime: 'node', status: 'running', status_detail: 'PID 21500', lifecycle: 'Auto (lazy)', trust: 'anthropic-official', author: 'Anthropic', idle_timeout: '15 minutes', last_used_at: new Date(Date.now() - 300000).toISOString(), tools_count: 37, installed_at: new Date().toISOString(), process: { pid: 21500, memory_mb: 128.5, cpu_percent: 1.2, uptime_seconds: 43200, restarts: 1 } },
				];
				const id = (args as any)?.id;
				return servers.find(s => s.id === id) ?? servers[0];
			},
			get_marketplace_item: (args) => {
				const id = (args as any)?.id;
				if (!id) return null;
				return {
					id, name: id.charAt(0).toUpperCase() + id.slice(1), shortDescription: 'A marketplace MCP server',
					description: 'Full description', version: '1.0.0', runtime: 'node', author: 'Marketplace',
					trust: 'community', license: 'MIT', sha256: 'abc123', verified_at: '2025-01-01',
					updated_at: '2025-01-01', downloads: 1000, rating: 4.5, reviewCount: 50,
					tags: ['test'], installed: false, capabilities: ['tool_call'],
					security_checks: [{ label: 'Tool schema verified', status: 'passed' }],
					requires_env: [],
				};
			},
			get_server_logs: () => [
				{ timestamp: new Date(Date.now() - 30000).toISOString(), level: 'info', message: 'Server started' },
				{ timestamp: new Date(Date.now() - 60000).toISOString(), level: 'warn', message: 'Connection pool resized' },
			],
			sync_marketplace: () => undefined,
		};

		const invoke = (cmd: string, args?: Record<string, unknown>) => {
			const handler = (handlers as Record<string, (args?: Record<string, unknown>) => unknown>)[cmd];
			if (handler) {
				return Promise.resolve(handler(args));
			}
			return Promise.resolve(null);
		};

		(window as any).__TAURI_INTERNALS__ = {
			invoke,
			metadata: { __windows: [], __currentWindow: { label: 'main' } },
		};
	});
}

export async function teardownMockTauri(page: Page) {
	await page.addInitScript(() => {
		delete (window as any).__TAURI_INTERNALS__;
	});
}
