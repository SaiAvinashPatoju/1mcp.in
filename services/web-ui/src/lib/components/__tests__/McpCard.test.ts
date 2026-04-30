import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import McpCard from '../McpCard.svelte';
import type { MarketplaceMcp } from '../../types';

const baseMcp: MarketplaceMcp = {
	id: 'github',
	name: 'GitHub',
	shortDescription: 'Search code, read issues/PRs.',
	version: '0.6.2',
	runtime: 'node',
	author: 'Anthropic',
	tags: ['github', 'git'],
	rating: 4.9,
	reviewCount: 634,
	downloads: 92800,
	verificationStatus: 'anthropic-official',
	publishedAt: '2024-11-05',
	installed: false,
};

describe('McpCard', () => {
	it('renders MCP name and author', () => {
		const { getByText } = render(McpCard, { props: { mcp: baseMcp, onInstall: vi.fn(), onUninstall: vi.fn() } });
		expect(getByText('GitHub')).toBeDefined();
		expect(getByText('by Anthropic')).toBeDefined();
	});

	it('shows trust badge with correct label', () => {
		const { getByText } = render(McpCard, { props: { mcp: baseMcp, onInstall: vi.fn(), onUninstall: vi.fn() } });
		expect(getByText('Anthropic Official')).toBeDefined();
	});

	it('shows install button when not installed', () => {
		const { getByText } = render(McpCard, { props: { mcp: baseMcp, onInstall: vi.fn(), onUninstall: vi.fn() } });
		expect(getByText('Install')).toBeDefined();
	});

	it('shows uninstall button when installed', () => {
		const installed = { ...baseMcp, installed: true };
		const { getByText } = render(McpCard, { props: { mcp: installed, onInstall: vi.fn(), onUninstall: vi.fn() } });
		expect(getByText('Uninstall')).toBeDefined();
	});

	it('calls onInstall when install clicked', async () => {
		const onInstall = vi.fn();
		const { getByText } = render(McpCard, { props: { mcp: baseMcp, onInstall, onUninstall: vi.fn() } });
		await fireEvent.click(getByText('Install'));
		expect(onInstall).toHaveBeenCalledOnce();
	});

	it('calls onUninstall when uninstall clicked', async () => {
		const onUninstall = vi.fn();
		const installed = { ...baseMcp, installed: true };
		const { getByText } = render(McpCard, { props: { mcp: installed, onInstall: vi.fn(), onUninstall } });
		await fireEvent.click(getByText('Uninstall'));
		expect(onUninstall).toHaveBeenCalledOnce();
	});

	it('shows runtime badge', () => {
		const { getByText } = render(McpCard, { props: { mcp: baseMcp, onInstall: vi.fn(), onUninstall: vi.fn() } });
		expect(getByText('node')).toBeDefined();
	});

	it('shows description', () => {
		const { getByText } = render(McpCard, { props: { mcp: baseMcp, onInstall: vi.fn(), onUninstall: vi.fn() } });
		expect(getByText('Search code, read issues/PRs.')).toBeDefined();
	});
});
