import { albumsApi, ApiError, artistsApi } from './api';
import { getCurrentUserId } from './auth';
import { getPendingMutations, updatePendingMutation } from './database-cache';
import { reconcileMutationResponse } from './database';
import { compareMutationOrder } from './database-reconciliation';
import type { PendingMutation, SuccessfulMutationResponse } from './types';

type ProcessorDependencies = {
	account: () => number | null;
	online: () => boolean;
	read: typeof getPendingMutations;
	update: typeof updatePendingMutation;
	send: (mutation: PendingMutation) => Promise<SuccessfulMutationResponse>;
	reconcile: (userId: number, mutationId: string, response: SuccessfulMutationResponse) => Promise<unknown>;
};

export type QueueResult = { completed: number; remaining: number; paused: boolean };

// Never coalesce create/update/delete sequences: each persisted request retains its own key.
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

const blocked = (mutation: PendingMutation, queue: PendingMutation[]) => {
	if (
		queue.some(
			(other) =>
				other.entity === mutation.entity &&
				other.entityId === mutation.entityId &&
				compareMutationOrder(other, mutation) < 0,
		)
	)
		return true;
	if (mutation.entity === 'artist' && mutation.operation === 'delete') {
		// Album changes, including their deletes, must finish before deleting their artist.
		return queue.some((other) => other.entity === 'album' && other.payload.artist_id === mutation.entityId);
	}
	if (mutation.entity === 'album') {
		return queue.some(
			(other) =>
				other.entity === 'artist' &&
				other.entityId === mutation.payload.artist_id &&
				other.operation !== 'delete' &&
				(typeof mutation.payload.artist_id !== 'number' ||
					other.status !== 'pending' ||
					compareMutationOrder(other, mutation) < 0),
		);
	}
	return false;
};

export const createQueueProcessor = (dependencies: ProcessorDependencies) => {
	let running: Promise<QueueResult> | null = null;
	const run = async (): Promise<QueueResult> => {
		const userId = dependencies.account();
		const result: QueueResult = { completed: 0, remaining: 0, paused: false };
		if (userId === null) return { ...result, paused: true };
		while (true) {
			const queue = (await dependencies.read(userId)).sort(compareMutationOrder);
			result.remaining = queue.length;
			if (dependencies.account() !== userId || !dependencies.online()) return { ...result, paused: true };
			const mutation = queue.find((item) => item.status === 'pending' && !blocked(item, queue));
			if (!mutation) return result;
			const unresolved =
				(mutation.operation !== 'create' && typeof mutation.entityId !== 'number') ||
				(mutation.entity === 'album' && typeof mutation.payload.artist_id !== 'number') ||
				(mutation.operation !== 'create' && !mutation.baseVersion);
			if (unresolved) {
				await dependencies.update(userId, mutation.mutationId, (item) => ({
					...item,
					status: 'failed',
					failure: { message: 'Missing server record or artist dependency' },
				}));
				continue;
			}
			const claimed = await dependencies.update(userId, mutation.mutationId, (item) => {
				if (item.status !== 'pending') return item;
				return { ...item, status: 'syncing', attempts: item.attempts + 1, failure: undefined };
			});
			if (!claimed || claimed.status !== 'syncing') continue;
			try {
				if (dependencies.account() !== userId || !dependencies.online()) {
					await dependencies.update(userId, claimed.mutationId, (item) => ({ ...item, status: 'pending' }));
					return { ...result, paused: true };
				}
				const response = await dependencies.send(claimed);
				if (dependencies.account() !== userId) return { ...result, paused: true };
				await dependencies.reconcile(userId, claimed.mutationId, response);
				result.completed++;
			} catch (error) {
				const permanent =
					error instanceof ApiError &&
					error.status >= 400 &&
					error.status < 500 &&
					![401, 408, 429].includes(error.status);
				await dependencies.update(userId, claimed.mutationId, (item) => ({
					...item,
					status:
						permanent ?
							error.status === 412 ?
								'conflict'
							:	'failed'
						:	'pending',
					failure: {
						message: error instanceof Error ? error.message : String(error),
						...(error instanceof ApiError ? { status: error.status, body: error.body } : {}),
					},
				}));
				if (!permanent) return { ...result, paused: true };
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

// Automatic triggers and coordination between tabs are added in later steps.
export const processMutationQueue = createQueueProcessor({
	account: getCurrentUserId,
	online: () => typeof navigator !== 'undefined' && navigator.onLine,
	read: getPendingMutations,
	update: updatePendingMutation,
	send: sendQueuedMutation,
	reconcile: reconcileMutationResponse,
});
