import { albumsLibrary, artistsLibrary, getDatabaseData } from './database';
import { getCurrentUserId, getSessionGeneration } from './auth';
import type { ExportAlbum, ExportArtist } from './library-export';
import type { EntityId } from './types';

type ImportedLibrary = { artists: ExportArtist[]; albums: ExportAlbum[] };

export function parseLibraryImport(text: string): ImportedLibrary {
	let data: unknown;

	try {
		data = JSON.parse(text);
	} catch {
		throw new Error('Choose a valid JSON file.');
	}

	const fail = (message: string): never => {
		throw new Error(message);
	};

	const object = (value: unknown): Record<string, unknown> => {
		if (value !== null && typeof value === 'object' && !Array.isArray(value)) {
			return value as Record<string, unknown>;
		}
		return fail('Invalid library record.');
	};

	const root = object(data);

	if (root.application !== 'better-music' || root.format_version !== 2) {
		fail('Choose a Better Music JSON export (format version 2).');
	}

	const library = object(root.library);

	if (!Array.isArray(library.artists) || !Array.isArray(library.albums)) {
		fail('Missing artists or albums.');
	}

	const id = (value: unknown): EntityId => {
		if (typeof value === 'number' && Number.isSafeInteger(value) && value > 0) {
			return value;
		}

		if (typeof value === 'string' && /^local:.+/.test(value)) {
			return value as EntityId;
		}

		return fail('Invalid record ID.');
	};

	const name = (value: unknown): string => {
		if (typeof value === 'string' && value.trim()) {
			return value.trim();
		}
		return fail('Artist names and album titles cannot be empty.');
	};

	const nullableString = (value: unknown): string | null => {
		if (value == null) {
			return null;
		}

		if (typeof value === 'string') {
			return value;
		}

		return fail('Invalid text field.');
	};

	const date = (value: unknown): string | null => {
		const result = nullableString(value);

		if (result !== null && (!/^\d{4}-\d{2}-\d{2}T/.test(result) || !Number.isFinite(Date.parse(result)))) {
			fail('Invalid listening date.');
		}

		return result;
	};

	const artistIds = new Set<EntityId>();
	const artistNames = new Set<string>();

	const artists = (library.artists as unknown[]).map((value) => {
		const row = object(value);
		const artist = {
			id: id(row.id),
			name: name(row.name),
			cover_url: nullableString(row.cover_url),
			spotify_id: nullableString(row.spotify_id),
			created_at: '',
		};

		if (artistIds.has(artist.id) || artistNames.has(artist.name.toLowerCase())) {
			fail('Duplicate artist in the file.');
		}

		artistIds.add(artist.id);
		artistNames.add(artist.name.toLowerCase());
		return artist;
	});

	const albumIds = new Set<EntityId>();
	const albumNames = new Set<string>();

	const albums = (library.albums as unknown[]).map((value) => {
		const row = object(value);
		const albumId = id(row.id),
			artistId = id(row.artist_id),
			title = name(row.title);

		if (!artistIds.has(artistId)) {
			fail('An album references a missing artist.');
		}

		const key = JSON.stringify([artistId, title.toLowerCase()]);

		if (albumIds.has(albumId) || albumNames.has(key)) {
			fail('Duplicate album in the file.');
		}

		albumIds.add(albumId);
		albumNames.add(key);

		if (typeof row.listened !== 'boolean') {
			fail('Invalid listening status.');
		}

		if (
			row.rating != null &&
			(typeof row.rating !== 'number' || !Number.isInteger(row.rating) || row.rating < 1 || row.rating > 10)
		) {
			fail('Ratings must be integers from 1 to 10.');
		}

		if (
			row.year != null &&
			(typeof row.year !== 'number' ||
				!Number.isInteger(row.year) ||
				row.year < -2147483648 ||
				row.year > 2147483647)
		) {
			fail('Invalid album year.');
		}

		const listenedAt = date(row.listened_at);
		if (!row.listened && (row.rating != null || listenedAt !== null)) {
			fail('Unlistened albums cannot have ratings or listening dates.');
		}

		return {
			id: albumId,
			artist_id: artistId,
			title,
			cover_url: nullableString(row.cover_url),
			spotify_id: nullableString(row.spotify_id),
			comment: nullableString(row.comment),
			year: (row.year as number | null) ?? null,
			rating: (row.rating as number | null) ?? null,
			listened: row.listened as boolean,
			listened_at: listenedAt,
			created_at: '',
		};
	});

	return { artists, albums };
}

export async function importLibraryJson(text: string): Promise<{ artists: number; albums: number; skipped: number }> {
	const source = parseLibraryImport(text);
	const userId = getCurrentUserId(),
		session = getSessionGeneration();
	const existing = await getDatabaseData();

	if (userId === null || !existing) {
		throw new Error('Sign in and load your library before importing.');
	}

	const checkSession = () => {
		if (getCurrentUserId() !== userId || getSessionGeneration() !== session) {
			throw new Error('Your session changed. Please retry the import.');
		}
	};

	const result = { artists: 0, albums: 0, skipped: 0 };
	const artists = new Map(existing.artists.map((artist) => [artist.name.toLowerCase(), artist.id]));
	const albums = new Set(
		existing.albums.map((album) => JSON.stringify([album.artist_id, album.title.toLowerCase()])),
	);

	const mappedIds = new Map<EntityId, EntityId>();

	try {
		for (const artist of source.artists) {
			checkSession();
			let targetId = artists.get(artist.name.toLowerCase());

			if (targetId === undefined) {
				const { name, cover_url, spotify_id } = artist;
				targetId = (await artistsLibrary.create({ name, cover_url, spotify_id })).id;

				artists.set(name.toLowerCase(), targetId);
				result.artists++;
			} else {
				result.skipped++;
			}

			mappedIds.set(artist.id, targetId);
		}

		for (const album of source.albums) {
			checkSession();

			const artist_id = mappedIds.get(album.artist_id)!;
			const key = JSON.stringify([artist_id, album.title.toLowerCase()]);

			if (albums.has(key)) {
				result.skipped++;
				continue;
			}

			const { title, cover_url, spotify_id, year, listened, rating, comment, listened_at } = album;

			await albumsLibrary.create({
				artist_id,
				title,
				cover_url,
				spotify_id,
				year,
				listened,
				rating,
				comment,
				listened_at,
			});

			albums.add(key);
			result.albums++;
		}
	} catch (error) {
		let message = 'Please try again.';

		if (error instanceof Error) {
			message = error.message;
		}

		throw new Error(
			`Import stopped after adding ${result.artists} artists and ${result.albums} albums. You can retry the file; existing records are skipped. ${message}`,
		);
	}

	return result;
}
