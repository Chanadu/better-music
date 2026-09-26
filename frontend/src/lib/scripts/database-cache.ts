import { sessionCache } from './storage';

const cachePrefix = 'betterMusicDatabaseData:';

export const databaseCacheKey = (userId: number) => `${cachePrefix}${userId}`;

export const clearStoredDatabaseCaches = () => {
	sessionCache.keys().forEach((key) => {
		if (key.startsWith(cachePrefix)) sessionCache.remove(key);
	});
};
