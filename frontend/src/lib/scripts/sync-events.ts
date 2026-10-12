import { syncConfig } from './sync-config';

export type SyncChange = {
	userId: number;
	type: 'library' | 'processor' | 'queue';
	state?: 'syncing' | 'idle' | 'paused';
	ownerId?: string;
};

const listeners = new Set<(change: SyncChange) => void>();
let channel: BroadcastChannel | null = null;
let initialized = false;

const receive = (value: unknown) => {
	if (!value || typeof value !== 'object') return;

	const change = value as SyncChange;

	if (!Number.isSafeInteger(change.userId) || change.userId <= 0) return;
	if (change.type !== 'library' && change.type !== 'processor' && change.type !== 'queue') return;

	for (const listener of listeners) listener(change);
};

const initialize = () => {
	if (initialized || typeof window === 'undefined') return;

	initialized = true;

	try {
		channel = new BroadcastChannel(syncConfig.changeChannel);
		channel.onmessage = (event) => receive(event.data);
	} catch {}

	window.addEventListener('storage', (event) => {
		if (event.key !== syncConfig.changeStorageKey || !event.newValue) return;

		try {
			receive(JSON.parse(event.newValue));
		} catch {}
	});
};

export const subscribeSyncChanges = (listener: (change: SyncChange) => void) => {
	initialize();
	listeners.add(listener);

	return () => listeners.delete(listener);
};

export const notifySyncChange = (change: SyncChange) => {
	initialize();
	receive(change);

	try {
		if (channel) channel.postMessage(change);
		else if (typeof localStorage !== 'undefined') {
			localStorage.setItem(
				syncConfig.changeStorageKey,
				JSON.stringify({ ...change, nonce: crypto.randomUUID() }),
			);
		}
	} catch {}
};
