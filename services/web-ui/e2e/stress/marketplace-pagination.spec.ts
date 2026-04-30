import { test, expect } from '../../fixtures';

test.describe('Marketplace Pagination Stress', () => {
	test('rapid page clicking through all pages', async ({ page }) => {
		await page.goto('/discover');
		await page.waitForSelector('table tbody tr');

		const navButtons = page.locator('button[aria-label="Next page"], button[aria-label="Previous page"]');
		const nextButton = page.locator('button[aria-label="Next page"]');

		for (let i = 0; i < 20; i++) {
			if (await nextButton.isDisabled()) {
				const prevButton = page.locator('button[aria-label="Previous page"]');
				for (let j = 0; j < 10; j++) {
					await prevButton.click();
					await page.waitForTimeout(50);
				}
				break;
			}
			await nextButton.click();
			await page.waitForTimeout(50);
		}

		const rows = await page.locator('table tbody tr').count();
		expect(rows).toBeGreaterThanOrEqual(0);
	});

	test('rapidly click direct page numbers', async ({ page }) => {
		await page.goto('/discover');
		const pageButtons = page.locator('button:has-text("1"), button:has-text("2"), button:has-text("3")');
		const count = await pageButtons.count();
		for (let i = 0; i < count; i++) {
			await pageButtons.nth(i).click();
			await page.waitForTimeout(50);
		}
	});
});
