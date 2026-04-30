import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
	testDir: './e2e',
	fullyParallel: false,
	forbidOnly: !!process.env.CI,
	retries: process.env.CI ? 1 : 0,
	workers: process.env.CI ? 2 : 1,
	reporter: process.env.CI
		? [['github'], ['html', { outputFolder: 'playwright-report' }]]
		: [['list'], ['html', { outputFolder: 'playwright-report' }]],
	use: {
		baseURL: 'http://localhost:1420',
		trace: process.env.CI ? 'on-first-retry' : 'retain-on-failure',
		screenshot: 'only-on-failure',
		video: process.env.CI ? 'retain-on-failure' : 'off',
	},
	projects: [
		{
			name: 'smoke',
			testMatch: 'e2e/smoke/**/*.spec.ts',
			use: { ...devices['Desktop Chrome'] },
		},
		{
			name: 'quality',
			testMatch: 'e2e/quality/**/*.spec.ts',
			use: { ...devices['Desktop Chrome'] },
		},
		{
			name: 'stress',
			testMatch: 'e2e/stress/**/*.spec.ts',
			use: { ...devices['Desktop Chrome'] },
			fullyParallel: false,
			workers: 1,
		},
		{
			name: 'integration',
			testMatch: 'e2e/integration/**/*.spec.ts',
			use: { ...devices['Desktop Chrome'] },
		},
	],
	webServer: [
		{
			command: 'npm run dev -- --mode test',
			url: 'http://localhost:1420',
			reuseExistingServer: !process.env.CI,
			cwd: 'services/web-ui',
			timeout: 30000,
		},
	],
});
