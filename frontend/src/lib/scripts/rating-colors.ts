import { appSettings } from './app-settings.svelte';

export function ratingColor(rating: number | null | undefined): string {
	if (rating == null || !Number.isFinite(rating) || rating < 1 || rating > 10) return '#9CA3AF';
	return appSettings.values.ratingColors[Math.round(rating) - 1];
}

export function ratingLabel(rating: number | null | undefined): string {
	if (rating == null || !Number.isFinite(rating) || rating < 1 || rating > 10) return '';
	return appSettings.values.ratingLabels[Math.round(rating) - 1];
}
