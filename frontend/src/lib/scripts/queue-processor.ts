import { albumsApi, ApiError, artistsApi } from './api';
import { AuthenticationError, getCurrentUserId } from './auth';
import { getPendingMutations, updatePendingMutation } from './database-cache';
import { withSyncOwnership, sendWithTimeout, SyncOwnershipLostError, type QueueOwnership } from './sync-coordination';
import { reconcileMutationResponse } from './database';
import { compareMutationOrder } from './database-reconciliation';
import type { PendingMutation, SuccessfulMutationResponse } from './types';
import { syncConfig } from './sync-config';

type ProcessorDependencies = {
	account: () => number | null;
	online: () => boolean;
	read: typeof getPendingMutations;
	update: typeof updatePendingMutation;
	send: (mutation: PendingMutation, signal?: AbortSignal) => Promise<SuccessfulMutationResponse>;
	reconcile: typeof reconcileMutationResponse;
};

export type QueueResult = {
	completed: number;
	remaining: number;
	paused: boolean;
	pauseReason?: 'connection' | 'authentication' | 'storage' | 'account' | 'coordination';
};

export const retryAfterDeadline = (value: string | null, now: number): number | undefined => {
	if (!value) return undefined;

	const trimmed = value.trim();

	if (trimmed.length > 0 && Array.from(trimmed).every((character) => character >= '0' && character <= '9')) {
		const deadline = now + Number(trimmed) * syncConfig.millisecondsPerSecond;

		return Number.isSafeInteger(deadline) ? deadline : undefined;
	}

	const weekday = trimmed.slice(0, 3);

	if (
		trimmed[3] !== ',' ||
		weekday.length !== 3 ||
		!Array.from(weekday.toLowerCase()).every((character) => character >= 'a' && character <= 'z')
	)
		return undefined;

	const deadline = Date.parse(trimmed);
	return Number.isFinite(deadline) ? Math.max(now, deadline) : undefined;
};

type Failure = NonNullable<PendingMutation['failure']>;

const classifyFailure = (error: unknown): Failure => {
	const message = error instanceof Error ? error.message : String(error);

	if (error instanceof AuthenticationError) return { kind: 'authentication', message };

	if (error instanceof ApiError) {
		const kind: Failure['kind'] =
			error.status === syncConfig.http.authentication ? 'authentication'
			: error.status === syncConfig.http.conflict ? 'conflict'
			: syncConfig.http.retryable.includes(error.status) ? 'retryable'
			: syncConfig.http.invalidData.includes(error.status) ? 'invalid-data'
			: 'permanent';

		return {
			kind,
			message,
			status: error.status,
			body: error.body,
			...(error.retryAfter ? { retryAfter: error.retryAfter } : {}),
		};
	}

	if (error instanceof TypeError || (error instanceof Error && error.name === 'AbortError')) {
		return { kind: 'connection', message };
	}

	return { kind: 'permanent', message };
};

export const sendQueuedMutation = async (
	mutation: PendingMutation,
	signal?: AbortSignal,
): Promise<SuccessfulMutationResponse> => {
	const options = { mutationId: mutation.mutationId, signal };

	if (mutation.entity === 'artist' && mutation.operation === 'create') {
		return { entity: 'artist', operation: 'create', record: await artistsApi.create(mutation.payload, options) };
	}

	if (mutation.entity === 'album' && mutation.operation === 'create') {
		if (typeof mutation.payload.artist_id !== 'number') throw new Error('Artist has no server ID');

		return {
			entity: 'album',
			operation: 'create',
			record: await albumsApi.create({ ...mutation.payload, artist_id: mutation.payload.artist_id }, options),
		};
	}

	if (typeof mutation.entityId !== 'number' || !mutation.baseVersion)
		throw new Error('Record has no server ID or version');

	const versioned = { ...options, baseVersion: mutation.baseVersion };

	if (mutation.entity === 'artist') {
		if (mutation.operation === 'update') {
			return {
				entity: 'artist',
				operation: 'update',
				record: await artistsApi.update(mutation.entityId, mutation.payload, versioned),
			};
		}

		await artistsApi.delete(mutation.entityId, versioned);
	} else {
		if (typeof mutation.payload.artist_id !== 'number') throw new Error('Artist has no server ID');

		if (mutation.operation === 'update') {
			return {
				entity: 'album',
				operation: 'update',
				record: await albumsApi.update(
					mutation.entityId,
					{ ...mutation.payload, artist_id: mutation.payload.artist_id },
					versioned,
				),
			};
		}

		await albumsApi.delete(mutation.entityId, mutation.payload.artist_id, versioned);
	}

	return { entity: mutation.entity, operation: 'delete' };
};

const blocked = (mutation: PendingMutation, queue: PendingMutation[]) =>
	queue.some((other) => {
		const earlier = compareMutationOrder(other, mutation) < 0;

		if (other.entity === mutation.entity && other.entityId === mutation.entityId) return earlier;

		if (mutation.entity === 'artist') {
			return (
				mutation.operation === 'delete' &&
				other.entity === 'album' &&
				other.payload.artist_id === mutation.entityId
			);
		}

		if (other.entity !== 'artist' || other.entityId !== mutation.payload.artist_id) return false;

		if (other.operation === 'delete') return mutation.operation !== 'delete' && other.status !== 'pending';

		return typeof mutation.payload.artist_id !== 'number' || other.status !== 'pending' || earlier;
	});

// Sending and retry scheduling use the same dependency and rate-limit rules.
export const getQueueWork = (queue: PendingMutation[], now = Date.now()) => {
	const rateLimit = queue.reduce(
		(deadline, item) =>
			item.failure?.status === syncConfig.http.rateLimit ? Math.max(deadline, item.nextAttemptAt ?? 0) : deadline,
		0,
	);
	const eligible = queue.filter((item) => item.status === 'pending' && !blocked(item, queue));
	const attemptAt = (item: PendingMutation) => Math.max(item.nextAttemptAt ?? 0, rateLimit);
	const retries = eligible.filter((item) => item.nextAttemptAt !== undefined || rateLimit > now);
	return {
		ready: eligible.find((item) => attemptAt(item) <= now),
		retryAt: retries.length ? Math.min(...retries.map(attemptAt)) : null,
	};
};

const missingDependency = (mutation: PendingMutation) =>
	(mutation.operation !== 'create' && (typeof mutation.entityId !== 'number' || !mutation.baseVersion)) ||
	(mutation.entity === 'album' && typeof mutation.payload.artist_id !== 'number');

const failurePauseReason = (failure: Failure) =>
	failure.kind === 'connection' || failure.kind === 'authentication' ? failure.kind : undefined;

const withFailure = (mutation: PendingMutation, failure: Failure): PendingMutation => {
	const updated = { ...mutation, failure, nextAttemptAt: undefined };
	if (failurePauseReason(failure)) return { ...updated, status: 'pending' };
	if (failure.kind === 'conflict') return { ...updated, status: 'conflict' };
	if (failure.kind !== 'retryable') return { ...updated, status: 'failed' };

	const retryFailures = (mutation.retryFailures ?? 0) + 1;

	const now = Date.now();
	const delay = syncConfig.retryDelaysMs[Math.min(retryFailures - 1, syncConfig.retryDelaysMs.length - 1)];

	return {
		...updated,
		status: retryFailures >= syncConfig.maximumFailures ? 'failed' : 'pending',
		retryFailures,
		nextAttemptAt: Math.max(retryAfterDeadline(failure.retryAfter ?? null, now) ?? 0, now + delay),
	};
};

export const createQueueProcessor = (dependencies: ProcessorDependencies) => {
	let running: Promise<QueueResult> | null = null;

	const run = async (existingOwnership?: QueueOwnership): Promise<QueueResult> => {
		const userId = dependencies.account();
		const result: QueueResult = { completed: 0, remaining: 0, paused: false };
		const pause = (pauseReason: NonNullable<QueueResult['pauseReason']>): QueueResult => ({
			...result,
			paused: true,
			pauseReason,
		});

		if (userId === null) return pause('authentication');

		const interrupted = (): QueueResult['pauseReason'] =>
			dependencies.account() !== userId ? 'account'
			: !dependencies.online() ? 'connection'
			: undefined;

		const runOwned = async (ownership: QueueOwnership): Promise<QueueResult> => {
			const update = (key: string, change: Parameters<typeof dependencies.update>[2]) =>
				dependencies.update(userId, key, change, ownership.check);

			while (true) {
				if (ownership.signal.aborted) throw new SyncOwnershipLostError();
				const queue = (await dependencies.read(userId)).sort(compareMutationOrder);
				result.remaining = queue.length;
				const pauseReason = interrupted();
				if (pauseReason) return pause(pauseReason);
				const { ready: mutation } = getQueueWork(queue);

				if (!mutation) return result;

				const unresolved = missingDependency(mutation);
				if (unresolved || (mutation.retryFailures ?? 0) >= syncConfig.maximumFailures) {
					await update(mutation.mutationId, (item) => ({
						...item,
						status: 'failed',
						failure:
							unresolved ?
								{
									kind: 'invalid-data',
									message: 'Missing server record or artist dependency',
								}
							:	item.failure,
					}));

					continue;
				}

				let didClaim = false;

				const claimed = await update(mutation.mutationId, (item) => {
					if (item.status !== 'pending' || (item.nextAttemptAt ?? 0) > Date.now()) return item;
					didClaim = true;
					return { ...item, status: 'syncing', attempts: item.attempts + 1 };
				});

				if (!didClaim || !claimed) continue;

				const beforeSend = interrupted();
				if (beforeSend) {
					await update(claimed.mutationId, (item) => ({
						...item,
						status: 'pending',
						attempts: item.attempts - 1,
					}));

					return pause(beforeSend);
				}

				let response: SuccessfulMutationResponse;

				try {
					response = await sendWithTimeout(ownership, (signal) => dependencies.send(claimed, signal));
				} catch (error) {
					if (error instanceof SyncOwnershipLostError || ownership.signal.aborted)
						throw new SyncOwnershipLostError();
					const failure = classifyFailure(error);

					await update(claimed.mutationId, (item) => withFailure(item, failure));

					const pauseReason = failurePauseReason(failure);

					if (pauseReason) return pause(pauseReason);

					continue;
				}

				if (dependencies.account() !== userId) return pause('account');

				try {
					ownership.check();
					await dependencies.reconcile(userId, claimed.mutationId, response, ownership.check);

					result.completed++;
				} catch (error) {
					if (error instanceof SyncOwnershipLostError || ownership.signal.aborted)
						throw new SyncOwnershipLostError();
					await update(claimed.mutationId, (item) => ({
						...item,
						status: 'pending',
						nextAttemptAt: undefined,
						failure: {
							kind: 'storage',
							message: error instanceof Error ? error.message : String(error),
						},
					}));

					return pause('storage');
				}
			}
		};
		try {
			if (existingOwnership) return await runOwned(existingOwnership);

			return (
				(await withSyncOwnership(userId, () => interrupted() === undefined, runOwned)) ??
				pause(interrupted() ?? 'coordination')
			);
		} catch (error) {
			if (error instanceof SyncOwnershipLostError) return pause(interrupted() ?? 'coordination');

			throw error;
		}
	};
	return (ownership?: QueueOwnership) => {
		if (!running)
			running = run(ownership).finally(() => {
				running = null;
			});

		return running;
	};
};

export const processMutationQueue = createQueueProcessor({
	account: getCurrentUserId,
	online: () => typeof navigator !== 'undefined' && navigator.onLine,
	read: getPendingMutations,
	update: updatePendingMutation,
	send: sendQueuedMutation,
	reconcile: reconcileMutationResponse,
});
