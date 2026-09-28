import { persistentStorage } from './storage';

export const daisyThemes = [
	'light',
	'dark',
	'cupcake',
	'bumblebee',
	'emerald',
	'corporate',
	'synthwave',
	'retro',
	'cyberpunk',
	'valentine',
	'halloween',
	'garden',
	'forest',
	'aqua',
	'lofi',
	'pastel',
	'fantasy',
	'wireframe',
	'black',
	'luxury',
	'dracula',
	'cmyk',
	'autumn',
	'business',
	'acid',
	'lemonade',
	'night',
	'coffee',
	'winter',
	'dim',
	'nord',
	'sunset',
	'caramellatte',
	'abyss',
	'silk',
] as const;

type DaisyTheme = (typeof daisyThemes)[number];
export type ThemePreference = 'system' | 'bettermusic' | DaisyTheme;

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
	theme: defineSetting(
		'bettermusic' as ThemePreference,
		(value): value is ThemePreference =>
			value === 'system' || value === 'bettermusic' || daisyThemes.includes(value as DaisyTheme),
	),
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
	deviceTheme = $state<'light' | 'dark'>('light');
	#loaded = false;
	#systemTheme = typeof window === 'undefined' ? null : window.matchMedia('(prefers-color-scheme: dark)');

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
		this.#updateDeviceTheme();
		this.#applyTheme();
		this.#systemTheme?.addEventListener('change', () => {
			this.#updateDeviceTheme();
			if (this.values.theme === 'system') this.#applyTheme();
		});
	}

	set<Key extends keyof SettingValues>(key: Key, value: SettingValues[Key]) {
		this.values[key] = value;
		persistentStorage.setJson(storageKey, this.values);
		if (key === 'theme') this.#applyTheme();
	}

	#applyTheme() {
		if (typeof document === 'undefined') return;

		const theme = this.values.theme === 'system' ? this.deviceTheme : this.values.theme;

		document.documentElement.dataset.theme = theme;
		const background = getComputedStyle(document.documentElement).getPropertyValue('--color-base-100').trim();
		if (background)
			document.querySelector<HTMLMetaElement>('meta[name="theme-color"]')?.setAttribute('content', background);
	}

	#updateDeviceTheme() {
		this.deviceTheme = this.#systemTheme?.matches ? 'dark' : 'light';
	}
}

export const appSettings = new AppSettings();
