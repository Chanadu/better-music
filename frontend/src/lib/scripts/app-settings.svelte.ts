import { persistentStorage } from './storage';

type SettingDefinition<T> = {
	defaultValue: T;
	isValid: (value: unknown) => value is T;
};

function defineSetting<T>(defaultValue: T, isValid: (value: unknown) => value is T): SettingDefinition<T> {
	return { defaultValue, isValid };
}

export const defaultRatingLabels = [
	"just don't",
	'terrible',
	'bad',
	'meh',
	'okay',
	'solid',
	'good',
	'great',
	'incredible',
	'whoa',
];

export const defaultRatingColors = [
	'#DC2626',
	'#EF4444',
	'#F97316',
	'#FB923C',
	'#FBBF24',
	'#B5CC38',
	'#84CC16',
	'#22C55E',
	'#10B981',
	'#14B8A6',
];

const definitions = {
	useNativeDropdowns: defineSetting(false, (value): value is boolean => typeof value === 'boolean'),
	ratingLabels: defineSetting(
		defaultRatingLabels,
		(value): value is string[] =>
			Array.isArray(value) &&
			value.length === 10 &&
			value.every((label) => typeof label === 'string' && label.length <= 40),
	),
	ratingColors: defineSetting(
		defaultRatingColors,
		(value): value is string[] =>
			Array.isArray(value) && value.length === 10 && value.every((color) => /^#[\dA-F]{6}$/i.test(color)),
	),
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
