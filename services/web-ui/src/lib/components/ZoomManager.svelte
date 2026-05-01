<script lang="ts">
	import { browser } from '$app/environment';
	import { onMount, onDestroy } from 'svelte';
	import { zoomLevel } from '$lib/stores';
	import { get } from 'svelte/store';

	const STORAGE_KEY = 'mach1_zoom_level';
	const MIN_ZOOM = 0.5;
	const MAX_ZOOM = 2.0;
	const STEP = 0.1;
	let indicatorVisible = false;
	let indicatorTimer: ReturnType<typeof setTimeout> | null = null;

	function getSavedZoom(): number {
		try {
			const saved = localStorage.getItem(STORAGE_KEY);
			if (saved) {
				const v = parseFloat(saved);
				if (!isNaN(v) && v >= MIN_ZOOM && v <= MAX_ZOOM) return v;
			}
		} catch {}
		return 1.0;
	}

	function saveZoom(v: number) {
		try { localStorage.setItem(STORAGE_KEY, String(v)); } catch {}
	}

	function applyZoom(v: number) {
		v = Math.round(v * 100) / 100;
		v = Math.max(MIN_ZOOM, Math.min(MAX_ZOOM, v));
		zoomLevel.set(v);
		document.documentElement.style.zoom = String(v);
		saveZoom(v);
		showIndicator(v);
	}

	function showIndicator(v: number) {
		indicatorVisible = true;
		if (indicatorTimer) clearTimeout(indicatorTimer);
		indicatorTimer = setTimeout(() => { indicatorVisible = false; }, 1200);
	}

	function zoomIn() { applyZoom(get(zoomLevel) + STEP); }
	function zoomOut() { applyZoom(get(zoomLevel) - STEP); }
	function zoomReset() { applyZoom(1.0); }

	function handleKeydown(e: KeyboardEvent) {
		const mod = e.metaKey || e.ctrlKey;
		if (!mod) return;

		if (e.key === '=' || e.key === '+') {
			e.preventDefault();
			zoomIn();
		} else if (e.key === '-') {
			e.preventDefault();
			zoomOut();
		} else if (e.key === '0') {
			e.preventDefault();
			zoomReset();
		}
	}

	onMount(() => {
		applyZoom(getSavedZoom());
		window.addEventListener('keydown', handleKeydown);
	});

	onDestroy(() => {
		window.removeEventListener('keydown', handleKeydown);
		if (indicatorTimer) clearTimeout(indicatorTimer);
	});
</script>

{#if indicatorVisible}
	<div class="zoom-indicator" aria-live="polite">
		<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/><line x1="11" y1="8" x2="11" y2="14"/><line x1="8" y1="11" x2="14" y2="11"/></svg>
		<span>{Math.round(get(zoomLevel) * 100)}%</span>
	</div>
{/if}

<style>
	.zoom-indicator {
		position: fixed;
		bottom: 80px;
		right: 24px;
		z-index: 9999;
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 10px 16px;
		border-radius: 10px;
		background: rgba(15, 15, 18, 0.92);
		border: 1px solid rgba(255, 255, 255, 0.08);
		backdrop-filter: blur(12px);
		font-size: 13px;
		font-weight: 600;
		color: rgba(255, 255, 255, 0.8);
		pointer-events: none;
		animation: zoom-fadein 0.15s ease-out;
		box-shadow: 0 4px 20px rgba(0,0,0,0.4);
	}
	@keyframes zoom-fadein {
		from { opacity: 0; transform: translateY(8px); }
		to { opacity: 1; transform: translateY(0); }
	}
</style>
