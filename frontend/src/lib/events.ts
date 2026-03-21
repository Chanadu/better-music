type WindowWithFlags = Window & Record<string, boolean | undefined>;

export const bindOnce = (flag: string, setup: () => void) => {
	const windowWithFlags = window as WindowWithFlags;
	if (windowWithFlags[flag]) return;
	windowWithFlags[flag] = true;
	setup();
};
