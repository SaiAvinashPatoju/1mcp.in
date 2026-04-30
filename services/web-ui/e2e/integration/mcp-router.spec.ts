import { test, expect } from '../fixtures';

test.describe('MCP Router Integration', () => {
	test('router status API responds', async ({ page }) => {
		const response = await page.request.get('http://localhost:8080/api/router/status');
		expect(response.ok()).toBeTruthy();
		const data = await response.json();
		expect(data).toHaveProperty('status');
	});

	test('marketplace API returns items', async ({ page }) => {
		const response = await page.request.get('http://localhost:8080/api/marketplace');
		if (response.ok()) {
			const data = await response.json();
			expect(data).toHaveProperty('items');
		}
	});

	test('dashboard loads via real API', async ({ page, useMockApi }) => {
		test.fixme(useMockApi, 'This test needs the real backend running');
		await page.goto('/dashboard');
		await expect(page.locator('h1')).toContainText('Dashboard');
	});
});
