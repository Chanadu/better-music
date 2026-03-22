export type DropdownMenuMode = 'custom' | 'system';

export const DROPDOWN_MENU_MODE_STORAGE_KEY = 'better-music:dropdown-menu-mode';
export const DROPDOWN_MENU_MODE_EVENT = 'better-music-dropdown-mode-change';
export const DEFAULT_DROPDOWN_MENU_MODE: DropdownMenuMode = 'custom';

const normalizeDropdownMenuMode = (value: string | null | undefined): DropdownMenuMode =>
	value === 'system' ? 'system' : DEFAULT_DROPDOWN_MENU_MODE;

export const getDropdownMenuMode = (): DropdownMenuMode => {
	if (typeof window === 'undefined') return DEFAULT_DROPDOWN_MENU_MODE;
	try {
		return normalizeDropdownMenuMode(localStorage.getItem(DROPDOWN_MENU_MODE_STORAGE_KEY));
	} catch {
		return DEFAULT_DROPDOWN_MENU_MODE;
	}
};

export const applyDropdownMenuMode = (
	mode: DropdownMenuMode = getDropdownMenuMode(),
): DropdownMenuMode => {
	if (typeof document !== 'undefined') {
		document.documentElement.dataset.dropdownMode = mode;
	}
	return mode;
};

export const setDropdownMenuMode = (mode: DropdownMenuMode): DropdownMenuMode => {
	const nextMode = normalizeDropdownMenuMode(mode);

	if (typeof window !== 'undefined') {
		try {
			localStorage.setItem(DROPDOWN_MENU_MODE_STORAGE_KEY, nextMode);
		} catch {
			// Ignore storage failures and keep the in-memory mode usable.
		}
	}

	applyDropdownMenuMode(nextMode);

	if (typeof window !== 'undefined') {
		window.dispatchEvent(
			new CustomEvent(DROPDOWN_MENU_MODE_EVENT, {
				detail: { mode: nextMode },
			}),
		);
	}

	return nextMode;
};

export const toggleDropdownMenuMode = (): DropdownMenuMode =>
	setDropdownMenuMode(getDropdownMenuMode() === 'custom' ? 'system' : 'custom');
