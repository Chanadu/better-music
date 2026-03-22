export const openDialogSafely = (dialog: HTMLDialogElement | null) => {
	if (!dialog || dialog.open || !dialog.isConnected) return;
	try {
		dialog.showModal();
	} catch {
		// Ignore transient disconnection while Astro swaps pages.
	}
};

export const closeDialogSafely = (dialog: HTMLDialogElement | null) => {
	if (!dialog?.open) return;
	dialog.close();
};

type TabConfig = {
	manualPanel: HTMLElement | null;
	spotifyPanel: HTMLElement | null;
	manualButton: HTMLElement | null;
	spotifyButton: HTMLElement | null;
	searchInput?: HTMLElement | null;
};

export const toggleDialogTabs = (activeTab: 'manual' | 'spotify', config: TabConfig) => {
	const { manualPanel, spotifyPanel, manualButton, spotifyButton, searchInput } = config;
	const isManual = activeTab === 'manual';
	const activeClass = ['bg-surface', 'text-text', 'shadow-sm'];
	const inactiveClass = ['text-text-muted', 'hover:text-text'];

	manualPanel?.classList.toggle('hidden', !isManual);
	spotifyPanel?.classList.toggle('hidden', isManual);

	if (isManual) {
		manualButton?.classList.add(...activeClass);
		manualButton?.classList.remove(...inactiveClass);
		spotifyButton?.classList.remove(...activeClass);
		spotifyButton?.classList.add(...inactiveClass);
		return;
	}

	spotifyButton?.classList.add(...activeClass);
	spotifyButton?.classList.remove(...inactiveClass);
	manualButton?.classList.remove(...activeClass);
	manualButton?.classList.add(...inactiveClass);
	searchInput?.focus();
};

export const toggleCreateTabs = toggleDialogTabs;
