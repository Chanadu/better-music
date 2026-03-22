export const ICON_PATHS = {
	listened: 'M5 12l5 5L20 7',
	newSparkle: 'M12 3v4M12 17v4M3 12h4M17 12h4M5.6 5.6l2.8 2.8M15.6 15.6l2.8 2.8M18.4 5.6l-2.8 2.8M8.4 15.6l-2.8 2.8',
	ratingStar: 'M12 17.3l-5.2 2.7 1-5.8-4.2-4.1 5.8-.8L12 4l2.6 5.3 5.8.8-4.2 4.1 1 5.8z',
	syncPending: 'M3 12a9 9 0 0 1 15.4-6.4M18 3v4h4M21 12a9 9 0 0 1-15.4 6.4M6 21v-4H2',
	edit: 'M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z',
	trash: 'M3 6h18M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2',
} as const;

type Tone = 'neutral' | 'secondary' | 'primary';

const createStrokeIcon = (pathD: string, className: string): SVGSVGElement => {
	const icon = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
	icon.setAttribute('viewBox', '0 0 24 24');
	icon.setAttribute('aria-hidden', 'true');
	icon.setAttribute('class', className);

	const path = document.createElementNS('http://www.w3.org/2000/svg', 'path');
	path.setAttribute('d', pathD);
	path.setAttribute('fill', 'none');
	path.setAttribute('stroke', 'currentColor');
	path.setAttribute('stroke-width', '2');
	path.setAttribute('stroke-linecap', 'round');
	path.setAttribute('stroke-linejoin', 'round');
	icon.appendChild(path);

	return icon;
};

const applyTone = (el: HTMLElement, tone: Tone, surfaceVar: '--surface' | '--surface-raised') => {
	if (tone === 'secondary') {
		el.style.borderColor = 'color-mix(in srgb, var(--secondary) 40%, var(--border))';
		el.style.background = `color-mix(in srgb, var(--secondary) 18%, var(${surfaceVar}))`;
		el.style.color = 'var(--secondary)';
		return;
	}
	if (tone === 'primary') {
		el.style.borderColor = 'color-mix(in srgb, var(--primary) 45%, var(--border))';
		el.style.background = `color-mix(in srgb, var(--primary) 16%, var(${surfaceVar}))`;
		el.style.color = 'var(--primary)';
		return;
	}
	el.style.borderColor = 'var(--border)';
	el.style.background = `var(${surfaceVar})`;
	el.style.color = 'var(--text-muted)';
};

const createBadge = (
	label: string,
	iconPath: string,
	tone: Tone,
	className: string,
	surfaceVar: '--surface' | '--surface-raised',
) => {
	const badge = document.createElement('span');
	badge.className = className;
	applyTone(badge, tone, surfaceVar);
	badge.appendChild(createStrokeIcon(iconPath, 'h-3.5 w-3.5 shrink-0'));
	const text = document.createElement('span');
	text.textContent = label;
	badge.appendChild(text);
	return badge;
};

export const createAlbumPill = (label: string, iconPath: string, tone: Tone = 'neutral') => {
	return createBadge(
		label,
		iconPath,
		tone,
		'inline-flex items-center gap-1.5 rounded-full border px-2.5 py-1 text-xs font-medium',
		'--surface-raised',
	);
};

export const createArtistChip = (label: string, iconPath: string, tone: Tone = 'neutral') => {
	return createBadge(
		label,
		iconPath,
		tone,
		'inline-flex items-center gap-1.5 rounded-full border px-2 py-0.5 text-xs font-medium',
		'--surface',
	);
};

export const applyRatingTone = (
	el: HTMLElement,
	rating: number,
	surfaceVar: '--surface' | '--surface-raised' = '--surface-raised',
) => {
	const clamped = Math.min(10, Math.max(0, rating));
	const ratingColor =
		clamped >= 5 ?
			`color-mix(in oklab, var(--rating-high) ${Math.round(((clamped - 5) / 5) * 100)}%, var(--rating-mid))`
		:	`color-mix(in oklab, var(--rating-mid) ${Math.round((clamped / 5) * 100)}%, var(--rating-low))`;
	el.style.borderColor = `color-mix(in srgb, ${ratingColor} 45%, var(--border))`;
	el.style.background = `color-mix(in srgb, ${ratingColor} 18%, var(${surfaceVar}))`;
	el.style.color = ratingColor;
};

export const createChevronIcon = () => {
	const icon = createStrokeIcon('M6 9l6 6 6-6', 'h-4 w-4 transition-transform duration-200');
	icon.style.color = 'var(--text-muted)';
	return icon;
};

type DropdownOption = {
	value: string;
	label: string;
};

type ThemedDropdownConfig = {
	triggerId: string;
	valueId: string;
	menuId: string;
	onSelect: (value: string) => void;
};

type ThemedDropdownController = {
	setOptions: (options: DropdownOption[], selectedValue: string) => void;
	close: () => void;
};

export const createThemedDropdown = (config: ThemedDropdownConfig): ThemedDropdownController | null => {
	const trigger = document.getElementById(config.triggerId) as HTMLButtonElement | null;
	const valueNode = document.getElementById(config.valueId) as HTMLSpanElement | null;
	const menu = document.getElementById(config.menuId) as HTMLDivElement | null;
	if (!trigger || !valueNode || !menu) return null;

	const root = trigger.closest('[data-themed-dropdown-root]') as HTMLDivElement | null;
	if (!root) return null;
	const chevron = trigger.querySelector('[data-dropdown-chevron="true"]') as SVGSVGElement | null;

	const syncOpenState = (isOpen: boolean) => {
		trigger.classList.toggle('border-primary/25', isOpen);
		trigger.classList.toggle('bg-surface-raised/78', isOpen);
		chevron?.classList.toggle('rotate-180', isOpen);
	};

	const close = () => {
		menu.classList.add('hidden');
		trigger.setAttribute('aria-expanded', 'false');
		syncOpenState(false);
	};

	const open = () => {
		menu.classList.remove('hidden');
		trigger.setAttribute('aria-expanded', 'true');
		syncOpenState(true);
	};

	if (trigger.dataset.bound !== 'true') {
		trigger.dataset.bound = 'true';
		trigger.addEventListener('click', () => {
			if (menu.classList.contains('hidden')) {
				open();
				return;
			}
			close();
		});

		document.addEventListener('click', (event) => {
			const target = event.target as Node | null;
			if (!target) return;
			if (root.contains(target)) return;
			close();
		});

		document.addEventListener('keydown', (event) => {
			if (event.key === 'Escape') close();
		});
	}

	const setOptions = (options: DropdownOption[], selectedValue: string) => {
		menu.innerHTML = '';
		let selectedLabel = '';

		for (const option of options) {
			const optionButton = document.createElement('button');
			optionButton.type = 'button';
			optionButton.dataset.value = option.value;
			optionButton.setAttribute('role', 'option');
			optionButton.setAttribute('aria-selected', option.value === selectedValue ? 'true' : 'false');
			optionButton.className =
				'themed-dropdown-option ui-subtle-pressable group flex w-full items-center gap-3 rounded-xl border px-3 py-2.5 text-left text-sm font-medium transition-all duration-150';

			const labelNode = document.createElement('span');
			labelNode.className = 'min-w-0 flex-1 truncate';
			labelNode.textContent = option.label;

			const indicator = document.createElement('span');
			indicator.className =
				'inline-flex h-5 w-5 shrink-0 items-center justify-center rounded-full border transition-all duration-150';

			if (option.value === selectedValue) {
				optionButton.classList.add('border-primary/25', 'bg-primary/12', 'text-primary', 'shadow-sm');
				indicator.classList.add('border-primary/20', 'bg-primary/12', 'text-primary');
				indicator.appendChild(createStrokeIcon('M7 12l3 3 7-7', 'h-3 w-3'));
				selectedLabel = option.label;
			} else {
				optionButton.classList.add('border-transparent', 'text-text', 'hover:border-border/55', 'hover:bg-surface-raised/78');
				indicator.classList.add('border-border/55', 'bg-surface/80', 'text-transparent', 'group-hover:border-primary/15');
			}

			optionButton.appendChild(labelNode);
			optionButton.appendChild(indicator);
			optionButton.addEventListener('click', () => {
				config.onSelect(option.value);
				close();
			});
			menu.appendChild(optionButton);
		}

		if (!selectedLabel && options.length > 0) selectedLabel = options[0].label;
		valueNode.textContent = selectedLabel;
	};

	return { setOptions, close };
};

export const createEditIconButton = (label: string) => {
	const button = document.createElement('button');
	button.type = 'button';
	button.className =
		'ui-subtle-pressable flex h-8 w-8 cursor-pointer items-center justify-center rounded-md border p-0';
	button.style.borderColor = 'var(--border)';
	button.style.color = 'var(--text-muted)';
	button.setAttribute('aria-label', label);

	const icon = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
	icon.setAttribute('viewBox', '0 0 24 24');
	icon.setAttribute('aria-hidden', 'true');
	icon.classList.add('h-4', 'w-4');

	const line = createStrokeIcon('M12 20h9', '');
	while (line.firstChild) icon.appendChild(line.firstChild);

	const body = createStrokeIcon('M16.5 3.5a2.1 2.1 0 0 1 3 3L7 19l-4 1 1-4 12.5-12.5z', '');
	while (body.firstChild) icon.appendChild(body.firstChild);

	button.appendChild(icon);
	return button;
};

export const bindSwipe = (
	container: HTMLElement,
	foreground: HTMLElement,
	onSwipeLeft: () => void,
	onSwipeRight: () => void,
) => {
	let startX = 0;
	let startY = 0;
	let currentX = 0;
	let isDragging = false;
	let isScrolling: boolean | null = null;
	const actionThreshold = Math.min(window.innerWidth * 0.4, 150); // Require swiping at least 40% of screen or 150px
	const velocityThreshold = 0.8;
	const slideOutDuration = 50;
	let lastTouchTime = 0;
	let lastTouchX = 0;

	container.addEventListener(
		'touchstart',
		(e) => {
			if (e.touches.length > 1) return;
			const target = e.target as Element | null;
			if (target?.closest('.album-drag-handle')) return;
			startX = e.touches[0].clientX;
			startY = e.touches[0].clientY;
			lastTouchX = startX;
			lastTouchTime = Date.now();
			isDragging = true;
			isScrolling = null;
			foreground.style.transition = 'none';

			const bg = container.firstElementChild as HTMLElement | null;
			if (bg && bg.children.length >= 2) {
				const leftAction = bg.firstElementChild as HTMLElement;
				const rightAction = bg.lastElementChild as HTMLElement;
				leftAction.style.opacity = '0';
				rightAction.style.opacity = '0';
			}
		},
		{ passive: true },
	);

	container.addEventListener(
		'touchmove',
		(e) => {
			if (!isDragging) return;

			const currentTouchX = e.touches[0].clientX;
			const currentTouchY = e.touches[0].clientY;
			const deltaX = currentTouchX - startX;
			const deltaY = currentTouchY - startY;

			if (isScrolling === null) {
				isScrolling = Math.abs(deltaY) > Math.abs(deltaX);
			}

			if (isScrolling) {
				isDragging = false;
				return;
			}

			if (e.cancelable) e.preventDefault();

			currentX = deltaX;
			lastTouchX = currentTouchX;
			lastTouchTime = Date.now();

			foreground.style.transform = `translateX(${currentX}px)`;

			// Dynamically show only the relevant action
			const bg = container.firstElementChild as HTMLElement;
			if (bg && bg.children.length >= 2) {
				const leftAction = bg.firstElementChild as HTMLElement;
				const rightAction = bg.lastElementChild as HTMLElement;
				if (currentX > 0) {
					// Swiping Right -> Left-side action (Edit, Primary)
					leftAction.style.opacity = '1';
					rightAction.style.opacity = '0';
				} else if (currentX < 0) {
					// Swiping Left -> Right-side action (Delete, Error)
					leftAction.style.opacity = '0';
					rightAction.style.opacity = '1';
				} else {
					leftAction.style.opacity = '0';
					rightAction.style.opacity = '0';
				}
			}
		},
		{ passive: false },
	);

	container.addEventListener('touchend', () => {
		if (!isDragging) return;
		isDragging = false;

		const timeDelta = Date.now() - lastTouchTime;
		const velocityX = timeDelta > 0 ? (currentX - (lastTouchX - startX)) / timeDelta : 0;

		const triggerRight = currentX > actionThreshold || velocityX > velocityThreshold;
		const triggerLeft = currentX < -actionThreshold || velocityX < -velocityThreshold;

		if (triggerRight) {
			// Slide all the way right
			foreground.style.transition = `transform ${slideOutDuration}ms cubic-bezier(0.32, 0.72, 0, 1)`;
			foreground.style.transform = `translateX(${window.innerWidth}px)`;
			setTimeout(() => {
				onSwipeRight();
				// Reset position after action triggers
				foreground.style.transition = 'none';
				foreground.style.transform = 'translateX(0)';
			}, slideOutDuration);
		} else if (triggerLeft) {
			// Slide all the way left
			foreground.style.transition = `transform ${slideOutDuration}ms cubic-bezier(0.32, 0.72, 0, 1)`;
			foreground.style.transform = `translateX(-${window.innerWidth}px)`;
			setTimeout(() => {
				onSwipeLeft();
				// Reset position after action triggers
				foreground.style.transition = 'none';
				foreground.style.transform = 'translateX(0)';
			}, slideOutDuration);
		} else {
			// Snap back
			foreground.style.transition = 'transform 0.3s cubic-bezier(0.32, 0.72, 0, 1)';
			foreground.style.transform = 'translateX(0)';
		}

		// Reset opacities on end
		const bg = container.firstElementChild as HTMLElement;
		if (bg && bg.children.length >= 2) {
			const leftAction = bg.firstElementChild as HTMLElement;
			const rightAction = bg.lastElementChild as HTMLElement;
			setTimeout(() => {
				if (!isDragging) {
					leftAction.style.opacity = '0';
					rightAction.style.opacity = '0';
				}
			}, slideOutDuration);
		}

		currentX = 0;
	});
};
