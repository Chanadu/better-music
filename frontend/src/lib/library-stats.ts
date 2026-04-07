import type { LibraryData } from './library-cache';

export type HeaderCountType = 'artists' | 'queuedAlbums' | 'doneAlbums';

export type LibraryStats = {
	totalArtists: number;
	totalAlbums: number;
	queuedAlbums: number;
	doneAlbums: number;
};

export const LIBRARY_DATA_CHANGED_EVENT = 'library-data-changed';

export const getLibraryStats = (data: LibraryData): LibraryStats => {
	const doneAlbums = data.albums.filter((album) => album.listened).length;

	return {
		totalArtists: data.artists.length,
		totalAlbums: data.albums.length,
		queuedAlbums: data.albums.length - doneAlbums,
		doneAlbums,
	};
};

export const getHeaderCountMeta = (
	type: HeaderCountType,
	stats: LibraryStats,
): { value: number; label: string } => {
	switch (type) {
		case 'artists':
			return {
				value: stats.totalArtists,
				label: stats.totalArtists === 1 ? 'Artist' : 'Artists',
			};
		case 'queuedAlbums':
			return {
				value: stats.queuedAlbums,
				label: stats.queuedAlbums === 1 ? 'Queued Album' : 'Queued Albums',
			};
		case 'doneAlbums':
			return {
				value: stats.doneAlbums,
				label: stats.doneAlbums === 1 ? 'Done Album' : 'Done Albums',
			};
	}
};
