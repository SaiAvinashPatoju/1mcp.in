import { test, expect } from '../fixtures';

test.describe('Navigation Smoke', () => {
	const routes = [
		{ path: '/dashboard', title: 'Dashboard' },
		{ path: '/servers', title: 'Servers' },
		{ path: '/discover', title: 'Marketplace' },
		{ path: '/clients', title: 'Clients' },
		{ path: '/settings', title: 'Settings' },
	];

	for (const route of routes) {
		test(`${route.path} loads without 404`, async ({ page }) => {
			const response = await page.goto(route.path);
			expect(response?.status()).toBeLessThan(400);
		});
	}

	test('console error count is zero across page loads', async ({ page }) => {
		const errors: string[] = [];
		page.on('console', (msg) => {
			if (msg.type() === 'error') errors.push(msg.text());
		});
		for (const route of routes) {
			await page.goto(route.path);
		}
		expect(errors.filter(e => !e.includes('favicon') && !e.includes('ws://'))).toHaveLength(0);
	});
});
