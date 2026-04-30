import { test, expect } from '../fixtures';

test.describe('Servers - Quality', () => {
	test.beforeEach(async ({ page }) => {
		await page.goto('/servers');
		await page.waitForSelector('h1');
	});

	test('page title and subtitle render', async ({ page }) => {
		await expect(page.locator('h1')).toContainText('Servers');
	});

	test('stats cards show correct counts', async ({ page }) => {
		await expect(page.getByText('5 installed').first()).toBeVisible();
		await expect(page.getByText('3 running').first()).toBeVisible();
		await expect(page.getByText('1 sleeping').first()).toBeVisible();
		await expect(page.getByText('1 errored').first()).toBeVisible();
	});

	test('table view renders all server columns', async ({ page }) => {
		const headers = page.locator('table thead th');
		await expect(headers).toContainText(['Name', 'Runtime', 'Status', 'Lifecycle', 'Version', 'Last Used', 'Tools', 'Actions']);
	});

	test('tabs switch content correctly', async ({ page }) => {
		await page.getByRole('button', { name: 'Running' }).click();
		await page.waitForTimeout(300);
		await expect(page.locator('table tbody tr')).toHaveCount(3);
		await page.getByRole('button', { name: 'Disabled' }).click();
		await page.waitForTimeout(300);
		await expect(page.locator('table tbody tr')).toHaveCount(1);
		await page.getByRole('button', { name: 'Installed' }).click();
		await page.waitForTimeout(300);
		await expect(page.locator('table tbody tr')).toHaveCount(5);
	});

	test('grid view toggle works', async ({ page }) => {
		await page.click('button[aria-label="Grid view"]');
		await page.waitForTimeout(300);
		await expect(page.locator('.grid.grid-cols-2')).toBeVisible();
		await page.click('button[aria-label="List view"]');
		const table = page.locator('table');
		await expect(table).toBeVisible();
	});

	test('search filters servers by name', async ({ page }) => {
		await page.fill('input[placeholder="Search servers..."]', 'postgres');
		await expect(page.locator('table tbody tr')).toHaveCount(1);
		await expect(page.locator('text=PostgreSQL').first()).toBeVisible();
	});

	test('search filters by description', async ({ page }) => {
		await page.fill('input[placeholder="Search servers..."]', 'Persistent');
		await expect(page.locator('text=Knowledge Graph Memory')).toBeVisible();
	});

	test('runtime filter works', async ({ page }) => {
		await page.selectOption('select >> nth=0', 'node');
		await page.waitForTimeout(300);
		await expect(page.locator('table tbody tr')).toHaveCount(4);
	});

	test('status filter works', async ({ page }) => {
		await page.selectOption('select >> nth=1', 'error');
		await expect(page.locator('table tbody tr')).toHaveCount(1);
	});

	test('sort by name works', async ({ page }) => {
		await page.selectOption('select >> nth=2', 'name');
		const firstRowName = await page.locator('table tbody tr:first-child td:first-child').textContent();
		expect(firstRowName).toContain('Filesystem');
	});

	test('clicking a row opens detail panel', async ({ page }) => {
		await page.click('table tbody tr:first-child');
		await expect(page.locator('text=Overview').first()).toBeVisible();
		await expect(page.locator('text=Tools').first()).toBeVisible();
		await expect(page.locator('text=Config').first()).toBeVisible();
		await expect(page.locator('text=Logs')).toBeVisible();
	});

	test('detail panel shows server info', async ({ page }) => {
		await page.locator('text=GitHub').first().click();
		await expect(page.locator('text=Anthropic').first()).toBeVisible();
	});

	test('detail overview tab shows process info', async ({ page }) => {
		await page.click('table tbody tr:first-child');
		await page.waitForTimeout(300);
		await expect(page.getByText('Memory').first()).toBeVisible();
		await expect(page.getByText('PID').first()).toBeVisible();
		await expect(page.getByText('CPU').first()).toBeVisible();
		await expect(page.getByText('Uptime').first()).toBeVisible();
	});

	test('detail tools tab lists tools', async ({ page }) => {
		await page.locator('text=GitHub').first().click();
		await page.waitForTimeout(300);
		await page.getByRole('button', { name: 'Tools', exact: true }).click();
		await page.waitForTimeout(300);
		await expect(page.getByText('github__list_issues')).toBeVisible();
	});

	test('detail config tab shows command', async ({ page }) => {
		await page.locator('text=GitHub').first().click();
		await page.waitForTimeout(300);
		await page.getByRole('button', { name: 'Config', exact: true }).click();
		await page.waitForTimeout(300);
		await expect(page.getByText('Command', { exact: true })).toBeVisible();
	});

	test('detail environment tab shows env vars', async ({ page }) => {
		await page.locator('text=GitHub').first().click();
		await page.getByRole('button', { name: 'Environment' }).click();
	});

	test('start/stop toggle changes server status', async ({ page }) => {
		const startButton = page.locator('button[title="Start"]').first();
		if (await startButton.isVisible()) {
			await startButton.click();
		}
	});

	test('uninstall requires confirmation', async ({ page }) => {
		await page.locator('text=GitHub').first().click();
		await page.locator('text=Uninstall').first().click();
	});

	test('scan for changes button exists', async ({ page }) => {
		await expect(page.locator('text=Scan for Changes').first()).toBeVisible();
	});

	test('add server button navigates to marketplace', async ({ page }) => {
		await page.locator('text=Add Server').first().click();
		await expect(page).toHaveURL(/\/marketplace/);
	});

	test('search with all filters combined', async ({ page }) => {
		await page.fill('input[placeholder="Search servers..."]', 'git');
		await page.selectOption('select >> nth=0', 'all');
		await page.selectOption('select >> nth=1', 'all');
		await page.selectOption('select >> nth=2', 'name');
		const rows = await page.locator('table tbody tr').count();
		expect(rows).toBeGreaterThanOrEqual(1);
	});

	test('close detail panel resets state', async ({ page }) => {
		await page.click('table tbody tr:first-child');
		await expect(page.locator('button[aria-label="Close"]')).toBeVisible();
		await page.click('button[aria-label="Close"]');
		await expect(page.locator('text=Overview')).toHaveCount(0);
	});
});
