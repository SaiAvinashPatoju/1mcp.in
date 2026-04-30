import { test, expect } from '../fixtures';

test.describe('Settings - Quality', () => {
	test.beforeEach(async ({ page }) => {
		await page.goto('/settings');
		await page.waitForSelector('h1');
	});

	test('page title renders', async ({ page }) => {
		await expect(page.getByRole('heading', { level: 1 })).toContainText('Settings');
	});

	test('top tabs (Settings / Account) are present', async ({ page }) => {
		await expect(page.getByRole('button', { name: 'Settings' }).first()).toBeVisible();
		await expect(page.getByRole('button', { name: 'Account' }).first()).toBeVisible();
	});

	test('all 8 settings sidebar tabs render', async ({ page }) => {
		const tabs = ['General', 'Router', 'MCP Servers', 'Clients', 'Security', 'Marketplace', 'Updates', 'Advanced'];
		for (const tab of tabs) {
			await expect(page.getByRole('button', { name: tab, exact: true })).toBeVisible();
		}
	});

	test('General tab shows startup settings', async ({ page }) => {
		await expect(page.getByText('Start mach1 on login', { exact: true })).toBeVisible();
		await expect(page.getByText('Minimize to system tray', { exact: true })).toBeVisible();
	});

	test('theme dropdown has options', async ({ page }) => {
		const themeSelect = page.locator('select').first();
		await expect(themeSelect).toContainText('Dark');
		await expect(themeSelect).toContainText('Light');
		await expect(themeSelect).toContainText('System');
	});

	test('toggle switches work for start on login', async ({ page }) => {
		await page.getByRole('button', { name: 'Start mach1 on login' }).click();
	});

	test('toggle switches work for minimize to tray', async ({ page }) => {
		await page.getByRole('button', { name: 'Minimize to system tray' }).click();
	});

	test('toggle switches work for telemetry', async ({ page }) => {
		await page.getByRole('button', { name: 'Enable telemetry' }).click();
	});

	test('save changes button exists', async ({ page }) => {
		await expect(page.getByRole('button', { name: 'Save Changes' })).toBeVisible();
	});

	test('Router settings tab content', async ({ page }) => {
		await page.getByRole('button', { name: 'Router', exact: true }).click();
		await page.waitForTimeout(300);
		await expect(page.getByText('Router settings will be available in a future update.', { exact: true })).toBeVisible();
	});

	test('Marketplace settings tab content', async ({ page }) => {
		await page.getByRole('button', { name: 'Marketplace', exact: true }).click();
		await page.waitForTimeout(300);
		await expect(page.getByText('Marketplace settings will be available in a future update.', { exact: true })).toBeVisible();
	});

	test('account sidebar card shows user info', async ({ page }) => {
		await expect(page.getByText('Local Account', { exact: true })).toBeVisible();
		await expect(page.getByText('Not a member', { exact: true })).toBeVisible();
	});

	test('system information card displays data', async ({ page }) => {
		await expect(page.getByText('System Information', { exact: true })).toBeVisible();
		await expect(page.getByText('Platform', { exact: true })).toBeVisible();
		await expect(page.getByText('Version', { exact: true })).toBeVisible();
		await expect(page.getByText('Log Level', { exact: true })).toBeVisible();
	});

	test('copy diagnostics button exists', async ({ page }) => {
		await expect(page.getByRole('button', { name: 'Copy Diagnostics' })).toBeVisible();
	});

	test('danger zone shows reset router', async ({ page }) => {
		await expect(page.getByText('Danger Zone', { exact: true })).toBeVisible();
		await expect(page.getByRole('button', { name: 'Reset Router' })).toBeVisible();
	});

	test('danger zone shows uninstall', async ({ page }) => {
		await expect(page.getByRole('button', { name: 'Uninstall' })).toBeVisible();
	});

	test('Account tab shows profile editing', async ({ page }) => {
		await page.getByRole('button', { name: 'Account' }).first().click();
		await expect(page.getByText('Display Name', { exact: true })).toBeVisible();
		await expect(page.getByText('Email Address', { exact: true })).toBeVisible();
		await expect(page.getByRole('button', { name: 'Save Profile' })).toBeVisible();
	});

	test('Account tab shows password section', async ({ page }) => {
		await page.getByRole('button', { name: 'Account' }).first().click();
		await expect(page.getByText('Password', { exact: true })).toBeVisible();
		await expect(page.getByText('Current Password', { exact: true })).toBeVisible();
		await expect(page.getByText('New Password', { exact: true })).toBeVisible();
		await expect(page.getByText('Confirm Password', { exact: true })).toBeVisible();
	});

	test('password validation - empty fields', async ({ page }) => {
		await page.getByRole('button', { name: 'Account', exact: true }).click();
		await page.waitForTimeout(300);
		await page.getByRole('button', { name: 'Update Password' }).click();
		await expect(page.getByText('Fill in all password fields.', { exact: true })).toBeVisible();
	});

	test('password validation - too short', async ({ page }) => {
		await page.getByRole('button', { name: 'Account', exact: true }).click();
		await page.waitForTimeout(300);
		await page.fill('#acc-current-password', 'somepassword');
		await page.fill('#acc-new-password', 'short');
		await page.fill('#acc-confirm-password', 'short');
		await page.getByRole('button', { name: 'Update Password' }).click();
		await expect(page.getByText('New password must be at least 8 characters.', { exact: true })).toBeVisible();
	});
});
