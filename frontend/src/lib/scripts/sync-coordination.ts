import { recoverInterruptedMutations, getSyncMetadata, setSyncActivity } from './database-cache';
import { notifySyncChange } from './sync-events';
import { syncConfig } from './sync-config';

export class SyncOwnershipLostError extends Error {
	constructor() {
		super('Synchronization ownership was lost');
		this.name = 'SyncOwnershipLostError';
	}
}

export type QueueOwnership = { signal: AbortSignal; check: () => void };

export const withSyncOwnership = async <T extends { paused: boolean }>(
	userId: number,
	canContinue: () => boolean,
	work: (ownership: QueueOwnership) => Promise<T>,
): Promise<T | null> => {
	if (!navigator.locks) throw new Error('This browser does not support synchronization locks');

	const controller = new AbortController();
	const cancel = () => controller.abort();

	const monitor = setInterval(() => {
		if (!canContinue()) cancel();
	}, syncConfig.interruptCheckIntervalMs);

	window.addEventListener('pagehide', cancel);

	const runLocked = async () => {
		const ownerId = crypto.randomUUID();

		const check = () => {
			if (controller.signal.aborted || !canContinue()) throw new SyncOwnershipLostError();
		};

		const notify = (state: 'syncing' | 'idle' | 'paused') =>
			notifySyncChange({ userId, type: 'processor', state, ownerId });

		let state: 'idle' | 'paused' = 'paused';
		let pauseReason: string | undefined;
		const { generation } = await getSyncMetadata(userId);

		try {
			check();

			await recoverInterruptedMutations(userId, check);

			await setSyncActivity(userId, generation, 'syncing');
			notify('syncing');

			const result = await work({ signal: controller.signal, check });
			state = result.paused ? 'paused' : 'idle';

			pauseReason = (result as T & { pauseReason?: string }).pauseReason;
			return result;
		} catch (error) {
			pauseReason =
				!navigator.onLine ? 'Offline. Reconnect to resume synchronization.'
				: error instanceof Error ? error.message
				: 'Synchronization was interrupted.';
			throw error;
		} finally {
			controller.abort();
			await setSyncActivity(
				userId,
				generation,
				state,
				pauseReason,
				state === 'paused' ? pauseReason : undefined,
			).catch(console.error);
			notify(state);
		}
	};

	try {
		return await navigator.locks.request(syncConfig.lockPrefix + userId, { signal: controller.signal }, runLocked);
	} catch (error) {
		if (
			error instanceof SyncOwnershipLostError ||
			(controller.signal.aborted && error instanceof DOMException && error.name === 'AbortError')
		)
			return null;

		throw error;
	} finally {
		clearInterval(monitor);
		window.removeEventListener('pagehide', cancel);
	}
};

export const sendWithTimeout = async <T>(ownership: QueueOwnership, send: (signal: AbortSignal) => Promise<T>) => {
	ownership.check();

	const controller = new AbortController();

	let cancel!: () => void;
	const interrupted = new Promise<never>((_, reject) => {
		cancel = () => {
			controller.abort();
			reject(new DOMException('Synchronization request interrupted', 'AbortError'));
		};
	});

	ownership.signal.addEventListener('abort', cancel, { once: true });

	const timeout = setTimeout(cancel, syncConfig.requestTimeoutMs);

	try {
		return await Promise.race([send(controller.signal), interrupted]);
	} finally {
		clearTimeout(timeout);
		ownership.signal.removeEventListener('abort', cancel);
	}
};
