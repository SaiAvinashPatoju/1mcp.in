import { writable, derived } from 'svelte/store';
import type { User } from './types';

// Set VITE_API_URL in your .env to point at the deployed mcpapiserver.
// Falls back to localhost for local development.
const API_URL = (import.meta.env.VITE_API_URL as string | undefined) ?? 'http://localhost:8080';

export const user = writable<User | null>(null);
export const authLoading = writable(false);

export const isAuthenticated = derived(user, ($u) => $u !== null);

/** Restore session from localStorage on page load. */
export async function restoreSession(): Promise<void> {
	const token = typeof localStorage !== 'undefined' ? localStorage.getItem('mcp_token') : null;
	if (!token) return;
	try {
		const res = await fetch(`${API_URL}/api/auth/me`, {
			headers: { Authorization: `Bearer ${token}` }
		});
		if (res.ok) {
			const { user: u } = await res.json();
			user.set(u);
		} else {
			localStorage.removeItem('mcp_token');
		}
	} catch {
		// Network unavailable — stay logged-out; local SQLite still works.
	}
}

export async function signIn(email: string, password: string): Promise<void> {
	authLoading.set(true);
	try {
		const res = await fetch(`${API_URL}/api/auth/login`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ email, password })
		});
		const data = await res.json();
		if (!res.ok) throw new Error(data.error ?? 'Login failed');
		if (typeof localStorage !== 'undefined') localStorage.setItem('mcp_token', data.token);
		user.set({ id: data.user.id, name: data.user.name, email: data.user.email });
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
		const data = await res.json();
		if (!res.ok) throw new Error(data.error ?? 'Registration failed');
		if (typeof localStorage !== 'undefined') localStorage.setItem('mcp_token', data.token);
		user.set({ id: data.user.id, name: data.user.name, email: data.user.email });
	} finally {
		authLoading.set(false);
	}
}

export function signOut(): void {
	if (typeof localStorage !== 'undefined') localStorage.removeItem('mcp_token');
	user.set(null);
}
