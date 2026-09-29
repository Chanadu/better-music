import type { Album, Artist } from './api-types';

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

export type SpotifyArtistCredit = {
	id: string;
	name: string;
};
