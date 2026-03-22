export type SpotifySearchElementIDs = {
	inputId: string;
	spinnerId: string;
	resultsId: string;
	warningId: string;
};

export type SpotifySearchElements = {
	input: HTMLInputElement | null;
	spinner: HTMLElement | null;
	results: HTMLUListElement | null;
	warning: HTMLElement | null;
};

type AvailabilityOptions = {
	isOnline?: () => boolean;
	offlinePlaceholder?: string;
};

type BindSpotifySearchOptions<T> = AvailabilityOptions & {
	elements: SpotifySearchElements;
	search: (query: string) => Promise<T[]>;
	renderResult: (item: T) => HTMLLIElement;
	emptyMessage?: string;
	debounceMs?: number;
	getErrorMessage?: (error: unknown) => string;
};

export const isNavigatorOnline = (): boolean => (typeof navigator === 'undefined' ? true : navigator.onLine);

export const getSpotifySearchElements = (ids: SpotifySearchElementIDs): SpotifySearchElements => ({
	input: document.getElementById(ids.inputId) as HTMLInputElement | null,
	spinner: document.getElementById(ids.spinnerId),
	results: document.getElementById(ids.resultsId) as HTMLUListElement | null,
	warning: document.getElementById(ids.warningId),
});

export const hideSpotifySearchResults = (results: HTMLUListElement | null): void => {
	results?.classList.add('hidden');
};

export const resetSpotifySearch = (elements: SpotifySearchElements): void => {
	if (elements.input) elements.input.value = '';
	elements.spinner?.classList.add('hidden');
	if (elements.results) elements.results.replaceChildren();
	hideSpotifySearchResults(elements.results);
};

export const syncSpotifySearchAvailability = (
	elements: SpotifySearchElements,
	options: AvailabilityOptions = {},
): void => {
	const onlineCheck = options.isOnline ?? isNavigatorOnline;
	const offline = !onlineCheck();

	if (elements.input) {
		elements.input.disabled = offline;
		elements.input.placeholder =
			offline ? (options.offlinePlaceholder ?? 'Spotify search requires a connection') : ' ';
		if (offline) elements.input.value = '';
	}

	elements.warning?.classList.toggle('hidden', !offline);

	if (offline) {
		elements.spinner?.classList.add('hidden');
		hideSpotifySearchResults(elements.results);
	}
};

const createMessageItem = (message: string, type: 'neutral' | 'error' = 'neutral'): HTMLLIElement => {
	const item = document.createElement('li');
	item.className =
		type === 'error' ?
			'px-4 py-3 text-center text-sm font-medium text-[var(--error)]'
		:	'px-4 py-3 text-center text-sm italic text-text-muted';
	item.textContent = message;
	return item;
};

export const bindSpotifySearch = <T>(options: BindSpotifySearchOptions<T>): void => {
	const {
		elements,
		search,
		renderResult,
		emptyMessage = 'No results found on Spotify',
		debounceMs = 350,
		getErrorMessage,
		isOnline = isNavigatorOnline,
		offlinePlaceholder,
	} = options;
	const input = elements.input;

	if (!input || input.dataset.bound === 'true') return;

	let searchDebounce: ReturnType<typeof setTimeout>;
	input.dataset.bound = 'true';
	input.addEventListener('input', () => {
		clearTimeout(searchDebounce);
		const query = input.value.trim();

		if (!isOnline()) {
			syncSpotifySearchAvailability(elements, { isOnline, offlinePlaceholder });
			return;
		}

		if (!query) {
			hideSpotifySearchResults(elements.results);
			return;
		}

		searchDebounce = setTimeout(async () => {
			const { results, spinner } = elements;
			if (!results || !spinner) return;

			spinner.classList.remove('hidden');
			try {
				const items = await search(query);
				results.replaceChildren();
				if (!items.length) {
					results.appendChild(createMessageItem(emptyMessage));
				} else {
					for (const item of items) results.appendChild(renderResult(item));
				}
				results.classList.remove('hidden');
			} catch (error) {
				results.replaceChildren(
					createMessageItem(
						getErrorMessage?.(error) ??
							(error instanceof Error ? error.message : 'Search failed. Try again.'),
						'error',
					),
				);
				results.classList.remove('hidden');
			} finally {
				spinner.classList.add('hidden');
			}
		}, debounceMs);
	});
};
