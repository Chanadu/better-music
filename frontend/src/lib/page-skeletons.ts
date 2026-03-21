export const APP_PAGE_ROUTES = ['/albums', '/listened', '/artists'] as const;

export type AppPageRoute = (typeof APP_PAGE_ROUTES)[number];
export type AppPageSwipeDirection = 'next' | 'prev';

const block = (className: string) => `<span class="page-skeleton-block ${className}"></span>`;

const renderSortToolbar = () => `
	<div class="page-skeleton-toolbar">
		${block('page-skeleton-toolbar-label')}
		<div class="page-skeleton-toolbar-controls">
			${block('page-skeleton-toolbar-pill')}
			${block('page-skeleton-toolbar-icon')}
		</div>
	</div>
`;

const renderAlbumCard = (variant: 'queue' | 'done', index: number) => `
	<div class="page-skeleton-card page-skeleton-card-${variant}" style="animation-delay: ${index * 55}ms">
		${block('page-skeleton-card-cover')}
		<div class="page-skeleton-card-lines">
			${block(`page-skeleton-line-title ${index % 2 === 0 ? 'page-skeleton-line-wide' : 'page-skeleton-line-medium'}`)}
			${block(`page-skeleton-line-meta ${index % 3 === 0 ? 'page-skeleton-line-medium' : 'page-skeleton-line-short'}`)}
		</div>
		${variant === 'done' ? block('page-skeleton-rating-badge') : block('page-skeleton-queue-chip')}
	</div>
`;

const renderArtistCard = (index: number) => `
	<div class="page-skeleton-card page-skeleton-card-artist" style="animation-delay: ${index * 55}ms">
		${block('page-skeleton-card-cover page-skeleton-card-cover-artist')}
		<div class="page-skeleton-card-lines">
			${block(`page-skeleton-line-title ${index % 2 === 0 ? 'page-skeleton-line-medium' : 'page-skeleton-line-short'}`)}
			${block('page-skeleton-line-meta page-skeleton-line-short')}
			<div class="page-skeleton-pill-row">
				${block('page-skeleton-pill')}
				${block(`page-skeleton-pill ${index % 2 === 0 ? 'page-skeleton-pill-short' : ''}`)}
			</div>
		</div>
	</div>
`;

const renderAlbumsContent = (variant: 'queue' | 'done') => `
	${renderSortToolbar()}
	<div class="page-skeleton-stack">
		${Array.from({ length: 4 }, (_, index) => renderAlbumCard(variant, index)).join('')}
	</div>
`;

const renderArtistsContent = () => `
	${renderSortToolbar()}
	<div class="page-skeleton-stack">
		${Array.from({ length: 4 }, (_, index) => renderArtistCard(index)).join('')}
	</div>
`;

const renderPageContent = (route: AppPageRoute) => {
	if (route === '/artists') return renderArtistsContent();
	if (route === '/listened') return renderAlbumsContent('done');
	return renderAlbumsContent('queue');
};

const renderPreviewHeader = (route: AppPageRoute) => {
	const titleClass =
		route === '/artists' ? 'page-skeleton-title-artists'
		: route === '/listened' ? 'page-skeleton-title-done'
		: 'page-skeleton-title-queue';

	return `
		<div class="page-skeleton-preview-header">
			<div class="page-skeleton-preview-header-inner">
				<div class="page-skeleton-preview-brand">
					${block('page-skeleton-brand-icon')}
					<div class="page-skeleton-preview-brand-copy">
						${block('page-skeleton-kicker')}
						${block(`page-skeleton-preview-title ${titleClass}`)}
					</div>
				</div>
				<div class="page-skeleton-preview-actions">
					${block('page-skeleton-header-action')}
					${block('page-skeleton-header-action page-skeleton-header-action-wide')}
				</div>
			</div>
		</div>
	`;
};

export const renderPageSwipePreview = (route: AppPageRoute) => `
	<div class="page-skeleton-preview-shell">
		${renderPreviewHeader(route)}
		<div class="page-skeleton-preview-main">
			<div class="page-skeleton-content-shell">
				${renderPageContent(route)}
			</div>
		</div>
	</div>
`;

export const renderPageLoadingSkeleton = (route: AppPageRoute) => `
	<div class="page-skeleton-loading-shell">
		<div class="page-skeleton-content-shell">
			${renderPageContent(route)}
		</div>
	</div>
`;

const getCurrentRoute = (): AppPageRoute | null => {
	const route = document.querySelector('[data-app-shell="true"]')?.getAttribute('data-current-route');
	if (!route) return null;
	return APP_PAGE_ROUTES.includes(route as AppPageRoute) ? (route as AppPageRoute) : null;
};

const getPreviewRoot = () => document.querySelector('[data-page-preview-root="true"]') as HTMLDivElement | null;

const getLoadingRoot = () => document.querySelector('[data-page-loading-overlay="true"]') as HTMLDivElement | null;

const setRouteMarkup = (
	root: HTMLDivElement | null,
	route: AppPageRoute,
	render: (targetRoute: AppPageRoute) => string,
) => {
	if (!root) return;
	if (root.dataset.route === route) return;
	root.dataset.route = route;
	root.innerHTML = render(route);
};

export const syncPageSkeletonRoute = () => {
	const previewRoot = getPreviewRoot();
	if (previewRoot) {
		previewRoot.classList.remove('is-visible');
		previewRoot.setAttribute('aria-hidden', 'true');
		previewRoot.dataset.route = '';
		previewRoot.innerHTML = '';
	}

	const currentRoute = getCurrentRoute();
	const loadingRoot = getLoadingRoot();
	if (!loadingRoot || !currentRoute) return;
	setRouteMarkup(loadingRoot, currentRoute, renderPageLoadingSkeleton);
	loadingRoot.classList.remove('is-visible');
	loadingRoot.setAttribute('aria-hidden', 'true');
};

export const setSwipePreview = (
	route: AppPageRoute,
	direction: AppPageSwipeDirection,
	progress: number,
) => {
	const previewRoot = getPreviewRoot();
	if (!previewRoot) return;

	setRouteMarkup(previewRoot, route, renderPageSwipePreview);
	previewRoot.classList.add('is-visible');
	previewRoot.setAttribute('aria-hidden', 'false');

	const previewShell = previewRoot.firstElementChild as HTMLElement | null;
	if (!previewShell) return;

	const clampedProgress = Math.min(1, Math.max(0, progress));
	const parallax = (1 - clampedProgress) * 26;
	previewRoot.style.opacity = String(0.5 + clampedProgress * 0.5);
	previewShell.style.transform = `translate3d(${direction === 'next' ? parallax : -parallax}px, 0, 0)`;
	previewShell.style.opacity = String(0.72 + clampedProgress * 0.28);
};

export const clearSwipePreview = () => {
	const previewRoot = getPreviewRoot();
	if (!previewRoot) return;

	previewRoot.classList.remove('is-visible');
	previewRoot.setAttribute('aria-hidden', 'true');
	previewRoot.style.opacity = '';

	const previewShell = previewRoot.firstElementChild as HTMLElement | null;
	if (previewShell) {
		previewShell.style.transform = '';
		previewShell.style.opacity = '';
	}
};

export const showPageLoadingSkeleton = (route: AppPageRoute) => {
	const loadingRoot = getLoadingRoot();
	if (!loadingRoot || getCurrentRoute() !== route) return;

	setRouteMarkup(loadingRoot, route, renderPageLoadingSkeleton);
	loadingRoot.classList.add('is-visible');
	loadingRoot.setAttribute('aria-hidden', 'false');
};

export const hidePageLoadingSkeleton = (route: AppPageRoute) => {
	const loadingRoot = getLoadingRoot();
	if (!loadingRoot || getCurrentRoute() !== route) return;

	loadingRoot.classList.remove('is-visible');
	loadingRoot.setAttribute('aria-hidden', 'true');
};
