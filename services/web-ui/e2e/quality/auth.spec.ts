import { test, expect } from '../fixtures';

test.describe('Auth - Quality', () => {
	test.beforeEach(async ({ page }) => {
		await page.addInitScript(() => {
			localStorage.removeItem('mcp_token');
			sessionStorage.removeItem('mcp_token');
		});
	});

	test('sign-in form renders', async ({ page }) => {
		await page.goto('/');
		await expect(page.getByRole('heading', { name: 'One router.' })).toBeVisible();
		await expect(page.locator('#email-input')).toBeVisible();
		await expect(page.locator('#password-input')).toBeVisible();
	});

	test('sign-up form shows name field', async ({ page }) => {
		await page.goto('/');
		await page.getByRole('button', { name: 'Sign Up' }).click();
		await expect(page.locator('#name-input')).toBeVisible();
	});

	test('toggle password visibility', async ({ page }) => {
		await page.goto('/');
		const input = page.locator('#password-input');
		await expect(input).toHaveAttribute('type', 'password');
		await page.locator('#password-input ~ button').click();
		await expect(input).toHaveAttribute('type', 'text');
	});

	test('remember me checkbox exists', async ({ page }) => {
		await page.goto('/');
		await expect(page.getByText('Remember me')).toBeVisible();
	});

	test('forgot password link visible', async ({ page }) => {
		await page.goto('/');
		await expect(page.locator('a[href="/forgot-password"]')).toBeVisible();
	});

	test('hero section elements visible', async ({ page }) => {
		await page.goto('/');
		await expect(page.getByText('Public Beta')).toBeVisible();
		await expect(page.getByText('Install now')).toBeVisible();
		await expect(page.getByText('View on GitHub')).toBeVisible();
	});

	test('terms of service link visible', async ({ page }) => {
		await page.goto('/');
		await expect(page.locator('a[href="/terms"]')).toBeVisible();
	});

	test('auth mode toggle switches content', async ({ page }) => {
		await page.goto('/');
		await page.getByRole('button', { name: 'Sign Up' }).click();
		await expect(page.locator('#name-input')).toBeVisible();
		await page.getByRole('button', { name: 'Sign In' }).first().click();
		await expect(page.locator('a[href="/forgot-password"]')).toBeVisible();
	});

	test('page title is correct', async ({ page }) => {
		await page.goto('/');
		const title = await page.title();
		expect(title).toContain('1mcp.in');
	});
});
