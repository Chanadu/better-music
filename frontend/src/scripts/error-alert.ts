export const formatErrorMessage = (error: unknown, fallback: string) => {
	const message = error instanceof Error ? error.message : fallback;
	const normalizedMessage = message.trim().replace(/\s+/g, ' ');
	if (!normalizedMessage) return fallback;

	return normalizedMessage.charAt(0).toUpperCase() + normalizedMessage.slice(1);
};

export const createErrorAlert = () => {
	const alert = document.querySelector<HTMLElement>('[data-error-alert]');
	const message = document.querySelector<HTMLElement>('[data-error-alert-message]');

	const show = (text: string) => {
		if (message) {
			message.textContent = text;
		}

		alert?.classList.remove('hidden');
	};

	const hide = () => {
		alert?.classList.add('hidden');
	};

	const showError = (error: unknown, fallback: string) => {
		show(formatErrorMessage(error, fallback));
	};

	return {
		hide,
		show,
		showError,
	};
};
