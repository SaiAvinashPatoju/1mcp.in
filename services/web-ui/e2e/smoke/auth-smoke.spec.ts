import { test, expect } from '../fixtures';

test.describe('Auth Smoke', () => {
	test.beforeEach(async ({ page }) => {
		await page.addInitScript(() => {
			localStorage.removeItem('mcp_token');
			sessionStorage.removeItem('mcp_token');
		});
	});

	test('landing page renders with sign-in form', async ({ page }) => {
		await page.goto('/');
		await expect(page.getByText('One router.')).toBeVisible();
		await expect(page.getByText('Every AI client.')).toBeVisible();
		await expect(page.getByRole('button', { name: 'Sign In' }).first()).toBeVisible();
		await expect(page.getByRole('button', { name: 'Sign Up' })).toBeVisible();
		await expect(page.locator('#email-input')).toBeVisible();
		await expect(page.locator('#password-input')).toBeVisible();
	});

	test('toggle between sign-in and sign-up', async ({ page }) => {
		await page.goto('/');
		await page.getByRole('button', { name: 'Sign Up' }).click();
		await expect(page.locator('#name-input')).toBeVisible();
		await page.getByRole('button', { name: 'Sign In' }).first().click();
		await expect(page.locator('a[href="/forgot-password"]')).toBeVisible();
	});

	test('social login buttons are visible', async ({ page }) => {
		await page.goto('/');
		await expect(page.getByRole('button', { name: 'Sign in with GitHub' })).toBeVisible();
		await expect(page.getByRole('button', { name: 'Sign in with Google' })).toBeVisible();
		await expect(page.getByRole('button', { name: 'Sign in with Discord' })).toBeVisible();
	});

	test('hero section value prop is displayed', async ({ page }) => {
		await page.goto('/');
		await expect(page.getByText('One router.')).toBeVisible();
		await expect(page.getByText('Every AI client.')).toBeVisible();
	});
});
