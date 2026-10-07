export const syncConfig = {
	maximumFailures: 4,
	retryDelaysMs: [1000, 2000, 4000],
	interruptCheckIntervalMs: 1000,
	millisecondsPerSecond: 1000,
	http: {
		authentication: 401,
		conflict: 412,
		rateLimit: 429,
		retryable: [408, 425, 429, 500, 502, 503, 504],
		invalidData: [400, 422],
	},
};
