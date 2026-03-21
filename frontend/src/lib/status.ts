export type StatusTone = 'neutral' | 'error' | 'success';

type StatusOptions = {
	boxed?: boolean;
};

export const setInlineStatus = (
	element: HTMLElement | null,
	message: string,
	type: StatusTone = 'neutral',
	options: StatusOptions = {},
) => {
	if (!element) return;

	element.textContent = message;
	element.dataset.statusType = type;

	if (!options.boxed) {
		element.style.color =
			type === 'error' ? 'var(--error)'
			: type === 'success' ? 'var(--success)'
			: 'var(--text-muted)';
		return;
	}

	if (!message) {
		element.classList.add('hidden');
		element.style.background = '';
		element.style.borderColor = '';
		element.style.color = '';
		return;
	}

	element.classList.remove('hidden');
	element.style.background = 'transparent';
	element.style.borderColor = 'transparent';

	if (type === 'error') {
		element.style.background = 'color-mix(in srgb, var(--error) 14%, var(--surface-raised))';
		element.style.borderColor = 'color-mix(in srgb, var(--error) 45%, var(--border))';
		element.style.color = 'var(--error)';
		return;
	}

	if (type === 'success') {
		element.style.background = 'color-mix(in srgb, var(--success) 14%, var(--surface-raised))';
		element.style.borderColor = 'color-mix(in srgb, var(--success) 45%, var(--border))';
		element.style.color = 'var(--success)';
		return;
	}

	element.style.background = 'color-mix(in srgb, var(--surface-raised) 70%, transparent)';
	element.style.borderColor = 'var(--border)';
	element.style.color = 'var(--text-muted)';
};
