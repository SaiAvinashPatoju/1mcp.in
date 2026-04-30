import { test, expect } from '../fixtures';

test.describe('Marketplace - Quality', () => {
	test.beforeEach(async ({ page }) => {
		await page.goto('/discover');
		await page.waitForSelector('h1');
	});

	test('page title shows Marketplace', async ({ page }) => {
		await expect(page.locator('h1')).toContainText('Marketplace');
	});

	test('MCPs tab is active by default', async ({ page }) => {
		await expect(page.getByRole('button', { name: 'MCPs' })).toBeVisible();
	});

	test('all marketplace items render', async ({ page }) => {
		const rows = page.locator('table tbody tr');
		await expect(rows).toHaveCount(8);
	});

	test('marketplace table has correct columns', async ({ page }) => {
		const headers = page.locator('table thead th');
		await expect(headers).toContainText(['Name', 'Maintainer', 'Runtime', 'Trust', 'Version', 'Installs', 'Status', 'Actions']);
	});

	test('search filters MCPs', async ({ page }) => {
		await page.fill('input[placeholder*="Search MCPs"]', 'slack');
		await expect(page.locator('table tbody tr')).toHaveCount(1);
		await expect(page.getByRole('cell', { name: /Slack/i })).toBeVisible();
	});

	test('trust filter works', async ({ page }) => {
		await page.selectOption('select >> nth=0', 'community');
		const rows = page.locator('table tbody tr');
		for (let i = 0; i < await rows.count(); i++) {
			await expect(rows.nth(i)).toContainText('community');
		}
	});

	test('runtime filter works', async ({ page }) => {
		await page.selectOption('select >> nth=1', 'python');
		await page.waitForTimeout(300);
		await expect(page.locator('table tbody tr')).toHaveCount(3);
	});

	test('status filter shows installed only', async ({ page }) => {
		await page.selectOption('select >> nth=2', 'installed');
		await page.waitForTimeout(300);
		await expect(page.locator('table tbody tr')).toHaveCount(1);
	});

	test('status filter shows available only', async ({ page }) => {
		await page.selectOption('select >> nth=2', 'available');
		await page.waitForTimeout(300);
		await expect(page.locator('table tbody tr')).toHaveCount(8);
	});

	test('sort by rating works', async ({ page }) => {
		await page.selectOption('select >> nth=3', 'rating');
		await page.waitForTimeout(300);
		const firstRating = await page.locator('table tbody tr:first-child td:nth-child(6)').textContent();
		expect(firstRating?.trim()).toBe('93k');
	});

	test('pagination shows correct count', async ({ page }) => {
		await expect(page.getByText('Showing 1 to 8 of 9 MCPs', { exact: true })).toBeVisible();
	});

	test('clicking a row opens detail panel', async ({ page }) => {
		await page.click('table tbody tr:first-child');
		await page.waitForTimeout(300);
		await expect(page.getByText('Installs').first()).toBeVisible();
		await expect(page.getByText('Rating', { exact: true })).toBeVisible();
		await expect(page.getByText('Updated', { exact: true })).toBeVisible();
	});

	test('detail panel shows security checks', async ({ page }) => {
		await page.click('table tbody tr:first-child');
		await expect(page.getByText('Security').first()).toBeVisible();
		await expect(page.getByText('Tool schema verified').first()).toBeVisible();
	});

	test('detail panel shows capabilities', async ({ page }) => {
		await page.click('table tbody tr:first-child');
		await expect(page.getByText('Capabilities').first()).toBeVisible();
	});

	test('install button changes to uninstall', async ({ page }) => {
		await page.click('tr:has-text("Fetch") button[title="Install"]');
		await expect(page.locator('tr:has-text("Fetch")').getByText('Installed')).toBeVisible();
	});

	test('Publish MCP button opens modal', async ({ page }) => {
		await page.getByRole('button', { name: /Publish MCP/i }).click();
		await expect(page.getByText('Upload').first()).toBeVisible();
	});

	test('Skills tab renders skill cards', async ({ page }) => {
		await page.getByRole('button', { name: 'Skills' }).click();
		await expect(page.getByText('Frontend Developer').first()).toBeVisible();
		await expect(page.getByText('Backend Developer').first()).toBeVisible();
	});

	test('skill card shows install/uninstall', async ({ page }) => {
		await page.getByRole('button', { name: 'Skills' }).click();
		await page.waitForTimeout(300);
		await expect(page.getByRole('button', { name: 'Install' }).first()).toBeVisible();
	});

	test('registry sync indicator visible', async ({ page }) => {
		await expect(page.getByText('Registry synced').first()).toBeVisible();
	});

	test('check for updates button exists', async ({ page }) => {
		await expect(page.getByRole('button', { name: /Check for updates/i })).toBeVisible();
	});

	test('empty search shows no MCPs found', async ({ page }) => {
		await page.fill('input[placeholder*="Search MCPs"]', 'zzzzz_nonexistent');
		await expect(page.getByText('No MCPs found', { exact: true })).toBeVisible();
	});

	test('detail panel shows license info', async ({ page }) => {
		await page.click('table tbody tr:first-child');
		await expect(page.getByText('License').first()).toBeVisible();
	});

	test('detail panel shows SHA256 digest', async ({ page }) => {
		await page.click('table tbody tr:first-child');
		await expect(page.getByText('Digest').first()).toBeVisible();
	});

	test('detail panel has Configure ENV button', async ({ page }) => {
		await page.click('table tbody tr:first-child');
		await expect(page.getByRole('button', { name: /Configure ENV/i })).toBeVisible();
	});
});
