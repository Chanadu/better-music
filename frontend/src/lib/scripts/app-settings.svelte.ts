import { persistentStorage } from './storage';

type SettingDefinition<T> = {
	defaultValue: T;
	isValid: (value: unknown) => value is T;
};

function defineSetting<T>(defaultValue: T, isValid: (value: unknown) => value is T): SettingDefinition<T> {
	return { defaultValue, isValid };
}

const definitions = {
	useNativeDropdowns: defineSetting(false, (value): value is boolean => typeof value === 'boolean'),
};

type SettingValues = {
	[Key in keyof typeof definitions]: (typeof definitions)[Key] extends SettingDefinition<infer Value> ? Value : never;
};

const storageKey = 'bettermusic:settings';

function defaultValues(): SettingValues {
	return Object.fromEntries(
		Object.entries(definitions).map(([key, definition]) => [key, definition.defaultValue]),
	) as SettingValues;
}

class AppSettings {
	values = $state(defaultValues());
	#loaded = false;

	load() {
		if (this.#loaded) return;

		const saved = persistentStorage.getJson<Record<string, unknown>>(storageKey);
		const values = this.values as Record<string, unknown>;

		if (saved) {
			for (const [key, definition] of Object.entries(definitions)) {
				if (definition.isValid(saved[key])) values[key] = saved[key];
			}
		}

		this.#loaded = true;
	}

	set<Key extends keyof SettingValues>(key: Key, value: SettingValues[Key]) {
		this.values[key] = value;
		persistentStorage.setJson(storageKey, this.values);
	}
}

export const appSettings = new AppSettings();
