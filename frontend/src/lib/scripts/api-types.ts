export type AccountResponse = {
	email: string;
};

export type ApiErrorResponse = {
	error: string;
};

export type CreateAlbumRequest = {
	artist_id: string;
	comment?: string | null;
	cover_url?: string | null;
	listened?: boolean;
	listened_at?: string | null;
	rating?: number | null;
	spotify_id?: string | null;
	title: string;
	year?: number | null;
};

export type CreateArtistRequest = {
	cover_url?: string | null;
	name: string;
	spotify_id?: string | null;
};

export type DeleteAccountRequest = {
	password: string;
};

export type MessageResponse = {
	message: string;
};

export type SpotifyAlbumArtist = {
	id: string;
	name: string;
};

export type SpotifyAlbumSearchResult = {
	artists: SpotifyAlbumArtist[];
	id: string;
	images: SpotifyImage[];
	name: string;
	release_date: string;
};

export type SpotifyArtistSearchResult = {
	id: string;
	images: SpotifyImage[];
	name: string;
};

export type SpotifyImage = {
	height: number;
	url: string;
	width: number;
};

export type UpdateAlbumRequest = {
	artist_id: string;
	comment?: string | null;
	cover_url?: string | null;
	listened?: boolean;
	listened_at?: string | null;
	rating?: number | null;
	spotify_id?: string | null;
	title?: string;
	year?: number | null;
};

export type UpdateArtistRequest = {
	cover_url?: string | null;
	name?: string;
	spotify_id?: string | null;
};

export type UpdateEmailRequest = {
	email: string;
	password: string;
};

export type UpdatePasswordRequest = {
	current_password: string;
	new_password: string;
};

export type Album = {
	artist_id: string;
	comment?: string | null;
	cover_url?: string | null;
	created_at: string;
	id: string;
	listened: boolean;
	listened_at?: string | null;
	rating?: number | null;
	spotify_id?: string | null;
	title: string;
	edited_at: string;
	_deleted?: boolean;
	year?: number | null;
};

export type Artist = {
	cover_url?: string | null;
	created_at: string;
	id: string;
	name: string;
	spotify_id?: string | null;
	edited_at: string;
	_deleted?: boolean;
};
