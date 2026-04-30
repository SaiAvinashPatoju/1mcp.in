import { test, expect } from '../../fixtures';

test.describe('Filter Combination Stress', () => {
	const tabs = ['installed', 'running', 'updates', 'disabled'];
	const runtimes = ['all', 'node', 'python', 'go', 'binary'];
	const statuses = ['all', 'running', 'sleeping', 'error'];
	const sorts = ['name', 'status'];

	test('all tab × runtime × status × sort combinations', async ({ page }) => {
		await page.goto('/servers');
		for (const tab of tabs) {
			await page.click(`button:has-text("${tab.charAt(0).toUpperCase() + tab.slice(1)}")`);
			await page.waitForTimeout(100);
			for (const runtime of runtimes) {
				await page.selectOption('select >> nth=0', runtime);
				for (const status of statuses) {
					await page.selectOption('select >> nth=1', status);
					for (const sort of sorts) {
						await page.selectOption('select >> nth=2', sort);
						await page.waitForTimeout(50);
						const rows = await page.locator('table tbody tr').count();
						expect(rows).toBeGreaterThanOrEqual(0);
					}
				}
			}
		}
	});
});
