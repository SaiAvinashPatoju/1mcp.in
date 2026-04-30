import { writable } from 'svelte/store';

export type ToastType = 'success' | 'error' | 'warning' | 'info';

export type ToastMessage = {
	id: number;
	message: string;
	type: ToastType;
};

function createToastStore() {
	const { subscribe, update } = writable<ToastMessage[]>([]);
	let nextId = 0;

	function add(message: string, type: ToastType, duration = 4000) {
		const id = nextId++;
		update((t) => [...t, { id, message, type }]);
		setTimeout(() => {
			update((t) => t.filter((m) => m.id !== id));
		}, duration);
	}

	return {
		subscribe,
		success: (msg: string) => add(msg, 'success'),
		error: (msg: string) => add(msg, 'error', 6000),
		warning: (msg: string) => add(msg, 'warning', 5000),
		info: (msg: string) => add(msg, 'info'),
		dismiss: (id: number) => update((t) => t.filter((m) => m.id !== id)),
	};
}

export const toast = createToastStore();
