import { navigate } from 'astro:transitions/client';

import { warmLibraryData } from './library-cache';

const APP_ROUTES = ['/albums', '/listened', '/artists'] as const;
const AXIS_LOCK_PX = 10;
const COMPLETE_DISTANCE_PX = 72;
const COMPLETE_DISTANCE_RATIO = 0.22;
const VELOCITY_THRESHOLD = 0.42;
const ENTRY_OFFSET_PX = 28;
const ENTRY_STATE_KEY = '__appPageSwipeEntry';
const EXIT_TRANSITION = 'transform 160ms cubic-bezier(0.22, 1, 0.36, 1), opacity 160ms ease';
const RESET_TRANSITION = 'transform 220ms cubic-bezier(0.22, 1, 0.36, 1), opacity 220ms ease';
const ENTRY_TRANSITION = 'transform 220ms cubic-bezier(0.16, 1, 0.3, 1), opacity 220ms ease';

type AppRoute = (typeof APP_ROUTES)[number];
type SwipeDirection = 'next' | 'prev';

type SwipeState = {
	shell: HTMLElement;
	route: AppRoute;
	direction: SwipeDirection | null;
	startX: number;
	startY: number;
	currentX: number;
	startTime: number;
	isScrolling: boolean | null;
};

type WindowWithPreloadState = Window & {
	__preloadedRoutes?: Record<string, boolean>;
};

let swipeState: SwipeState | null = null;
let gestureBindingsReady = false;
let navigationPending = false;

const getActiveShell = () => document.querySelector('[data-app-shell="true"]') as HTMLElement | null;

const isMobileViewport = () => window.matchMedia('(max-width: 767px)').matches;

const prefersReducedMotion = () => window.matchMedia('(prefers-reduced-motion: reduce)').matches;

const getRoute = (value?: string | null): AppRoute | null => {
	if (!value) return null;
	return APP_ROUTES.includes(value as AppRoute) ? (value as AppRoute) : null;
};

const getAdjacentRoute = (route: AppRoute, direction: SwipeDirection): AppRoute | null => {
	const index = APP_ROUTES.indexOf(route);
	if (index === -1) return null;
	if (direction === 'prev') return APP_ROUTES[index - 1] ?? null;
	return APP_ROUTES[index + 1] ?? null;
};

const isGestureTargetAllowed = (target: EventTarget | null) => {
	if (!(target instanceof Element)) return false;
	return !target.closest(
		'dialog[open], [data-no-page-swipe], .interactive-list-card, .album-drag-handle, button, a, input, textarea, select, label, summary, [role="button"], [contenteditable="true"], [data-themed-dropdown-root]',
	);
};

const setShellOffset = (shell: HTMLElement, offsetX: number) => {
	const progress = Math.min(1, Math.abs(offsetX) / Math.max(window.innerWidth * 0.45, 1));
	shell.style.transform = `translate3d(${offsetX}px, 0, 0)`;
	shell.style.opacity = String(Math.max(0.88, 1 - progress * 0.12));
};

const clearShellOffset = (shell: HTMLElement) => {
	shell.style.transition = '';
	shell.style.transform = '';
	shell.style.opacity = '';
	shell.style.willChange = '';
};

const prefetchRoute = (route: AppRoute | null) => {
	if (!route) return;
	const preloadState = ((window as WindowWithPreloadState).__preloadedRoutes ??= {});
	if (preloadState[route]) return;
	preloadState[route] = true;

	void fetch(route, {
		method: 'GET',
		credentials: 'same-origin',
		headers: { 'x-preload': '1' },
	}).catch(() => {
		preloadState[route] = false;
	});
};

const resetSwipe = (state: SwipeState) => {
	state.shell.style.transition = RESET_TRANSITION;
	setShellOffset(state.shell, 0);
	window.setTimeout(() => clearShellOffset(state.shell), 220);
};

const storeEntryAnimation = (direction: SwipeDirection, route: AppRoute) => {
	sessionStorage.setItem(
		ENTRY_STATE_KEY,
		JSON.stringify({
			direction,
			route,
			time: Date.now(),
		}),
	);
};

const applyEntryAnimation = () => {
	const shell = getActiveShell();
	if (!shell || !isMobileViewport() || prefersReducedMotion()) return;

	const rawState = sessionStorage.getItem(ENTRY_STATE_KEY);
	if (!rawState) return;
	sessionStorage.removeItem(ENTRY_STATE_KEY);

	try {
		const { direction, route, time } = JSON.parse(rawState) as {
			direction?: SwipeDirection;
			route?: string;
			time?: number;
		};
		const currentRoute = getRoute(shell.dataset.currentRoute);
		if (!direction || route !== currentRoute || !time || Date.now() - time > 1200) return;

		const entryOffset = direction === 'next' ? ENTRY_OFFSET_PX : -ENTRY_OFFSET_PX;
		shell.style.transition = 'none';
		setShellOffset(shell, entryOffset);

		requestAnimationFrame(() => {
			requestAnimationFrame(() => {
				shell.style.transition = ENTRY_TRANSITION;
				setShellOffset(shell, 0);
				window.setTimeout(() => clearShellOffset(shell), 240);
			});
		});
	} catch {
		sessionStorage.removeItem(ENTRY_STATE_KEY);
	}
};

const finishSwipeNavigation = (state: SwipeState, destination: AppRoute) => {
	storeEntryAnimation(state.direction, destination);
	if (prefersReducedMotion()) {
		navigationPending = true;
		void navigate(destination);
		return;
	}

	navigationPending = true;
	state.shell.style.transition = EXIT_TRANSITION;
	setShellOffset(state.shell, state.direction === 'next' ? -window.innerWidth : window.innerWidth);

	window.setTimeout(() => {
		void navigate(destination);
	}, 115);
};

const cancelActiveSwipe = () => {
	if (!swipeState) return;
	clearShellOffset(swipeState.shell);
	swipeState = null;
};

const handleTouchStart = (event: TouchEvent) => {
	if (gestureBindingsReady === false || navigationPending || !isMobileViewport()) return;
	if (event.touches.length !== 1 || document.querySelector('dialog[open]')) return;

	const shell = getActiveShell();
	const route = getRoute(shell?.dataset.currentRoute);
	if (!shell || !route || !isGestureTargetAllowed(event.target)) return;

	const touch = event.touches[0];
	prefetchRoute(getAdjacentRoute(route, 'prev'));
	prefetchRoute(getAdjacentRoute(route, 'next'));
	warmLibraryData();

	shell.style.transition = 'none';
	shell.style.willChange = 'transform, opacity';

	swipeState = {
		shell,
		route,
		direction: null,
		startX: touch.clientX,
		startY: touch.clientY,
		currentX: 0,
		startTime: Date.now(),
		isScrolling: null,
	};
};

const handleTouchMove = (event: TouchEvent) => {
	if (!swipeState || event.touches.length !== 1) return;

	const touch = event.touches[0];
	const rawDeltaX = touch.clientX - swipeState.startX;
	const rawDeltaY = touch.clientY - swipeState.startY;

	if (swipeState.isScrolling === null) {
		if (Math.abs(rawDeltaX) < AXIS_LOCK_PX && Math.abs(rawDeltaY) < AXIS_LOCK_PX) return;
		swipeState.isScrolling = Math.abs(rawDeltaY) > Math.abs(rawDeltaX);
	}

	if (swipeState.isScrolling) {
		cancelActiveSwipe();
		return;
	}

	const candidateDirection =
		rawDeltaX < 0 ? 'next'
		: rawDeltaX > 0 ? 'prev'
		: null;

	if (!candidateDirection) return;

	const destination = getAdjacentRoute(swipeState.route, candidateDirection);
	if (!destination) {
		swipeState.direction = null;
		swipeState.currentX = 0;
		setShellOffset(swipeState.shell, 0);
		return;
	}

	swipeState.direction = candidateDirection;
	swipeState.currentX = rawDeltaX;

	if (event.cancelable) event.preventDefault();
	setShellOffset(swipeState.shell, rawDeltaX);
};

const handleTouchEnd = () => {
	if (!swipeState) return;

	const state = swipeState;
	swipeState = null;

	if (!state.direction) {
		clearShellOffset(state.shell);
		return;
	}

	const destination = getAdjacentRoute(state.route, state.direction);
	if (!destination) {
		resetSwipe(state);
		return;
	}

	const elapsed = Math.max(1, Date.now() - state.startTime);
	const velocity = Math.abs(state.currentX / elapsed);
	const completionDistance = Math.max(COMPLETE_DISTANCE_PX, window.innerWidth * COMPLETE_DISTANCE_RATIO);
	const reachedThreshold = Math.abs(state.currentX) >= completionDistance || velocity >= VELOCITY_THRESHOLD;

	if (reachedThreshold) {
		finishSwipeNavigation(state, destination);
		return;
	}

	resetSwipe(state);
};

const handleNavigationSettled = () => {
	navigationPending = false;
	applyEntryAnimation();
	warmLibraryData();
};

export const refreshAppShellGesture = () => {
	applyEntryAnimation();
	warmLibraryData();

	if (gestureBindingsReady) return;
	gestureBindingsReady = true;

	document.addEventListener('touchstart', handleTouchStart, { capture: true, passive: true });
	document.addEventListener('touchmove', handleTouchMove, { capture: true, passive: false });
	document.addEventListener('touchend', handleTouchEnd, { capture: true, passive: true });
	document.addEventListener('touchcancel', cancelActiveSwipe, { capture: true, passive: true });
	document.addEventListener('astro:after-swap', handleNavigationSettled);
	document.addEventListener('astro:page-load', handleNavigationSettled);
};
