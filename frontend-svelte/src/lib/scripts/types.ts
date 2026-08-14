export type * from './api-types';

export type SpotifyRow = {
	id: string;
	name: string;
	meta?: string;
	imageUrl?: string;
	artistId?: string;
	artistName?: string;
	releaseYear?: string;
};
