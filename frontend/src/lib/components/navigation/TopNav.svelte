<script lang="ts">
	import { logout } from '$lib/scripts/auth';
	import LogoutIcon from '../icons/LogoutIcon.svelte';

	let { breadcrumbs }: { breadcrumbs: string[] } = $props();
	let loggingOut = $state(false);

	async function signOut() {
		loggingOut = true;
		await logout();
		location.assign('/login');
	}
</script>

<header class="navbar">
	<div class="navbar-start">
		<nav class="breadcrumbs text-base-content px-1 text-2xl font-bold" aria-label="Breadcrumb">
			<ul>
				{#each breadcrumbs as breadcrumb, index}
					<li>
						<span aria-current={index === breadcrumbs.length - 1 ? 'page' : undefined}>
							{breadcrumb}
						</span>
					</li>
				{/each}
			</ul>
		</nav>
	</div>

	<div class="navbar-end">
		<button
			type="button"
			class="btn btn-square btn-outline btn-accent"
			aria-label="Log out"
			disabled={loggingOut}
			onclick={signOut}
		>
			<LogoutIcon class="h-5 w-5" />
		</button>
	</div>
</header>
