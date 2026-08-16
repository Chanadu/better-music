export function withReturnTo(href: string, currentUrl: URL) {
	const target = new URL(href, currentUrl.origin);
	target.searchParams.set('from', `${currentUrl.pathname}${currentUrl.search}${currentUrl.hash}`);
	return `${target.pathname}${target.search}${target.hash}`;
}

export function getReturnHref(currentUrl: URL, fallback: string) {
	const from = currentUrl.searchParams.get('from');
	if (!from?.startsWith('/') || from.startsWith('//')) return fallback;

	try {
		const target = new URL(from, currentUrl.origin);
		if (target.origin !== currentUrl.origin) return fallback;
		return `${target.pathname}${target.search}${target.hash}`;
	} catch {
		return fallback;
	}
}
