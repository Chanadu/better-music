import { onMount } from 'svelte';
import { persistentStorage } from './storage';

type SortPreference<T extends string> = {
	sort: T;
	reversed: boolean;
};

export function useSortPreference<T extends string>(
	key: () => string,
	validSorts: readonly T[],
	fallback: SortPreference<T>,
) {
	let preference = $state({ ...fallback });
	let loaded = $state(false);

	onMount(() => {
		const saved = persistentStorage.getJson<Partial<SortPreference<T>>>(key());

		if (saved && validSorts.includes(saved.sort as T) && typeof saved.reversed === 'boolean') {
			preference.sort = saved.sort as T;
			preference.reversed = saved.reversed;
		}

		loaded = true;
	});

	$effect(() => {
		if (!loaded) return;

		persistentStorage.setJson(key(), preference);
	});

	return preference;
}
