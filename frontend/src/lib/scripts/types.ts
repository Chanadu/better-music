import type { Album, Artist, SpotifyAlbumArtist } from './api-types';

export type * from './api-types';

export type LibraryEntity = 'artist' | 'album';

export type EntityId = string;

export type RecordSyncStatus = 'pending' | 'syncing' | 'failed' | 'paused' | 'synced';

export type DatabaseData = {
	artists: Artist[];
	albums: Album[];
	loadedAt: number;
};

export type SpotifyRow = {
	id: string;
	name: string;
	meta?: string;
	imageUrl?: string;
	artists?: SpotifyArtistCredit[];
	releaseYear?: string;
};

export type SpotifyArtistCredit = SpotifyAlbumArtist;
