<script lang="ts">
	import MediaThumbnail from '../common/MediaThumbnail.svelte';
	import type { SpotifyRow as Row } from '$lib/scripts/types';

	interface Props {
		row?: Row;
		type: 'artist' | 'album';
		loading: boolean;
		message: string;
		selected?: Row;
	}

	let { row, type, loading, message, selected = $bindable(undefined) }: Props = $props();
</script>

<li class="p-0">
	<label class={row ? 'block cursor-pointer' : 'block cursor-default'}>
		<input
			type="radio"
			name={`spotify_${type}`}
			class="peer sr-only"
			value={row?.id ?? ''}
			checked={!!row && selected?.id === row.id}
			disabled={!row || loading}
			onclick={(event) => {
				if (row && selected?.id === row.id) {
					event.preventDefault();
					selected = undefined;
				}
			}}
			onchange={(event) => {
				if (row && event.currentTarget.checked) selected = row;
			}}
		/>

		<div
			class="list-row hover:bg-base-300/50 peer-checked:bg-primary peer-checked:text-primary-content peer-checked:hover:bg-primary peer-checked:hover:text-primary-content items-center"
		>
			<MediaThumbnail
				variant="result"
				imageUrl={loading ? '' : (row?.imageUrl ?? '')}
				label={loading ? '' : (row?.name ?? '')}
				alt={row ? `${row.name} ${type} image` : ''}
				fallbackText={loading || !row ? '' : undefined}
			/>

			<div class="min-w-0">
				{#if loading}
					<span class="loading loading-dots loading-md"></span>
				{:else}
					<div class={`truncate ${!row ? 'text-base-content/35' : ''}`}>
						{row?.name ?? message}
					</div>

					<div class={`truncate text-xs ${!row ? 'text-base-content/35' : ''}`}>
						{row?.meta ?? ''}
					</div>
				{/if}
			</div>
		</div>
	</label>
</li>
