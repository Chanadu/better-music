import { albumsApi, ApiError, artistsApi } from './api';
import { AuthenticationError, getCurrentUserId } from './auth';
import { getPendingMutations, updatePendingMutation } from './database-cache';
import { reconcileMutationResponse } from './database';
import { compareMutationOrder } from './database-reconciliation';
import type { PendingMutation, SuccessfulMutationResponse } from './types';
import { syncConfig } from './sync-config';

type ProcessorDependencies = {
	account: () => number | null;
	online: () => boolean;
	read: typeof getPendingMutations;
	update: typeof updatePendingMutation;
	send: (mutation: PendingMutation) => Promise<SuccessfulMutationResponse>;
	reconcile: (userId: number, mutationId: string, response: SuccessfulMutationResponse) => Promise<unknown>;
};

export type QueueResult = {
	completed: number;
	remaining: number;
	paused: boolean;
	pauseReason?: 'connection' | 'authentication' | 'storage' | 'account';
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

export const sendQueuedMutation = async (mutation: PendingMutation): Promise<SuccessfulMutationResponse> => {
	const options = { mutationId: mutation.mutationId };

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

	const run = async (): Promise<QueueResult> => {
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

		while (true) {
			const queue = (await dependencies.read(userId)).sort(compareMutationOrder);
			result.remaining = queue.length;
			const pauseReason = interrupted();
			if (pauseReason) return pause(pauseReason);
			const now = Date.now();

			const rateLimit = queue.reduce(
				(deadline, item) =>
					item.failure?.status === syncConfig.http.rateLimit ?
						Math.max(deadline, item.nextAttemptAt ?? 0)
						: deadline,
				0,
			);

			const eligible = queue.filter((item) => item.status === 'pending' && !blocked(item, queue));
			const mutation = eligible.find((item) => (item.nextAttemptAt ?? 0) <= now && rateLimit <= now);

			if (!mutation) {
				if (!eligible.length) return result;

				const next = Math.max(rateLimit, Math.min(...eligible.map((item) => item.nextAttemptAt ?? now)));

				await new Promise<void>((resolve) =>
					setTimeout(resolve, Math.min(syncConfig.interruptCheckIntervalMs, Math.max(1, next - now))),
				);

				continue;
			}

			const unresolved = missingDependency(mutation);
			if (unresolved || (mutation.retryFailures ?? 0) >= syncConfig.maximumFailures) {
				await dependencies.update(userId, mutation.mutationId, (item) => ({
					...item,
					status: 'failed',
					failure:
						unresolved ?
							{ kind: 'invalid-data', message: 'Missing server record or artist dependency' }
							: item.failure,
				}));

				continue;
			}

			let didClaim = false;

			const claimed = await dependencies.update(userId, mutation.mutationId, (item) => {
				if (item.status !== 'pending' || (item.nextAttemptAt ?? 0) > Date.now()) return item;
				didClaim = true;
				return { ...item, status: 'syncing', attempts: item.attempts + 1 };
			});

			if (!didClaim || !claimed) continue;

			const beforeSend = interrupted();
			if (beforeSend) {
				await dependencies.update(userId, claimed.mutationId, (item) => ({
					...item,
					status: 'pending',
					attempts: item.attempts - 1,
				}));

				return pause(beforeSend);
			}

			let response: SuccessfulMutationResponse;

			try {
				response = await dependencies.send(claimed);
			} catch (error) {
				const failure = classifyFailure(error);

				await dependencies.update(userId, claimed.mutationId, (item) => withFailure(item, failure));

				const pauseReason = failurePauseReason(failure);

				if (pauseReason) return pause(pauseReason);

				continue;
			}

			if (dependencies.account() !== userId) return pause('account');

			try {
				await dependencies.reconcile(userId, claimed.mutationId, response);

				result.completed++;
			} catch (error) {
				await dependencies.update(userId, claimed.mutationId, (item) => ({
					...item,
					status: 'pending',
					nextAttemptAt: undefined,
					failure: { kind: 'storage', message: error instanceof Error ? error.message : String(error) },
				}));

				return pause('storage');
			}
		}
	};
	return () => {
		if (!running)
			running = run().finally(() => {
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
