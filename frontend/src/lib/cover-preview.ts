export const syncCoverPreview = (
	input: HTMLInputElement | null,
	image: HTMLImageElement | null,
	emptyState: HTMLElement | null,
	url: string | null | undefined,
) => {
	const nextValue = url?.trim() ?? '';

	if (input) input.value = nextValue;
	if (!image || !emptyState) return;

	if (!nextValue) {
		image.src = '';
		image.classList.add('hidden');
		emptyState.classList.remove('hidden');
		return;
	}

	image.src = nextValue;
	image.classList.remove('hidden');
	emptyState.classList.add('hidden');
};
