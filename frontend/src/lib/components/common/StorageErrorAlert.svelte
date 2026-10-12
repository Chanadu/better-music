<script lang="ts">
	let { message, onretry }: { message: string; onretry: () => Promise<unknown> } = $props();
	let retrying = $state(false);
	let retryError = $state('');
	async function retry() {
		if (retrying) return;
		retrying = true;
		retryError = '';
		try {
			await onretry();
		} catch (error) {
			retryError = error instanceof Error ? error.message : 'Storage is still unavailable. Please try again.';
		} finally {
			retrying = false;
		}
	}
</script>

<div class="border-error/20 bg-error/5 rounded-xl border p-4" role="alert">
	<div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
		<div class="min-w-0">
			<p class="text-error text-sm font-semibold">
				Unable to save on this device
			</p>

			<p class="text-base-content/70 mt-1 text-sm leading-relaxed break-words">
				{retryError || message}
			</p>
		</div>

		<button
			type="button"
			class="btn btn-error btn-soft btn-sm hover-lift shrink-0 self-start rounded-full sm:self-auto"
			disabled={retrying}
			onclick={retry}
		>
			{#if retrying}
				<span class="loading loading-xs" aria-hidden="true">
				</span>
			{/if}
			{retrying ? 'Trying…' : 'Try again'}
		</button>
	</div>
</div>
