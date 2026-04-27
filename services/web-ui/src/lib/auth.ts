import { writable, derived } from 'svelte/store';
import type { User } from './types';

// Set VITE_API_URL in your .env to point at the deployed mcpapiserver.
// Falls back to localhost for local development.
const API_URL = (import.meta.env.VITE_API_URL as string | undefined) ?? 'http://localhost:8080';

export const user = writable<User | null>(null);
export const authLoading = writable(false);
export const rememberMe = writable(true);

export const isAuthenticated = derived(user, ($u) => $u !== null);

function getStorage(): Storage | null {
	if (typeof localStorage === 'undefined') return null;
	return localStorage;
}

function getToken(): string | null {
	if (typeof localStorage === 'undefined') return null;
	// Check localStorage first (remember me), then sessionStorage
	return localStorage.getItem('mcp_token') ?? sessionStorage.getItem('mcp_token');
}

function setToken(token: string, remember: boolean): void {
	if (typeof localStorage === 'undefined') return;
	if (remember) {
		localStorage.setItem('mcp_token', token);
		sessionStorage.removeItem('mcp_token');
	} else {
		sessionStorage.setItem('mcp_token', token);
		localStorage.removeItem('mcp_token');
	}
}

function clearToken(): void {
	if (typeof localStorage === 'undefined') return;
	localStorage.removeItem('mcp_token');
	sessionStorage.removeItem('mcp_token');
}

/** Restore session from storage on page load. */
export async function restoreSession(): Promise<void> {
	const token = getToken();
	if (!token) return;
	try {
		const res = await fetch(`${API_URL}/api/auth/me`, {
			headers: { Authorization: `Bearer ${token}` }
		});
		if (res.ok) {
			const { user: u } = await res.json();
			user.set(u);
		} else {
			clearToken();
		}
	} catch {
		// Network unavailable — stay logged-out; local SQLite still works.
	}
}

export async function signIn(email: string, password: string, remember = true): Promise<void> {
	authLoading.set(true);
	try {
		const res = await fetch(`${API_URL}/api/auth/login`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ email, password })
		});
		
		if (!res.ok) {
			const data = await res.json().catch(() => ({}));
			throw new Error(data.error ?? `HTTP ${res.status}: Login failed`);
		}
		
		const data = await res.json();
		if (!data.token || !data.user) {
			throw new Error('Invalid response: missing token or user data');
		}
		
		setToken(data.token, remember);
		user.set({ id: data.user.id, name: data.user.name, email: data.user.email });
	} catch (error) {
		const msg = error instanceof Error ? error.message : 'Login failed';
		authLoading.set(false);
		throw new Error(msg);
	} finally {
		authLoading.set(false);
	}
}

export async function signUp(name: string, email: string, password: string): Promise<void> {
	authLoading.set(true);
	try {
		const res = await fetch(`${API_URL}/api/auth/register`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ name, email, password })
		});
		
		if (!res.ok) {
			const data = await res.json().catch(() => ({}));
			throw new Error(data.error ?? `HTTP ${res.status}: Registration failed`);
		}
		
		const data = await res.json();
		if (!data.token || !data.user) {
			throw new Error('Invalid response: missing token or user data');
		}
		
		setToken(data.token, true); // always remember on signup
		user.set({ id: data.user.id, name: data.user.name, email: data.user.email });
	} catch (error) {
		const msg = error instanceof Error ? error.message : 'Registration failed';
		authLoading.set(false);
		throw new Error(msg);
	} finally {
		authLoading.set(false);
	}
}

export function signOut(): void {
	clearToken();
	user.set(null);
}
