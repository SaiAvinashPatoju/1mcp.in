import { test, expect } from '../fixtures';

test.describe('Clients - Quality', () => {
	test.beforeEach(async ({ page }) => {
		await page.goto('/clients');
		await page.waitForSelector('h1');
	});

	test('page title and connection summary', async ({ page }) => {
		await expect(page.locator('h1')).toContainText('Clients');
		await expect(page.getByText('2 of 8 connected')).toBeVisible();
	});

	test('router status card shows running state', async ({ page }) => {
		await expect(page.getByRole('heading', { name: 'mach1 Router' })).toBeVisible();
		await expect(page.getByText('RUNNING')).toBeVisible();
	});

	test('router status details are displayed', async ({ page }) => {
		await expect(page.getByText('Transport').first()).toBeVisible();
		await expect(page.getByText('Uptime').first()).toBeVisible();
		await expect(page.getByText('Port: 3000')).toBeVisible();
	});

	test('all 8 clients render in the table', async ({ page }) => {
		const rows = page.locator('table tbody tr');
		await expect(rows).toHaveCount(8);
	});

	test('table has correct columns', async ({ page }) => {
		const headers = page.locator('table thead th');
		await expect(headers).toContainText(['Client', 'Status', 'Transport', 'Config Path', 'Last Seen', 'Routing', 'Actions']);
	});

	test('connected clients show CONNECTED status', async ({ page }) => {
		await expect(page.getByRole('table').getByText('CONNECTED', { exact: true })).toHaveCount(2);
	});

	test('not connected clients show NOT CONNECTED', async ({ page }) => {
		await expect(page.getByRole('table').getByText('NOT CONNECTED', { exact: true })).toHaveCount(2);
	});

	test('disconnected clients show DISCONNECTED', async ({ page }) => {
		await expect(page.getByRole('table').getByText('DISCONNECTED', { exact: true })).toHaveCount(4);
	});

	test('search filters clients', async ({ page }) => {
		await page.getByPlaceholder('Search clients...').fill('Codex');
		await page.waitForTimeout(500);
		const rows = await page.locator('table tbody tr').count();
		expect(rows).toBeGreaterThanOrEqual(1);
	});

	test('status filter shows connected only', async ({ page }) => {
		await page.selectOption('select', 'connected');
		await expect(page.locator('table tbody tr')).toHaveCount(2);
	});

	test('supported clients tab filters list', async ({ page }) => {
		await page.getByRole('button', { name: 'Supported Clients' }).click();
		const rows = await page.locator('table tbody tr').count();
		expect(rows).toBeGreaterThanOrEqual(7);
	});

	test('clicking a row opens detail panel', async ({ page }) => {
		await page.getByRole('row').filter({ hasText: 'VS Code' }).first().click();
		await expect(page.getByRole('heading', { name: 'Connection' })).toBeVisible();
		const panel = page.locator('.w-80');
		await expect(panel.getByText('Transport')).toBeVisible();
		await expect(panel.getByText('Config Path')).toBeVisible();
	});

	test('detail panel shows routing health', async ({ page }) => {
		await page.getByRole('row').filter({ hasText: 'VS Code' }).first().click();
		await expect(page.getByRole('heading', { name: 'Routing Health' })).toBeVisible();
	});

	test('setup button exists for not-connected clients', async ({ page }) => {
		await expect(page.getByRole('button', { name: 'Setup' }).first()).toBeVisible();
	});

	test('disconnect button exists for connected clients', async ({ page }) => {
		await expect(page.getByRole('button', { name: 'Disconnect' }).first()).toBeVisible();
	});

	test('Connect All Supported button exists', async ({ page }) => {
		await expect(page.getByRole('button', { name: 'Connect All Supported' })).toBeVisible();
	});

	test('refresh button exists', async ({ page }) => {
		await expect(page.getByRole('button', { name: 'Refresh' })).toBeVisible();
	});
});
