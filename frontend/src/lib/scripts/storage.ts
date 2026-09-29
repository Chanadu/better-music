const createSafeStorage = (getStorage: () => Storage) => {
	const get = (key: string) => {
		try {
			return getStorage().getItem(key);
		} catch {
			return null;
		}
	};

	const set = (key: string, value: string) => {
		try {
			getStorage().setItem(key, value);
		} catch {}
	};

	const remove = (key: string) => {
		try {
			getStorage().removeItem(key);
		} catch {}
	};

	const getJson = <T>(key: string): T | null => {
		try {
			const value = get(key);
			return value === null ? null : (JSON.parse(value) as T);
		} catch {
			return null;
		}
	};

	const setJson = (key: string, value: unknown) => {
		try {
			set(key, JSON.stringify(value));
		} catch {}
	};

	return { get, set, remove, getJson, setJson };
};

export const persistentStorage = createSafeStorage(() => localStorage);
