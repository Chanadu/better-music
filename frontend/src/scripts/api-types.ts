export type ApiErrorResponse = {
	error?: string;
	message?: string;
};

export type AuthRequest = {
	email: string;
	password: string;
};

export type RefreshTokenRequest = {
	refresh_token: string;
};

export type TokenResponse = {
	access_token: string;
	refresh_token: string;
	token_type: string;
	expires_in: number;
};

export type Artist = {
	id: number;
	name: string;
	cover_url?: string;
	spotify_id?: string;
	created_at: string;
};

export type CreateArtistRequest = {
	name: string;
	cover_url?: string;
	spotify_id?: string;
};

export type UpdateArtistRequest = {
	name?: string;
	cover_url?: string;
	spotify_id?: string;
};

export type Album = {
	id: number;
	artist_id: number;
	title: string;
	cover_url?: string;
	year?: number;
	spotify_id?: string;
	listened: boolean;
	rating?: number;
	comment?: string;
	listened_at?: string;
	created_at: string;
};

export type CreateAlbumRequest = {
	artist_id: number;
	title: string;
	spotify_id?: string;
};

export type UpdateAlbumRequest = {
	artist_id: number;
	title?: string;
	cover_url?: string;
	year?: number;
	spotify_id?: string;
	listened?: boolean;
	rating?: number;
	comment?: string;
	listened_at?: string;
};

export type ArtistIDRequest = {
	artist_id: number;
};

export type SpotifyImage = {
	url: string;
	width: number;
	height: number;
};

export type SpotifyArtistSearchResult = {
	id: string;
	name: string;
	images: SpotifyImage[];
};

export type SpotifyAlbumArtist = {
	id: string;
	name: string;
};

export type SpotifyAlbumSearchResult = {
	id: string;
	name: string;
	artists: SpotifyAlbumArtist[];
	images: SpotifyImage[];
	release_date: string;
};

export type MessageResponse = {
	message: string;
};
