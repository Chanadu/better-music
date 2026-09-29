import type { Album, Artist, SpotifyAlbumArtist } from './api-types';

export type * from './api-types';

export type DatabaseData = { artists: Artist[]; albums: Album[]; loadedAt: number };

export type SpotifyRow = {
	id: string;
	name: string;
	meta?: string;
	imageUrl?: string;
	artists?: SpotifyArtistCredit[];
	releaseYear?: string;
};

export type SpotifyArtistCredit = SpotifyAlbumArtist;
