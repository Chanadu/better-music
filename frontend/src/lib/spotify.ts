import { apiFetch, getErrorMessage } from './api';

export type SpotifyArtist = {
	id: string;
	name: string;
	images: Array<{ url: string; width: number; height: number }>;
};

export type SpotifyAlbum = {
	id: string;
	name: string;
	artists: Array<{ id: string; name: string }>;
	images: Array<{ url: string; width: number; height: number }>;
	release_date: string;
};

const spotifyFetch = async <T>(endpoint: string): Promise<T> => {
	const response = await apiFetch(endpoint, { method: 'GET' }, { auth: true });

	if (!response.ok) {
		throw new Error(await getErrorMessage(response, 'Spotify request failed.'));
	}

	return response.json() as Promise<T>;
};

export const searchArtists = async (query: string): Promise<SpotifyArtist[]> => {
	return spotifyFetch<SpotifyArtist[]>(`/api/spotify/search/artists?q=${encodeURIComponent(query)}&limit=6`);
};

export const searchAlbums = async (query: string): Promise<SpotifyAlbum[]> => {
	return spotifyFetch<SpotifyAlbum[]>(`/api/spotify/search/albums?q=${encodeURIComponent(query)}&limit=6`);
};
