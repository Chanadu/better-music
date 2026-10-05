import type {
	Album,
	Artist,
	CreateAlbumRequest,
	CreateArtistRequest,
	SpotifyAlbumArtist,
	UpdateAlbumRequest,
	UpdateArtistRequest,
} from './api-types';

export type * from './api-types';

export type DatabaseData = { artists: Artist[]; albums: Album[]; loadedAt: number };

export type MutationEntity = 'artist' | 'album';
export type MutationOperation = 'create' | 'update' | 'delete';
export type MutationStatus = 'pending' | 'syncing' | 'conflict' | 'failed';
export type TemporaryId = `local:${string}`;
export type EntityId = number | TemporaryId;
export type RecordSyncStatus = MutationStatus | 'synced';

type RecordIdentity = { id: number; version: number } | { id: TemporaryId; version: null };

export type LocalArtist = Omit<Artist, 'id' | 'version'> & RecordIdentity;
export type LocalAlbum = Omit<Album, 'id' | 'artist_id' | 'version'> &
	RecordIdentity & {
		artist_id: EntityId;
	};

export type LocalDatabaseData = {
	artists: LocalArtist[];
	albums: LocalAlbum[];
	loadedAt: number;
};

export type CreateAlbumMutationPayload = Omit<CreateAlbumRequest, 'artist_id'> & {
	artist_id: EntityId;
};

export type UpdateAlbumMutationPayload = Omit<UpdateAlbumRequest, 'artist_id'> & {
	artist_id: EntityId;
};

type MutationPayloads = {
	artist: {
		create: CreateArtistRequest;
		update: UpdateArtistRequest;
		delete: null;
	};
	album: {
		create: CreateAlbumMutationPayload;
		update: UpdateAlbumMutationPayload;
		delete: { artist_id: EntityId };
	};
};

type PendingMutationBase<Entity extends MutationEntity, Operation extends MutationOperation> = {
	mutationId: string;
	userId: number;
	entity: Entity;
	operation: Operation;
	entityId: EntityId;
	payload: MutationPayloads[Entity][Operation];
	baseVersion?: number;
	createdAt: number;
	attempts: number;
	status: MutationStatus;
};

export type PendingMutation = {
	[Entity in MutationEntity]: {
		[Operation in MutationOperation]: PendingMutationBase<Entity, Operation>;
	}[MutationOperation];
}[MutationEntity];

export type SpotifyRow = {
	id: string;
	name: string;
	meta?: string;
	imageUrl?: string;
	artists?: SpotifyArtistCredit[];
	releaseYear?: string;
};

export type SpotifyArtistCredit = SpotifyAlbumArtist;
