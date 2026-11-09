<script lang="ts">
  import { onMount } from 'svelte';
  import Nutrition from './Nutrition.svelte';
  import Weight from './Weight.svelte';

  // Progress: your weight, and what you ate. Each view has its own #address
  // (#weight, #nutrition) so links and Back land on it.
  const views = [
    { id: 'weight', label: 'Weight' },
    { id: 'nutrition', label: 'Nutrition' },
  ] as const;
  type View = (typeof views)[number]['id'];

  const fromHash = (): View => (location.hash === '#nutrition' ? 'nutrition' : 'weight');
  let view = $state<View>(fromHash());

  onMount(() => {
    const onHash = () => (view = fromHash());
    addEventListener('hashchange', onHash);
    return () => removeEventListener('hashchange', onHash);
  });
</script>

<nav
  class="grid grid-cols-2 gap-1 rounded-xl bg-zinc-100 p-1 dark:bg-zinc-900"
  aria-label="Progress"
>
  {#each views as v (v.id)}
    <a
      href="#{v.id}"
      class="rounded-lg py-1.5 text-center text-sm font-medium {view === v.id
        ? 'bg-white shadow-sm dark:bg-zinc-800'
        : 'text-zinc-500'}"
      aria-current={view === v.id ? 'page' : undefined}>{v.label}</a
    >
  {/each}
</nav>

{#if view === 'nutrition'}
  <Nutrition />
{:else}
  <Weight />
{/if}
