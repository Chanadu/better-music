<script lang="ts">
	let {
		value = $bindable(''),
		shown = $bindable(false),
		autocomplete = 'current-password',
		hint = 'Enter at least 4 characters',
	}: {
		value?: string;
		shown?: boolean;
		autocomplete?: 'current-password' | 'new-password';
		hint?: string;
	} = $props();
</script>

<div class="flex w-full flex-col gap-1">
	<label class="input validator w-full">
		<svg class="h-[1em] opacity-50" viewBox="0 0 24 24">
			<g stroke-linejoin="round" stroke-linecap="round" stroke-width="2.5" fill="none" stroke="currentColor">
				<rect width="18" height="11" x="3" y="11" rx="2"></rect>
				<path d="M7 11V7a5 5 0 0 1 10 0v4"></path>
			</g>
		</svg>

		<input
			type={shown ? 'text' : 'password'}
			name="password"
			placeholder="password"
			required
			minlength="4"
			{autocomplete}
			bind:value
		/>

		<button
			type="button"
			class="btn btn-square btn-ghost btn-xs swap swap-rotate"
			class:swap-active={shown}
			aria-label={shown ? 'Hide password' : 'Show password'}
			onclick={() => (shown = !shown)}
		>
			<svg
				class="swap-off h-4 w-4"
				viewBox="0 0 24 24"
				fill="none"
				stroke="currentColor"
				stroke-width="2"
				stroke-linecap="round"
				stroke-linejoin="round"
			>
				<path d="M2 12s3.5-7 10-7 10 7 10 7-3.5 7-10 7-10-7-10-7Z"></path>
				<circle cx="12" cy="12" r="3"></circle>
			</svg>
			<svg
				class="swap-on h-4 w-4"
				viewBox="0 0 24 24"
				fill="none"
				stroke="currentColor"
				stroke-width="2"
				stroke-linecap="round"
				stroke-linejoin="round"
			>
				<path d="M10.7 5.1A10.7 10.7 0 0 1 12 5c6.5 0 10 7 10 7a18.9 18.9 0 0 1-3.2 4.2"></path>
				<path d="M6.6 6.8C3.7 8.8 2 12 2 12s3.5 7 10 7a9.9 9.9 0 0 0 5.4-1.6"></path>
				<path d="M2 2l20 20"></path>
				<path d="M9.9 9.9A3 3 0 0 0 14.1 14.1"></path>
			</svg>
		</button>
	</label>

	<div class="validator-hint hidden w-full">{hint}</div>
</div>
