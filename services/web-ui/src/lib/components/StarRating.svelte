<script lang="ts">
	export let rating: number;
	export let max: number = 5;
	export let size: number = 13;
</script>

<div class="flex items-center gap-0.5">
	{#each Array(max) as _, i}
		{@const filled = i < Math.floor(rating)}
		{@const partial = !filled && i < rating}
		<svg width={size} height={size} viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
			<path
				d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z"
				fill={filled ? '#d29922' : partial ? '#d2992260' : 'none'}
				stroke={filled || partial ? '#d29922' : '#484f58'}
				stroke-width="1.5"
				stroke-linejoin="round"
			/>
		</svg>
	{/each}
</div>
