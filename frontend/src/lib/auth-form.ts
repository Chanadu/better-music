type AuthFormState = {
	form: HTMLFormElement;
	errorMessage: HTMLElement;
	submitButton: HTMLButtonElement;
	resetErrorState: () => void;
	setSubmittingState: (label: string) => void;
	setFailureState: (message: string, label?: string) => void;
};

export const setupAuthForm = (
	form: HTMLFormElement | null,
	errorMessage: HTMLElement | null,
	defaultLabel: string,
	inputNames: string[],
): AuthFormState | null => {
	if (!form || !errorMessage) return null;

	const submitButton = form.querySelector('button[type="submit"]') as HTMLButtonElement | null;
	if (!submitButton) return null;

	submitButton.setAttribute('data-original-text', submitButton.textContent || defaultLabel);

	const resetErrorState = () => {
		errorMessage.classList.add('hidden');
		submitButton.disabled = false;
		submitButton.textContent = submitButton.getAttribute('data-original-text') || defaultLabel;
	};

	for (const inputName of inputNames) {
		const input = form.querySelector(`input[name="${inputName}"]`) as HTMLInputElement | null;
		input?.addEventListener('input', resetErrorState);
	}

	return {
		form,
		errorMessage,
		submitButton,
		resetErrorState,
		setSubmittingState: (label) => {
			submitButton.disabled = true;
			submitButton.textContent = label;
		},
		setFailureState: (message, label = 'Try again') => {
			errorMessage.textContent = message;
			errorMessage.classList.remove('hidden');
			submitButton.disabled = true;
			submitButton.textContent = label;
		},
	};
};
