<script lang="ts">
	import FloatingField from '../common/FloatingField.svelte';
	import MediaThumbnail from '../common/MediaThumbnail.svelte';
	import type { Artist } from '$lib/scripts/types';

	let {
		artists,
		artistId = $bindable(''),
		name = $bindable(''),
		year = $bindable(''),
		comment = $bindable(''),
	}: {
		artists: Artist[];
		artistId?: string;
		name?: string;
		year?: string;
		comment?: string;
	} = $props();

	function isDigit(character: string) {
		return character >= '0' && character <= '9';
	}

	function cleanYear(event: Event) {
		year = [...(event.currentTarget as HTMLInputElement).value].filter(isDigit).join('');
	}
</script>

<div class="flex flex-col gap-3">
	<FloatingField label="Artist">
		<select class="select w-full" bind:value={artistId}>
			<option value="">Select artist</option>
			{#each artists as artist}
				<option value={artist.id}>{artist.name}</option>
			{/each}
		</select>
	</FloatingField>

	<div class="flex items-center gap-4">
		<div class="basis-1/3 pt-3">
			<MediaThumbnail
				variant="album-preview"
				label={name}
				alt={name ? `${name} album preview` : ''}
				emptyFallback=""
			/>
		</div>

		<div class="flex min-w-0 flex-1 flex-col gap-3">
			<FloatingField label="Album Title">
				<input type="text" class="input w-full" placeholder="e.g. 2014 Forest Hills Drive" bind:value={name} />
			</FloatingField>

			<FloatingField label="Year">
				<input
					type="text"
					inputmode="numeric"
					pattern="[0-9]{4}"
					maxlength="4"
					class="input w-full"
					class:validator={!!year}
					placeholder="2014"
					value={year}
					oninput={cleanYear}
				/>
				<div class="validator-hint hidden">Enter a valid year</div>
			</FloatingField>
		</div>
	</div>

	<FloatingField label="Comment">
		<textarea class="textarea w-full" placeholder="Classic album" bind:value={comment}></textarea>
	</FloatingField>
</div>
