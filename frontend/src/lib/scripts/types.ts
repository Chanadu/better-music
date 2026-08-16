export type * from './api-types';

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
