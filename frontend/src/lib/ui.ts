export const ICON_PATHS = {
	listened: 'M5 12l5 5L20 7',
	newSparkle: 'M12 3v4M12 17v4M3 12h4M17 12h4M5.6 5.6l2.8 2.8M15.6 15.6l2.8 2.8M18.4 5.6l-2.8 2.8M8.4 15.6l-2.8 2.8',
	ratingStar: 'M12 17.3l-5.2 2.7 1-5.8-4.2-4.1 5.8-.8L12 4l2.6 5.3 5.8.8-4.2 4.1 1 5.8z',
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

export const createChevronIcon = () => {
	const icon = createStrokeIcon('M6 9l6 6 6-6', 'h-4 w-4 transition-transform duration-200');
	icon.style.color = 'var(--text-muted)';
	return icon;
};

export const createEditIconButton = (label: string) => {
	const button = document.createElement('button');
	button.type = 'button';
	button.className = 'flex h-8 w-8 cursor-pointer items-center justify-center rounded-md border p-0';
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
	onSwipeRight: () => void
) => {
	let startX = 0;
	let startY = 0;
	let currentX = 0;
	let isDragging = false;
	let isScrolling: boolean | null = null;
	const threshold = 75; 

	container.addEventListener('touchstart', (e) => {
		if (e.touches.length > 1) return;
		startX = e.touches[0].clientX;
		startY = e.touches[0].clientY;
		isDragging = true;
		isScrolling = null;
		foreground.style.transition = 'none';
	}, { passive: true });

	container.addEventListener('touchmove', (e) => {
		if (!isDragging) return;
		const x = e.touches[0].clientX;
		const y = e.touches[0].clientY;
		const deltaX = x - startX;
		const deltaY = y - startY;

		if (isScrolling === null) {
			isScrolling = Math.abs(deltaY) > Math.abs(deltaX);
		}

		if (isScrolling) {
			isDragging = false;
			return;
		}

		currentX = deltaX;
		if (currentX > 0) currentX = Math.min(currentX, 100);
		else currentX = Math.max(currentX, -100);

		foreground.style.transform = `translateX(${currentX}px)`;
		if (e.cancelable) e.preventDefault();
	}, { passive: false });

	container.addEventListener('touchend', () => {
		if (!isDragging) return;
		isDragging = false;
		
		foreground.style.transition = 'transform 0.3s cubic-bezier(0.32, 0.72, 0, 1)';
		
		if (currentX > threshold) {
			onSwipeRight();
		} else if (currentX < -threshold) {
			onSwipeLeft();
		}
		
		foreground.style.transform = 'translateX(0)';
		currentX = 0;
	});
};
