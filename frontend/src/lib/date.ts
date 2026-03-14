export const formatDateTime = (value?: string | null, fallback = 'Not provided'): string => {
	if (!value) return fallback;
	const date = new Date(value);
	if (Number.isNaN(date.getTime())) return value;
	return date.toLocaleDateString();
};

export const toDateTimeLocal = (value?: string | null): string => {
	if (!value) return '';
	const date = new Date(value);
	if (Number.isNaN(date.getTime())) return '';
	const pad = (num: number) => String(num).padStart(2, '0');
	return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
};

export const getCurrentDateTimeLocal = (): string => {
	const date = new Date();
	const pad = (num: number) => String(num).padStart(2, '0');
	return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
};

export const toDateLocal = (value?: string | null): string => {
	if (!value) return '';
	const date = new Date(value);
	if (Number.isNaN(date.getTime())) return '';
	const pad = (num: number) => String(num).padStart(2, '0');
	return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`;
};

export const getCurrentDateLocal = (): string => {
	const date = new Date();
	const pad = (num: number) => String(num).padStart(2, '0');
	return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`;
};
