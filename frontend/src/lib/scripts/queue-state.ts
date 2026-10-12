import { compareMutationOrder } from './database-reconciliation';
import type { PendingMutation } from './types';

export const hasUnknownOutcome = (mutation: PendingMutation) =>
	mutation.outcomeUnknown ??
	(mutation.status === 'syncing' ||
		(mutation.attempts > 0 && ['connection', 'storage', 'retryable'].includes(mutation.failure?.kind ?? '')));

export const getBlockingMutations = (mutation: PendingMutation, queue: readonly PendingMutation[]) =>
	queue.filter((other) => {
		if (other.mutationId === mutation.mutationId) return false;
		const earlier = compareMutationOrder(other, mutation) < 0;
		if (other.entity === mutation.entity && other.entityId === mutation.entityId) return earlier;
		if (mutation.entity === 'artist')
			return (
				mutation.operation === 'delete' &&
				other.entity === 'album' &&
				other.payload.artist_id === mutation.entityId
			);
		if (other.entity !== 'artist' || other.entityId !== mutation.payload.artist_id) return false;
		if (other.operation === 'delete') return mutation.operation !== 'delete' && other.status !== 'pending';
		return typeof mutation.payload.artist_id !== 'number' || other.status !== 'pending' || earlier;
	});

export const getMutationStatus = (mutation: PendingMutation, queue: readonly PendingMutation[]) =>
	mutation.status === 'pending' && !hasUnknownOutcome(mutation) && getBlockingMutations(mutation, queue).length ?
		'blocked'
	:	mutation.status;

const needsRecovery = (item: PendingMutation) =>
	['failed', 'conflict', 'blocked'].includes(item.status) || hasUnknownOutcome(item);

export const getChangesNeedingAttention = (queue: readonly PendingMutation[]) =>
	queue.filter((item) => needsRecovery(item) || getBlockingMutations(item, queue).some(needsRecovery));

export const hasActiveQueueWork = (queue: readonly PendingMutation[]) =>
	queue.some((item) => hasUnknownOutcome(item) || ['pending', 'syncing'].includes(getMutationStatus(item, queue)));

export const getQueueCounts = (queue: readonly PendingMutation[]) => {
	const counts = { pending: 0, syncing: 0, failed: 0, conflict: 0, blocked: 0, unknown: 0 };
	for (const item of queue) {
		counts[getMutationStatus(item, queue)]++;
		if (hasUnknownOutcome(item)) counts.unknown++;
	}
	return counts;
};
