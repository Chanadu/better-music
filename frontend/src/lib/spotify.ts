// Spotify Client Credentials Flow (client-side)
// Requires PUBLIC_SPOTIFY_CLIENT_ID and PUBLIC_SPOTIFY_CLIENT_SECRET in your frontend .env
// (Astro only exposes PUBLIC_* vars to browser JS)

const CLIENT_ID = import.meta.env.PUBLIC_SPOTIFY_CLIENT_ID as string | undefined;
const CLIENT_SECRET = import.meta.env.PUBLIC_SPOTIFY_CLIENT_SECRET as string | undefined;

let accessToken: string | null = null;
let tokenExpiry = 0;

const ensureToken = async (): Promise<string> => {
	if (!CLIENT_ID || !CLIENT_SECRET) {
		console.error('Spotify credentials missing');
		throw new Error('Spotify is not configured. (Credentials missing)');
	}

	if (accessToken && Date.now() < tokenExpiry) return accessToken;

	try {
		const basic = btoa(`${CLIENT_ID}:${CLIENT_SECRET}`);
		const res = await fetch('https://accounts.spotify.com/api/token', {
			method: 'POST',
			headers: {
				Authorization: `Basic ${basic}`,
				'Content-Type': 'application/x-www-form-urlencoded',
			},
			body: 'grant_type=client_credentials',
		});

		if (!res.ok) {
			const text = await res.text().catch(() => '');
			throw new Error(`Spotify auth failed (${res.status}): ${text || 'Invalid credentials'}`);
		}

		const data = (await res.json()) as { access_token: string; expires_in: number };
		accessToken = data.access_token;
		tokenExpiry = Date.now() + data.expires_in * 1000 - 30_000;
		return accessToken!;
	} catch (err) {
		accessToken = null; // Clear on error
		throw err;
	}
};

const spotifyFetch = async (endpoint: string): Promise<unknown> => {
	try {
		const token = await ensureToken();
		const res = await fetch(`https://api.spotify.com/v1${endpoint}`, {
			headers: { Authorization: `Bearer ${token}` },
		});

		if (res.status === 401) {
			accessToken = null; // Forces token refresh next time
			throw new Error('Spotify session expired. Please search again.');
		}

		if (!res.ok) throw new Error(`Spotify API error: ${res.status}`);
		return res.json();
	} catch (err) {
		console.error('Spotify Fetch Error:', err);
		throw err;
	}
};

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

export const searchArtists = async (query: string): Promise<SpotifyArtist[]> => {
	const data = (await spotifyFetch(`/search?q=${encodeURIComponent(query)}&type=artist&limit=6`)) as {
		artists: { items: SpotifyArtist[] };
	};
	return data.artists.items;
};

export const searchAlbums = async (query: string): Promise<SpotifyAlbum[]> => {
	const data = (await spotifyFetch(`/search?q=${encodeURIComponent(query)}&type=album&limit=6`)) as {
		albums: { items: SpotifyAlbum[] };
	};
	return data.albums.items;
};

export const getArtistBySpotifyID = async (spotifyID: string): Promise<SpotifyArtist> => {
	return spotifyFetch(`/artists/${encodeURIComponent(spotifyID)}`) as Promise<SpotifyArtist>;
};

export const getAlbumBySpotifyID = async (spotifyID: string): Promise<SpotifyAlbum> => {
	return spotifyFetch(`/albums/${encodeURIComponent(spotifyID)}`) as Promise<SpotifyAlbum>;
};
