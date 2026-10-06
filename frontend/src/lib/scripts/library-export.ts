import { albumsApi, artistsApi } from './api';
import { getDatabaseData } from './database';
import type { DatabaseData, EntityId } from './types';

export const LIBRARY_EXPORT_FORMAT_VERSION = 2 as const;

export type LibraryExportSource = Pick<DatabaseData, 'artists' | 'albums'>;

export async function fetchFreshLibraryForExport(): Promise<LibraryExportSource> {
	const [artists, albums] = await Promise.all([artistsApi.list(), albumsApi.list()]);

	return { artists, albums };
}

export type ExportArtist = {
	id: EntityId;
	name: string;
	cover_url: string | null;
	spotify_id: string | null;
	created_at: string;
};

export type ExportAlbum = {
	id: EntityId;
	artist_id: EntityId;
	title: string;
	cover_url: string | null;
	year: number | null;
	spotify_id: string | null;
	listened: boolean;
	rating: number | null;
	comment: string | null;
	listened_at: string | null;
	created_at: string;
};

export type LibraryExportV2 = {
	format_version: typeof LIBRARY_EXPORT_FORMAT_VERSION;
	application: 'better-music';
	exported_at: string;
	library: {
		artists: ExportArtist[];
		albums: ExportAlbum[];
	};
};

export function buildLibraryExport(source: LibraryExportSource, exportedAt = new Date()): LibraryExportV2 {
	const artists = [...source.artists]
		.sort((first, second) => String(first.id).localeCompare(String(second.id), undefined, { numeric: true }))
		.map((artist): ExportArtist => ({
			id: artist.id,
			name: artist.name,
			cover_url: artist.cover_url ?? null,
			spotify_id: artist.spotify_id ?? null,
			created_at: artist.created_at,
		}));
	const albums = [...source.albums]
		.sort((first, second) => String(first.id).localeCompare(String(second.id), undefined, { numeric: true }))
		.map((album): ExportAlbum => ({
			id: album.id,
			artist_id: album.artist_id,
			title: album.title,
			cover_url: album.cover_url ?? null,
			year: album.year ?? null,
			spotify_id: album.spotify_id ?? null,
			listened: album.listened,
			rating: album.rating ?? null,
			comment: album.comment ?? null,
			listened_at: album.listened_at ?? null,
			created_at: album.created_at,
		}));

	return {
		format_version: LIBRARY_EXPORT_FORMAT_VERSION,
		application: 'better-music',
		exported_at: exportedAt.toISOString(),
		library: { artists, albums },
	};
}

export function createLibraryExportBlob(exportData: LibraryExportV2): Blob {
	return new Blob([`${JSON.stringify(exportData, null, 2)}\n`], { type: 'application/json' });
}

export async function downloadLibraryExport(): Promise<string> {
	const source = (await getDatabaseData()) ?? (await fetchFreshLibraryForExport());
	const exportedAt = new Date();
	const exportData = buildLibraryExport(source, exportedAt);
	const blob = createLibraryExportBlob(exportData);
	const filename = `better-music-library-${exportedAt.toISOString().slice(0, 10)}.json`;
	const url = URL.createObjectURL(blob);
	const link = document.createElement('a');

	link.href = url;
	link.download = filename;
	link.hidden = true;
	document.body.append(link);

	try {
		link.click();
	} finally {
		link.remove();
		setTimeout(() => URL.revokeObjectURL(url), 0);
	}

	return filename;
}
