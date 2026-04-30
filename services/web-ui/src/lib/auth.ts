import { browser } from '$app/environment';
import { writable, derived } from 'svelte/store';
import type { User } from './types';

// Set VITE_API_URL in your .env to point at the deployed mcpapiserver.
// Falls back to localhost for local development.
const API_URL = (import.meta.env.VITE_API_URL as string | undefined) ?? 'http://localhost:8080';
const isTauri = browser && '__TAURI_INTERNALS__' in window;

type AuthResult = {
	token: string;
	user: User;
};

export const user = writable<User | null>(null);
export const authLoading = writable(false);
export const rememberMe = writable(true);

export const isAuthenticated = derived(user, ($u) => $u !== null);

async function invokeDesktop<T>(command: string, args: Record<string, unknown>): Promise<T> {
	const { invoke } = await import('@tauri-apps/api/core');
	return invoke<T>(command, args);
}

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
		if (isTauri) {
			const currentUser = await invokeDesktop<User>('auth_me', { token });
			user.set(currentUser);
			return;
		}

		const res = await fetch(`${API_URL}/api/auth/me`, {
			headers: { Authorization: `Bearer ${token}` }
		});
		if (res.ok) {
			const { user: u } = await res.json();
			user.set(u);
		} else if (res.status === 401) {
			clearToken();
		}
	} catch {
		// Network unavailable — stay logged-out; local SQLite still works.
	}
}

export async function signIn(email: string, password: string, remember = true): Promise<void> {
	authLoading.set(true);
	try {
		if (isTauri) {
			const data = await invokeDesktop<AuthResult>('auth_login', { email, password });
			setToken(data.token, remember);
			user.set({ id: data.user.id, name: data.user.name, email: data.user.email });
			return;
		}

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
		if (isTauri) {
			const data = await invokeDesktop<AuthResult>('auth_register', { name, email, password });
			setToken(data.token, true);
			user.set({ id: data.user.id, name: data.user.name, email: data.user.email });
			return;
		}

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

export async function updateProfile(name: string, email: string): Promise<void> {
	const token = getToken();
	if (!token) {
		throw new Error('You need to sign in again before updating your account.');
	}

	authLoading.set(true);
	try {
		if (isTauri) {
			const updated = await invokeDesktop<User>('auth_update_profile', { token, name, email });
			user.set(updated);
			return;
		}

		const res = await fetch(`${API_URL}/api/auth/me`, {
			method: 'PATCH',
			headers: {
				'Content-Type': 'application/json',
				Authorization: `Bearer ${token}`
			},
			body: JSON.stringify({ name, email })
		});

		if (!res.ok) {
			const data = await res.json().catch(() => ({}));
			throw new Error(data.error ?? `HTTP ${res.status}: Profile update failed`);
		}

		const data = await res.json();
		if (!data.user) {
			throw new Error('Invalid response: missing user data');
		}

		user.set({ id: data.user.id, name: data.user.name, email: data.user.email });
	} catch (error) {
		const msg = error instanceof Error ? error.message : 'Profile update failed';
		throw new Error(msg);
	} finally {
		authLoading.set(false);
	}
}

export async function changePassword(currentPassword: string, newPassword: string): Promise<void> {
	const token = getToken();
	if (!token) {
		throw new Error('You need to sign in again before changing your password.');
	}

	authLoading.set(true);
	try {
		if (isTauri) {
			await invokeDesktop<void>('auth_change_password', {
				token,
				currentPassword,
				newPassword
			});
			return;
		}

		const res = await fetch(`${API_URL}/api/auth/password`, {
			method: 'PATCH',
			headers: {
				'Content-Type': 'application/json',
				Authorization: `Bearer ${token}`
			},
			body: JSON.stringify({
				current_password: currentPassword,
				new_password: newPassword
			})
		});

		if (!res.ok) {
			const data = await res.json().catch(() => ({}));
			throw new Error(data.error ?? `HTTP ${res.status}: Password update failed`);
		}
	} catch (error) {
		const msg = error instanceof Error ? error.message : 'Password update failed';
		throw new Error(msg);
	} finally {
		authLoading.set(false);
	}
}

export function signOut(): void {
	clearToken();
	user.set(null);
}
