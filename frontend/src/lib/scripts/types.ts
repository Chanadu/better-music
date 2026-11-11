import type { Album as ApiAlbum, Artist as ApiArtist, SpotifyAlbumArtist } from './generated/api';

type OptionalFields<T, K extends keyof T> = Omit<T, K> & Partial<Pick<T, K>>;

export type Album = OptionalFields<
	ApiAlbum,
	'comment' | 'cover_url' | 'listened_at' | 'rating' | 'spotify_id' | 'year' | '_deleted'
>;

export type Artist = OptionalFields<ApiArtist, 'cover_url' | 'spotify_id' | '_deleted'>;

export type CreateArtistRequest = Pick<Artist, 'name' | 'cover_url' | 'spotify_id'>;
export type UpdateArtistRequest = Partial<CreateArtistRequest>;

type AlbumFields = Omit<Album, 'id' | 'created_at' | 'edited_at' | '_deleted'>;
export type UpdateAlbumRequest = Pick<AlbumFields, 'artist_id'> & Partial<Omit<AlbumFields, 'artist_id'>>;
export type CreateAlbumRequest = UpdateAlbumRequest & Pick<AlbumFields, 'title'>;

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
