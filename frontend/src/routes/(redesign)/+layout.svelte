<script lang="ts">
	import './redesign.css';
	import { beforeNavigate } from '$app/navigation';

	let { children } = $props();

	// Leave this document so styles from the two layouts never mix.
	beforeNavigate(({ to, cancel }) => {
		if (to?.route.id?.startsWith('/(library)')) {
			cancel();
			location.assign(to.url.href);
		}
	});
</script>

{@render children()}
