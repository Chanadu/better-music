<script lang="ts">
	import EyeIcon from '../icons/EyeIcon.svelte';
	import EyeOffIcon from '../icons/EyeOffIcon.svelte';
	import LockIcon from '../icons/LockIcon.svelte';

	let {
		value = $bindable(''),
		shown = $bindable(false),
		autocomplete = 'current-password',
		hint = 'Enter at least 4 characters',
		validate = true,
	}: {
		value?: string;
		shown?: boolean;
		autocomplete?: 'current-password' | 'new-password';
		hint?: string;
		validate?: boolean;
	} = $props();
</script>

<div class="flex w-full flex-col gap-1">
	<label class="input w-full" class:validator={validate}>
		<LockIcon class="h-[1em] opacity-50" />

		<input
			type={shown ? 'text' : 'password'}
			name="password"
			placeholder="password"
			required={validate}
			minlength={validate ? 4 : undefined}
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
			<EyeIcon class="swap-off h-4 w-4" />
			<EyeOffIcon class="swap-on h-4 w-4" />
		</button>
	</label>

	{#if validate}
		<div class="validator-hint hidden w-full">{hint}</div>
	{/if}
</div>
