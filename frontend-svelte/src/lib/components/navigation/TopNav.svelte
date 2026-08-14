<script lang="ts">
	import { logout } from '$lib/scripts/auth';

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
			<svg
				viewBox="0 0 24 24"
				fill="none"
				stroke="currentColor"
				stroke-width="2"
				stroke-linecap="round"
				stroke-linejoin="round"
				class="h-5 w-5"
			>
				<path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"></path>
				<path d="M16 17l5-5-5-5"></path>
				<path d="M21 12H9"></path>
			</svg>
		</button>
	</div>
</header>
