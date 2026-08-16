import { onMount } from 'svelte';

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
		try {
			const saved = JSON.parse(localStorage.getItem(key()) ?? '') as Partial<SortPreference<T>>;

			if (validSorts.includes(saved.sort as T) && typeof saved.reversed === 'boolean') {
				preference.sort = saved.sort as T;
				preference.reversed = saved.reversed;
			}
		} catch {}

		loaded = true;
	});

	$effect(() => {
		if (!loaded) return;

		try {
			localStorage.setItem(key(), JSON.stringify(preference));
		} catch {}
	});

	return preference;
}
