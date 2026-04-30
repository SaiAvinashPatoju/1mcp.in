import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
import StarRating from '../StarRating.svelte';

describe('StarRating', () => {
	it('renders with correct rating prop', () => {
		const { container } = render(StarRating, { props: { rating: 4 } });
		const paths = container.querySelectorAll('path');
		expect(paths.length).toBe(5);
	});

	it('renders with 0 rating', () => {
		const { container } = render(StarRating, { props: { rating: 0 } });
		const paths = container.querySelectorAll('path');
		expect(paths.length).toBe(5);
	});

	it('renders with 5 rating', () => {
		const { container } = render(StarRating, { props: { rating: 5 } });
		const paths = container.querySelectorAll('path');
		expect(paths.length).toBe(5);
	});

	it('renders with custom size', () => {
		const { container } = render(StarRating, { props: { rating: 3, size: 20 } });
		const svg = container.querySelector('svg');
		expect(svg?.getAttribute('width')).toBe('20');
	});
});
