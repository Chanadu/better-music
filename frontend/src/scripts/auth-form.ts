import { authApi, ApiError } from './api';
import { saveTokens, type TokenResponse } from './auth';

const setError = (form: HTMLFormElement, message: string) => {
	const errorElement = form.querySelector<HTMLElement>('[data-auth-error]');
	if (!errorElement) return;

	errorElement.textContent = message;
	errorElement.classList.toggle('hidden', message === '');
};

const updateButton = (form: HTMLFormElement) => {
	const inputs = Array.from(form.querySelectorAll<HTMLInputElement>('input[required]'));
	const button = form.querySelector<HTMLButtonElement>('button[type="submit"]');

	inputs.forEach((input) => {
		const targetName = input.dataset.match;
		if (!targetName) return;

		const target = form.querySelector<HTMLInputElement>(`input[name="${targetName}"]`);
		input.setCustomValidity(target && input.value === target.value ? '' : 'Passwords must match');
	});

	const disabled = !inputs.every((input) => input.checkValidity());
	if (!button) return;

	button.disabled = disabled;
	button.classList.toggle('btn-disabled', disabled);
};

const submitAuthForm = async (form: HTMLFormElement) => {
	const formData = new FormData(form);
	const email = String(formData.get('email') ?? '');
	const password = String(formData.get('password') ?? '');
	const action = form.dataset.authAction;

	const body = { email, password };
	const tokens: TokenResponse = action === 'register' ? await authApi.register(body) : await authApi.login(body);

	saveTokens(tokens);
	window.location.assign(form.dataset.successPath ?? '/');
};

export const setupAuthForms = () => {
	document.querySelectorAll<HTMLFormElement>('[data-auth-form]').forEach((form) => {
		const button = form.querySelector<HTMLButtonElement>('button[type="submit"]');
		const loader = form.querySelector<HTMLElement>('[data-auth-loading]');

		if (!button) return;

		form.querySelectorAll<HTMLInputElement>('input[required]').forEach((input) => {
			input.addEventListener('input', () => updateButton(form));
		});

		form.addEventListener('submit', async (event) => {
			event.preventDefault();

			if (!form.checkValidity()) {
				form.reportValidity();
				return;
			}

			setError(form, '');
			button.disabled = true;
			button.classList.add('btn-disabled');
			loader?.classList.remove('hidden');
			loader?.classList.add('flex');

			try {
				await submitAuthForm(form);
			} catch (error) {
				setError(
					form,
					error instanceof ApiError ? error.message : 'Could not reach the server. Please try again.',
				);
			} finally {
				loader?.classList.add('hidden');
				loader?.classList.remove('flex');
				updateButton(form);
			}
		});

		updateButton(form);
	});
};
