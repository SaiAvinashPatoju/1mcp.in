import { test, expect } from '../../fixtures';

test.describe('Modal Open/Close Stress', () => {
	test('rapidly open and close publish modal', async ({ page }) => {
		await page.goto('/discover');
		for (let i = 0; i < 5; i++) {
			await page.click('text=Publish MCP');
			await page.waitForTimeout(100);
			await page.keyboard.press('Escape');
			await page.waitForTimeout(100);
		}
	});

	test('rapidly open and close server detail panels', async ({ page }) => {
		await page.goto('/servers');
		const rows = page.locator('table tbody tr');
		const count = await rows.count();
		for (let i = 0; i < Math.min(count, 10); i++) {
			await rows.nth(i).click();
			await page.waitForTimeout(100);
			const closeBtn = page.locator('button[aria-label="Close"]');
			if (await closeBtn.isVisible()) {
				await closeBtn.click();
			}
		}
	});

	test('rapidly open and close client detail panels', async ({ page }) => {
		await page.goto('/clients');
		const rows = page.locator('table tbody tr');
		const count = await rows.count();
		for (let i = 0; i < Math.min(count, 8); i++) {
			await rows.nth(i).click();
			await page.waitForTimeout(100);
			const closeBtn = page.locator('button[aria-label="Close"]');
			if (await closeBtn.isVisible()) {
				await closeBtn.click();
			}
		}
	});

	test('rapidly switch between server detail tabs', async ({ page }) => {
		await page.goto('/servers');
		await page.locator('table tbody tr').first().click();
		await page.waitForTimeout(200);
		const tabs = ['Overview', 'Tools', 'Config', 'Environment', 'Logs'];
		for (let i = 0; i < 3; i++) {
			for (const tab of tabs) {
				const tabBtn = page.locator(`button:has-text("${tab}")`).last();
				if (await tabBtn.isVisible()) {
					await tabBtn.click();
					await page.waitForTimeout(50);
				}
			}
		}
	});
});
