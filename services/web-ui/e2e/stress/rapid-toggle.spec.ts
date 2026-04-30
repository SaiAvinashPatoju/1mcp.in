import { test, expect } from '../../fixtures';

test.describe('Rapid Toggle Stress', () => {
	test('rapidly toggle server start/stop', async ({ page }) => {
		await page.goto('/servers');
		await page.waitForSelector('table tbody tr');

		for (let i = 0; i < 20; i++) {
			const toggleButton = page.locator('button[title="Start"]').first();
			const stopButton = page.locator('button[title="Stop"]').first();
			if (await toggleButton.isVisible()) {
				await toggleButton.click();
			} else if (await stopButton.isVisible()) {
				await stopButton.click();
			}
			await page.waitForTimeout(100);
		}

		const rows = await page.locator('table tbody tr').count();
		expect(rows).toBeGreaterThan(0);
	});

	test('rapidly click back and forth between list and grid view', async ({ page }) => {
		await page.goto('/servers');
		for (let i = 0; i < 10; i++) {
			await page.click('button[aria-label="Grid view"]');
			await page.waitForTimeout(50);
			await page.click('button[aria-label="List view"]');
			await page.waitForTimeout(50);
		}
		await expect(page.locator('table')).toBeVisible();
	});
});
