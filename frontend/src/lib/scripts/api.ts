import {
	deleteAccount,
	getAccount,
	getSpotifyAlbum,
	getSpotifyArtist,
	searchSpotifyAlbums,
	searchSpotifyArtists,
	updateAccountEmail,
	updateAccountPassword,
} from './generated/api';

export { ApiError } from './api-fetch';

export const accountApi = {
	get: getAccount,
	updateEmail: updateAccountEmail,
	updatePassword: updateAccountPassword,
	delete: deleteAccount,
};

export const spotifyApi = {
	getArtist: (id: string) => getSpotifyArtist(encodeURIComponent(id)),
	getAlbum: (id: string) => getSpotifyAlbum(encodeURIComponent(id)),
	searchArtists: (q: string, limit?: number) =>
		searchSpotifyArtists({ q, limit: limit === undefined ? undefined : String(limit) }),
	searchAlbums: (q: string, limit?: number) =>
		searchSpotifyAlbums({ q, limit: limit === undefined ? undefined : String(limit) }),
};
