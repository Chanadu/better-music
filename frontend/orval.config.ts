import { defineConfig } from 'orval';

const apiMutator = (name: string) => ({ path: './src/lib/scripts/api-fetch.ts', name });
const accountChange = { mutator: apiMutator('accountChangeFetch') };

export default defineConfig({
	betterMusic: {
		input: './openapi.json',
		output: {
			target: './src/lib/scripts/generated/api.ts',
			client: 'fetch',
			override: {
				operations: {
					getSession: { mutator: apiMutator('sessionFetch') },
					updateAccountEmail: accountChange,
					updateAccountPassword: accountChange,
					deleteAccount: accountChange,
				},
				fetch: {
					includeHttpResponseReturnType: false,
				},
				mutator: apiMutator('apiFetch'),
			},
		},
	},
});
