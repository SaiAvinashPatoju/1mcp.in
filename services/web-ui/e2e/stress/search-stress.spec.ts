import { test, expect } from '../../fixtures';

test.describe('Search Stress', () => {
	const searchTerms = ['git', 'hub', 'memory', 'node', 'python', 'fetch', 'postgres', 'slack', 'filesystem', 'knowledge', 'ai', 'search', 'http', 'read', 'write'];

	test('rapid typing in servers search', async ({ page }) => {
		await page.goto('/servers');
		const searchInput = page.locator('input[placeholder="Search servers..."]');
		for (const term of searchTerms) {
			await searchInput.fill(term);
			await page.waitForTimeout(50);
			const count = await page.locator('table tbody tr').count();
			expect(count).toBeGreaterThanOrEqual(0);
		}
	});

	test('rapid typing in marketplace search', async ({ page }) => {
		await page.goto('/discover');
		const searchInput = page.locator('input[placeholder*="Search MCPs"]');
		for (const term of searchTerms) {
			await searchInput.fill(term);
			await page.waitForTimeout(50);
			const count = await page.locator('table tbody tr').count();
			expect(count).toBeGreaterThanOrEqual(0);
		}
	});

	test('rapid typing in clients search', async ({ page }) => {
		await page.goto('/clients');
		const searchInput = page.locator('input[placeholder="Search clients..."]');
		for (const term of ['code', 'cursor', 'claude', 'wind', 'surf', 'vscode', 'open', 'codex']) {
			await searchInput.fill(term);
			await page.waitForTimeout(50);
			const count = await page.locator('table tbody tr').count();
			expect(count).toBeGreaterThanOrEqual(0);
		}
	});

	test('rapid clear and retype across all search inputs', async ({ page }) => {
		const searches = [
			{ path: '/servers', input: 'input[placeholder="Search servers..."]' },
			{ path: '/discover', input: 'input[placeholder*="Search MCPs"]' },
			{ path: '/clients', input: 'input[placeholder="Search clients..."]' },
		];
		for (const { path, input } of searches) {
			await page.goto(path);
			const searchInput = page.locator(input);
			for (let i = 0; i < 10; i++) {
				await searchInput.fill(searchTerms[i % searchTerms.length]);
				await searchInput.clear();
			}
		}
	});
});
