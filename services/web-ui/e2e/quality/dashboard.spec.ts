import { test, expect } from '../fixtures';

test.describe('Dashboard - Quality', () => {
	test.beforeEach(async ({ page }) => {
		await page.goto('/dashboard');
	});

	test('router status card shows running state', async ({ page }) => {
		await expect(page.getByText('RUNNING', { exact: true })).toBeVisible({ timeout: 10000 });
		await expect(page.getByRole('heading', { name: 'mach1 Router' })).toBeVisible();
	});

	test('router uptime and transport are displayed', async ({ page }) => {
		await expect(page.getByText('Transport')).toBeVisible();
		await expect(page.getByText('Uptime')).toBeVisible();
		await expect(page.getByText('Port (HTTP)')).toBeVisible();
		await expect(page.getByText('Metrics', { exact: true }).first()).toBeVisible();
	});

	test('MCP servers table lists all servers', async ({ page }) => {
		await page.waitForSelector('table tbody tr', { timeout: 10000 });
		const rows = page.locator('table tbody tr');
		await expect(rows).toHaveCount(5);
	});

	test('server table shows correct columns', async ({ page }) => {
		const headers = page.locator('table thead th');
		await expect(headers.nth(0)).toContainText('Name');
		await expect(headers.nth(7)).toContainText('Actions');
	});

	test('server status indicators are correct', async ({ page }) => {
		await page.waitForSelector('text=Filesystem', { timeout: 10000 });
		const body = page.locator('table tbody');
		await expect(body.locator('text=Running')).toHaveCount(3);
		await expect(body.locator('text=Sleeping')).toHaveCount(1);
		await expect(body.locator('text=Error')).toHaveCount(1);
	});

	test('search input exists on table', async ({ page }) => {
		await page.waitForSelector('table tbody tr', { timeout: 10000 });
		await expect(page.locator('input[placeholder="Search servers..."]')).toBeVisible();
	});

	test('server summary counts are accurate', async ({ page }) => {
		await expect(page.getByText(/5 servers installed/)).toBeVisible({ timeout: 10000 });
		await expect(page.getByText(/3 running/)).toBeVisible();
	});

	test('system usage gauges render', async ({ page }) => {
		await expect(page.getByText('CPU Usage')).toBeVisible();
		await expect(page.getByText('Memory Usage')).toBeVisible();
		await expect(page.getByText('Disk Usage')).toBeVisible();
	});

	test('client connections card shows connected clients', async ({ page }) => {
		await expect(page.getByText('Client Connections')).toBeVisible();
	});

	test('recent activity feed displays events', async ({ page }) => {
		await expect(page.getByRole('heading', { name: 'Recent Activity' })).toBeVisible();
	});

	test('quick action buttons navigate correctly', async ({ page }) => {
		await page.getByText('Add Server').click();
		await expect(page).toHaveURL(/\/discover/);
	});

	test('restart router button exists', async ({ page }) => {
		await expect(page.getByText('Restart Router')).toBeVisible();
	});

	test('open logs button exists', async ({ page }) => {
		await expect(page.getByText('Open Logs')).toBeVisible();
	});

	test('server start/stop toggle buttons work', async ({ page }) => {
		await page.waitForSelector('table tbody tr', { timeout: 10000 });
		const stopBtn = page.locator('button[title="Stop"]').first();
		await expect(stopBtn).toBeVisible();
		const startBtn = page.locator('button[title="Start"]').first();
		await expect(startBtn).toBeVisible();
	});

	test('command line input exists', async ({ page }) => {
		await expect(page.getByText('Command Line')).toBeVisible();
		await expect(page.locator('input[placeholder="Type a command..."]')).toBeVisible();
	});

	test('server count in summary matches table', async ({ page }) => {
		await page.waitForSelector('table tbody tr', { timeout: 10000 });
		const summaryText = await page.getByText(/servers installed/).textContent();
		const rows = await page.locator('table tbody tr').count();
		expect(summaryText).toContain(String(rows));
	});
});
