const ratingColors = [
	'#DC2626',
	'#EF4444',
	'#F97316',
	'#FB923C',
	'#FBBF24',
	'#B5CC38',
	'#84CC16',
	'#22C55E',
	'#10B981',
	'#14B8A6',
] as const;

export function ratingColor(rating: number | null | undefined): string {
	if (rating == null || !Number.isFinite(rating) || rating < 1 || rating > 10) return '#9CA3AF';
	return ratingColors[Math.round(rating) - 1];
}
