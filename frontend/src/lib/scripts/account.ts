import { ApiError } from './api';
import { clearTokens } from './auth';

export function accountErrorMessage(value: unknown) {
	return value instanceof ApiError ? value.message : 'Could not reach the server. Please try again.';
}

export function finishAccountAction() {
	clearTokens();
	location.assign('/login');
}
