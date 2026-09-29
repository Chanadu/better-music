import { ApiError } from './api';
import { clearSession } from './auth';

export function accountErrorMessage(value: unknown) {
	return value instanceof ApiError ? value.message : 'Could not reach the server. Please try again.';
}

export async function finishAccountAction() {
	await clearSession();
	location.assign('/login');
}
