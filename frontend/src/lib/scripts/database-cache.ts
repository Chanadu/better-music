const cachePrefix = 'betterMusicDatabaseData:';

export const databaseCacheKey = (userId: number) => `${cachePrefix}${userId}`;

export const clearStoredDatabaseCaches = () => {
	for (let index = sessionStorage.length - 1; index >= 0; index -= 1) {
		const key = sessionStorage.key(index);

		if (key?.startsWith(cachePrefix)) sessionStorage.removeItem(key);
	}
};
