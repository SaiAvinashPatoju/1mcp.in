import { test, expect } from '../../fixtures';

test.describe('Navigation Stress', () => {
	const routes = ['/dashboard', '/servers', '/discover', '/clients', '/settings'];

	for (let i = 0; i < 50; i++) {
		const route = routes[i % routes.length];
		test(`rapid navigate to ${route} (iteration ${i + 1})`, async ({ page }) => {
			await page.goto(route, { waitUntil: 'domcontentloaded' });
			const status = await page.waitForSelector('h1', { timeout: 5000 }).then(() => 'ok').catch(() => 'timeout');
			expect(status).toBe('ok');
			const errors: string[] = [];
			page.on('console', (msg) => {
				if (msg.type() === 'error') errors.push(msg.text());
			});
			expect(errors.filter(e => !e.includes('favicon') && !e.includes('ws://'))).toHaveLength(0);
		});
	}
});
