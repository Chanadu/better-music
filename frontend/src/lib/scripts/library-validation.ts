export function isLibraryName(value: unknown): value is string {
	if (typeof value !== 'string') return false;
	const name = value.trim();
	return name.length > 0 && name.length <= 4096;
}

export function isAlbumRating(value: unknown): value is number {
	return typeof value === 'number' && Number.isInteger(value) && value >= 1 && value <= 10;
}

export function isAlbumYear(value: unknown): value is number {
	return typeof value === 'number' && Number.isInteger(value) && value >= -2147483648 && value <= 2147483647;
}
