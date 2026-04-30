import { test as base } from '@playwright/test';
import { setupMockApi, teardownMockApi } from './helpers/mock-api';
import { setupMockTauri, teardownMockTauri } from './helpers/mock-tauri';

export type TestOptions = {
	useMockApi: boolean;
	useMockTauri: boolean;
};

export const test = base.extend<TestOptions>({
	useMockApi: [true, { option: true }],
	useMockTauri: [true, { option: true }],

	page: async ({ page, useMockApi, useMockTauri }, use) => {
		await page.addInitScript(() => {
			localStorage.setItem('mcp_token', 'playwright-test-token');
			sessionStorage.removeItem('mcp_token');
		});
		if (useMockTauri) {
			await setupMockTauri(page);
		}
		if (useMockApi) {
			await setupMockApi(page);
		}
		await use(page);
		if (useMockApi) {
			await teardownMockApi(page);
		}
		if (useMockTauri) {
			await teardownMockTauri(page);
		}
	},
});

export { expect } from '@playwright/test';
