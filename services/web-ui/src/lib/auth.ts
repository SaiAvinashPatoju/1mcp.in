import { writable, derived, get } from 'svelte/store';
import type { User } from './types';

export const user = writable<User | null>(null);
export const authLoading = writable(false);

export const isAuthenticated = derived(user, ($u) => $u !== null);

export async function signIn(email: string, _password: string) {
	authLoading.set(true);
	await new Promise((r) => setTimeout(r, 600));
	user.set({
		id: 'u1',
		name: email
			.split('@')[0]
			.replace(/[._]/g, ' ')
			.replace(/\b\w/g, (c) => c.toUpperCase()),
		email
	});
	authLoading.set(false);
}

export async function signUp(name: string, email: string, _password: string) {
	authLoading.set(true);
	await new Promise((r) => setTimeout(r, 800));
	user.set({ id: 'u1', name, email });
	authLoading.set(false);
}

export function signOut() {
	user.set(null);
}
