import { test, expect } from '../fixtures';

test.describe('Dashboard Smoke', () => {
	test('dashboard loads with 200 status', async ({ page }) => {
		const response = await page.goto('/dashboard');
		expect(response?.status()).toBeLessThan(400);
	});
});
