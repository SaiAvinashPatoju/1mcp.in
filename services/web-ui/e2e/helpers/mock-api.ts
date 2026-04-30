import type { Page } from '@playwright/test';
import { seedRouterStatus, seedSystemUsage, seedActivityLog, seedMcpServers, seedClients, seedMarketplace, seedSkills, seedInstalled, seedUser } from './seed-data';

const DEFAULT_HANDLERS: Record<string, (url: URL) => { status: number; body: unknown }> = {
	'GET /api/auth/me': () => ({ status: 200, body: { user: seedUser } }),
	'GET /api/stats': () => ({ status: 200, body: { total_users: 1247 } }),
	'GET /api/router/status': () => ({ status: 200, body: seedRouterStatus }),
	'GET /api/system/usage': () => ({ status: 200, body: seedSystemUsage }),
	'GET /api/activity': () => ({ status: 200, body: { activities: seedActivityLog } }),
	'GET /api/mcp/servers': () => ({ status: 200, body: { servers: seedMcpServers } }),
	'GET /api/clients/connections': () => ({ status: 200, body: { clients: seedClients } }),
	'GET /api/marketplace': () => ({ status: 200, body: { items: seedMarketplace } }),
	'GET /api/skills': () => ({ status: 200, body: { items: seedSkills } }),
	'GET /api/installed': () => ({ status: 200, body: { items: seedInstalled } }),
	'GET /api/system/info': () => ({
		status: 200,
		body: {
			platform: 'Linux x86_64',
			version: 'v1.0.0',
			router_status: 'running',
			transport: 'stdio',
			uptime_seconds: 86400,
			metrics_endpoint: '127.0.0.1:3031/metrics',
			data_directory: '~/.1mcp',
		},
	}),
	'GET /api/settings': () => ({
		status: 200,
		body: {
			start_on_login: true,
			minimize_to_tray: true,
			theme: 'dark',
			language: 'System Default',
			telemetry_enabled: false,
			log_level: 'info',
		},
	}),
};

function matchKey(method: string, pathname: string): string | null {
	const normalized = pathname.replace(/\/+$/, '');
	const candidates = [`${method} ${normalized}`, `${method} ${normalized}/`];
	for (const key of Object.keys(DEFAULT_HANDLERS)) {
		if (candidates.includes(key)) return key;
		if (key.endsWith('/*') && normalized.startsWith(key.slice(0, -1))) return key;
	}
	return null;
}

export async function setupMockApi(page: Page) {
	await page.route('**/api/**', async (route) => {
		const url = new URL(route.request().url());
		const method = route.request().method();
		const key = matchKey(method, url.pathname);

		if (key) {
			const handler = DEFAULT_HANDLERS[key];
			if (handler) {
				const { status, body } = handler(url);
				await route.fulfill({ status, contentType: 'application/json', body: JSON.stringify(body) });
				return;
			}
		}

		if (method === 'POST' || method === 'PATCH' || method === 'DELETE') {
			await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ ok: true }) });
			return;
		}

		await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({}) });
	});
}

export async function teardownMockApi(page: Page) {
	await page.unroute('**/api/**');
}
