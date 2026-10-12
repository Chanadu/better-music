import { ApiError, artistsApi, albumsApi } from './api';
import { getCurrentUserId, getSessionGeneration, getValidAccessToken } from './auth';
import { getStoredSyncState, getSyncMetadata, updateLibraryAndOutbox, updatePendingMutation } from './database-cache';
import { compareMutationOrder } from './database-reconciliation';
import { getBlockingMutations, hasUnknownOutcome } from './queue-state';
import { withSyncOwnership, sendWithTimeout, type QueueOwnership } from './sync-coordination';
import { notifySyncChange } from './sync-events';
import type { Artist, Album, PendingMutation, UpdateArtistRequest, UpdateAlbumMutationPayload } from './types';

type Record = Artist | Album;
export type RecoveryPlan = {
	userId: number;
	session: string | null;
	generation: string;
	revision: number;
	mutation: PendingMutation;
	baseline: Record | null;
	removeIds: string[];
	messages: string[];
	action: 'discard' | 'correct';
};

export const mutationLabel = (item: PendingMutation) => {
	const record = item.after ?? item.before;
	if (record) return 'name' in record ? record.name : record.title;
	if (item.operation !== 'delete') {
		const text = item.entity === 'artist' ? item.payload.name : item.payload.title;
		if (text != null) return text;
	}
	return `${item.entity === 'artist' ? 'Artist' : 'Album'} ${item.entityId}`;
};

const sameRecord = (left: PendingMutation, right: PendingMutation) =>
	left.entity === right.entity && left.entityId === right.entityId;
const laterEdits = (item: PendingMutation, queue: PendingMutation[]) =>
	queue.filter((other) => sameRecord(item, other) && compareMutationOrder(other, item) > 0);
const removedByDiscard = (item: PendingMutation, queue: PendingMutation[]) => {
	if (item.operation !== 'create') return [item];
	const same = queue.filter((other) => sameRecord(item, other));
	if (item.entity === 'album') return same;
	return [...same, ...queue.filter((other) => other.entity === 'album' && other.payload.artist_id === item.entityId)];
};

const affectedByRecovery = (item: PendingMutation, queue: PendingMutation[], action: RecoveryPlan['action']) =>
	action === 'discard' && item.operation === 'create' ?
		removedByDiscard(item, queue)
	:	[item, ...laterEdits(item, queue)];

const requireUser = () => {
	const userId = getCurrentUserId();
	if (userId === null) throw new Error('Sign in to the account that owns this change.');
	return userId;
};

const withRecovery = async <T>(
	userId: number,
	session: string | null,
	work: (ownership: QueueOwnership) => Promise<T>,
) => {
	const result = await withSyncOwnership(
		userId,
		() => getCurrentUserId() === userId && getSessionGeneration() === session,
		async (ownership) => ({ paused: false, value: await work(ownership) }),
	);
	if (!result) throw new Error('The account changed or recovery was interrupted. Reopen the change and try again.');
	return result.value;
};

const requireConnection = async (ownership: QueueOwnership) => {
	if (!navigator.onLine)
		throw new Error(
			'Reconnect before reconciling a request or restoring an older change without a saved snapshot.',
		);
	if (!(await getValidAccessToken())) throw new Error('Sign in to the same account before reconciling this change.');
	ownership.check();
};

const reconcileAffected = async (userId: number, affected: PendingMutation[], ownership: QueueOwnership) => {
	const { reconcileUncertainMutation } = await import('./queue-processor');
	for (const item of [...affected].sort(compareMutationOrder)) {
		if (!hasUnknownOutcome(item)) continue;
		await requireConnection(ownership);
		await reconcileUncertainMutation(userId, item.mutationId, ownership);
	}
};

const readBaseline = async (item: PendingMutation, ownership: QueueOwnership): Promise<Record | null> => {
	if (item.operation === 'create') return null;
	if (
		item.status === 'conflict' &&
		item.failure?.body &&
		typeof item.failure.body === 'object' &&
		'current' in item.failure.body
	)
		return (item.failure.body as { current: Record | null }).current;
	if (item.before !== undefined) return item.before;
	if (typeof item.entityId !== 'number') return null;
	await requireConnection(ownership);
	try {
		return await sendWithTimeout<Record>(ownership, (signal) =>
			item.entity === 'artist' ?
				artistsApi.get(item.entityId as number, signal)
			:	albumsApi.get(item.entityId as number, (item.payload as { artist_id: number }).artist_id, signal),
		);
	} catch (error) {
		if (error instanceof ApiError && error.status === 404) return null;
		throw error;
	}
};

export const prepareRecovery = async (mutationId: string, action: RecoveryPlan['action']): Promise<RecoveryPlan> => {
	const userId = requireUser();
	const session = getSessionGeneration();
	return withRecovery(userId, session, async (ownership) => {
		let state = await getStoredSyncState(userId);
		let item = state.mutations.find((item) => item.mutationId === mutationId);
		if (!item) throw new Error('This change has already completed or was removed.');
		const affected = affectedByRecovery(item, state.mutations, action);
		await reconcileAffected(userId, [...affected, ...getBlockingMutations(item, state.mutations)], ownership);
		state = await getStoredSyncState(userId);
		item = state.mutations.find((item) => item.mutationId === mutationId);
		if (!item)
			throw new Error(
				'The original request completed on the server. Review the remaining changes before choosing recovery.',
			);
		if (action === 'correct' && (item.status !== 'failed' || item.operation === 'delete'))
			throw new Error(
				'Only a failed create or update can be corrected here. Conflicts have their own resolution workflow.',
			);
		const revision = state.library?.revision ?? 0;
		const baseline = await readBaseline(item, ownership);
		if (action === 'correct' && item.operation === 'update' && !baseline)
			throw new Error(
				'The original server record is unavailable. Resolve its deletion or discard this edit before correcting it.',
			);
		const latest = await getStoredSyncState(userId);
		if ((latest.library?.revision ?? 0) !== revision)
			throw new Error(
				'Changes were saved while preparing recovery. Open the action again to review the latest consequences.',
			);
		const removed = action === 'discard' ? removedByDiscard(item, state.mutations) : [];
		const later = laterEdits(item, state.mutations);
		const messages = [
			action === 'correct' ?
				'Corrected data is submitted as a new change. Unchanged retries resend the original request.'
			:	`Remove this ${item.operation} from the queue.`,
		];
		if (action === 'discard' && item.operation === 'create') {
			messages.push(
				`Remove ${removed.length} queued operation${removed.length === 1 ? '' : 's'}, including this record’s later edits and deletions${item.entity === 'artist' ? ' and its dependent album changes' : ''}. The local record${item.entity === 'artist' ? ' and dependent local albums' : ''} will be removed.`,
			);
			for (const other of removed.filter((other) => other.mutationId !== item!.mutationId))
				messages.push(`Remove ${other.operation}: ${mutationLabel(other)}.`);
		} else {
			messages.push(
				later.length ?
					`Preserve ${later.length} later edit${later.length === 1 ? '' : 's'} and replay them over the ${action === 'correct' ? 'corrected' : 'restored'} record.`
				:	'No later edits to this record will be removed.',
			);
			if (!baseline && item.operation !== 'create')
				messages.push(
					'The original record is unavailable. Later updates will stay blocked; later deletions can still be retried.',
				);
			messages.push(
				'Dependent records are preserved. If their required parent is unavailable, their requests remain blocked.',
			);
		}
		if (later.some((item) => item.attempts > 0))
			messages.push(
				'Previously attempted later requests with known outcomes receive new IDs when rebased. Their failures remain visible until retried.',
			);
		ownership.check();
		return {
			userId,
			session,
			generation: (await getSyncMetadata(userId)).generation,
			revision,
			mutation: item,
			baseline,
			removeIds: removed.map((item) => item.mutationId),
			messages,
			action,
		};
	});
};

const applyRecordMutation = (record: Record | null, item: PendingMutation): Record | null => {
	if (item.operation === 'delete') return null;
	if (item.operation === 'create')
		return {
			...item.payload,
			id: item.entityId,
			version: null,
			created_at: new Date(item.createdAt).toISOString(),
			...(item.entity === 'album' ? { listened: item.payload.listened ?? false } : {}),
		} as Record;
	if (!record) return null;
	const updated = { ...record, ...item.payload } as Record;
	if (item.entity === 'album' && item.payload.listened === false)
		Object.assign(updated, { rating: null, listened_at: null });
	return updated;
};

export const correctionRecord = (plan: RecoveryPlan): Record | null =>
	plan.mutation.after ?? applyRecordMutation(plan.baseline, plan.mutation);

const resetRequest = (item: PendingMutation, newKey: boolean): PendingMutation => ({
	...item,
	mutationId: newKey ? crypto.randomUUID() : item.mutationId,
	attempts: newKey ? 0 : item.attempts,
	status: 'pending',
	outcomeUnknown: false,
	blockedReason: undefined,
	failure: undefined,
	retryFailures: 0,
	nextAttemptAt: undefined,
});

const rebaseMutation = (item: PendingMutation, record: Record | null, targetId: string): PendingMutation => {
	const rebased = {
		...(item.mutationId !== targetId && item.attempts > 0 ? resetRequest(item, true) : item),
		status: item.status,
		failure: item.failure,
		before: record,
	};
	if (record?.version && item.operation !== 'create') rebased.baseVersion = record.version;
	if (!record && item.operation === 'update') {
		rebased.status = 'blocked';
		rebased.blockedReason = 'The original record is unavailable. Resolve the missing record or discard this edit.';
	}
	return rebased;
};

const validateCorrection = (item: PendingMutation, payload: UpdateArtistRequest | UpdateAlbumMutationPayload) => {
	const text =
		item.entity === 'artist' ?
			(payload as UpdateArtistRequest).name
		:	(payload as UpdateAlbumMutationPayload).title;
	if (!text?.trim()) throw new Error(`${item.entity === 'artist' ? 'Name' : 'Title'} is required.`);
	if (item.entity === 'album') {
		const album = payload as UpdateAlbumMutationPayload;
		if (album.artist_id !== item.payload.artist_id)
			throw new Error('An album’s artist cannot be changed during recovery.');
		if (album.year != null && !Number.isInteger(album.year)) throw new Error('Year must be an integer.');
		if (album.rating != null && (!Number.isInteger(album.rating) || album.rating < 1 || album.rating > 10))
			throw new Error('Rating must be between 1 and 10.');
	}
};

export const applyRecovery = async (plan: RecoveryPlan, payload?: UpdateArtistRequest | UpdateAlbumMutationPayload) =>
	withRecovery(plan.userId, plan.session, async (ownership) => {
		const state = await getStoredSyncState(plan.userId);
		if (state.metadata?.generation !== plan.generation)
			throw new Error('The local cache changed. Open recovery again.');
		let corrected: Record | null = null;
		const result = await updateLibraryAndOutbox(
			plan.userId,
			(current) => {
				if ((current.library.revision ?? 0) !== plan.revision)
					throw new Error(
						'New changes were saved. Open recovery again to review what will be preserved or removed.',
					);
				const target = current.mutations.find((item) => item.mutationId === plan.mutation.mutationId);
				if (!target) throw new Error('The change has already completed or was removed.');
				const affected = affectedByRecovery(target, current.mutations, plan.action);
				if (affected.some(hasUnknownOutcome))
					throw new Error('Reconcile all affected requests with unknown outcomes before recovery.');
				let mutations = current.mutations.filter((item) => !plan.removeIds.includes(item.mutationId));
				let library = { ...current.library };
				if (plan.action === 'discard' && target.operation === 'create') {
					library.artists = library.artists.filter(
						(item) => target.entity !== 'artist' || item.id !== target.entityId,
					);
					library.albums = library.albums.filter((item) =>
						target.entity === 'artist' ? item.artist_id !== target.entityId : item.id !== target.entityId,
					);
					return { library, mutations };
				}
				if (plan.action === 'correct') {
					if (!payload) throw new Error('Corrected fields are required.');
					validateCorrection(target, payload);
					const replacement = {
						...resetRequest(target, true),
						payload: { ...target.payload, ...payload },
					} as PendingMutation;
					if (replacement.operation !== 'delete') {
						if (replacement.entity === 'artist')
							replacement.payload.name = replacement.payload.name!.trim();
						else replacement.payload.title = replacement.payload.title!.trim();
					}
					mutations = mutations.map((item) => (item.mutationId === target.mutationId ? replacement : item));
				}
				let record = plan.baseline;
				for (const item of [...mutations].sort(compareMutationOrder)) {
					if (!sameRecord(target, item) || compareMutationOrder(item, target) < 0) continue;
					const index = mutations.findIndex((other) => other.mutationId === item.mutationId);
					const rebased = rebaseMutation(item, record, target.mutationId);
					record = applyRecordMutation(record, rebased);
					mutations[index] = {
						...rebased,
						after: item.operation === 'delete' ? null : (record ?? item.after),
					};
					if (plan.action === 'correct' && compareMutationOrder(item, target) === 0) corrected = record;
				}
				if (target.entity === 'artist')
					library.artists = [
						...library.artists.filter((item) => item.id !== target.entityId),
						...(record ? [record as Artist] : []),
					];
				else
					library.albums = [
						...library.albums.filter((item) => item.id !== target.entityId),
						...(record ? [record as Album] : []),
					];
				if (target.entity === 'artist' && !plan.baseline && target.operation !== 'create')
					mutations = mutations.map((item) =>
						(
							item.entity === 'album' &&
							item.operation !== 'delete' &&
							item.payload.artist_id === target.entityId
						) ?
							{
								...item,
								status: 'blocked',
								blockedReason:
									'The required artist is unavailable. Restore the artist or discard this album change.',
							}
						:	item,
					);
				return { library, mutations };
			},
			ownership.check,
			plan.generation,
		);
		notifySyncChange({ userId: plan.userId, type: 'queue' });
		return { ...result, corrected: corrected as Record | null };
	});

export const retryFailedMutation = async (mutationId: string) => {
	const userId = requireUser();
	const session = getSessionGeneration();
	await withRecovery(userId, session, async (ownership) => {
		const state = await getStoredSyncState(userId);
		const item = state.mutations.find((item) => item.mutationId === mutationId);
		if (!item) return;
		await reconcileAffected(userId, [item], ownership);
		await updatePendingMutation(
			userId,
			mutationId,
			(current) => {
				if (hasUnknownOutcome(current)) throw new Error('The request outcome is still unknown.');
				if (current.status !== 'failed')
					throw new Error(
						'Only failed requests can be retried. Resolve blocked dependencies or conflicts first.',
					);
				return resetRequest(current, false);
			},
			ownership.check,
		);
	});
	notifySyncChange({ userId, type: 'queue' });
};

export const reconcileUnknownMutation = async (mutationId: string) => {
	const userId = requireUser();
	await withRecovery(userId, getSessionGeneration(), async (ownership) => {
		const state = await getStoredSyncState(userId);
		const item = state.mutations.find((item) => item.mutationId === mutationId);
		if (item) await reconcileAffected(userId, [...getBlockingMutations(item, state.mutations), item], ownership);
	});
	notifySyncChange({ userId, type: 'queue' });
};
