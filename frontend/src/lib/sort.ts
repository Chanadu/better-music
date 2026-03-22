import { createThemedDropdown } from './ui';

export type SortDirection = 'asc' | 'desc';

type SortOption<TValue extends string> = {
	value: TValue;
	label: string;
};

type SortControlsConfig<TValue extends string> = {
	controlPrefix: string;
	options: Array<SortOption<TValue>>;
	getSortBy: () => TValue;
	setSortBy: (value: TValue) => void;
	getSortDirection: () => SortDirection;
	setSortDirection: (value: SortDirection) => void;
	onChange: () => void;
};

export const syncSortDirectionButton = (
	button: HTMLButtonElement | null,
	direction: SortDirection,
) => {
	if (!button) return;

	if (direction === 'asc') {
		button.title = 'Ascending';
		button.setAttribute('aria-label', 'Sort direction: ascending');
		button.innerHTML =
			'<svg viewBox="0 0 24 24" class="h-4 w-4" aria-hidden="true"><path d="M12 7l-4 6h8z" fill="currentColor"></path></svg>';
		return;
	}

	button.title = 'Descending';
	button.setAttribute('aria-label', 'Sort direction: descending');
	button.innerHTML =
		'<svg viewBox="0 0 24 24" class="h-4 w-4" aria-hidden="true"><path d="M12 17l4-6H8z" fill="currentColor"></path></svg>';
};

export const initializeSortControls = <TValue extends string>({
	controlPrefix,
	options,
	getSortBy,
	setSortBy,
	getSortDirection,
	setSortDirection,
	onChange,
}: SortControlsConfig<TValue>) => {
	const directionButton = document.getElementById(`${controlPrefix}-direction`) as HTMLButtonElement | null;
	if (!directionButton) return;

	const fallbackValue = options[0]?.value;
	if (fallbackValue && !options.some((option) => option.value === getSortBy())) {
		setSortBy(fallbackValue);
	}

	const nativeSelect = document.getElementById(
		`${controlPrefix}-by-native`,
	) as HTMLSelectElement | null;
	const nativeValueNode = document.getElementById(
		`${controlPrefix}-by-native-value`,
	) as HTMLSpanElement | null;
	const syncNativeValueNode = () => {
		if (!nativeSelect || !nativeValueNode) return;
		const selectedOption = nativeSelect.selectedOptions[0];
		nativeValueNode.textContent = selectedOption?.textContent ?? options[0]?.label ?? 'Select';
	};
	if (nativeSelect) {
		nativeSelect.innerHTML = '';
		for (const option of options) {
			const nativeOption = document.createElement('option');
			nativeOption.value = option.value;
			nativeOption.textContent = option.label;
			nativeSelect.appendChild(nativeOption);
		}
		nativeSelect.value = getSortBy();
		syncNativeValueNode();
		if (nativeSelect.dataset.bound !== 'true') {
			nativeSelect.dataset.bound = 'true';
			nativeSelect.addEventListener('change', () => {
				setSortBy(nativeSelect.value as TValue);
				syncNativeValueNode();
				onChange();
			});
		}
	}

	const sortDropdown = createThemedDropdown({
		triggerId: `${controlPrefix}-by-trigger`,
		valueId: `${controlPrefix}-by-value`,
		menuId: `${controlPrefix}-by-menu`,
		onSelect: (value) => {
			setSortBy(value as TValue);
			onChange();
		},
	});
	sortDropdown?.setOptions(options, getSortBy());
	syncNativeValueNode();
	syncSortDirectionButton(directionButton, getSortDirection());

	if (directionButton.dataset.bound !== 'true') {
		directionButton.dataset.bound = 'true';
		directionButton.addEventListener('click', () => {
			setSortDirection(getSortDirection() === 'asc' ? 'desc' : 'asc');
			syncSortDirectionButton(directionButton, getSortDirection());
			onChange();
		});
	}
};
