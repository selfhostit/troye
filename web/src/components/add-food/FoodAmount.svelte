<script lang="ts">
  import { amountOf, foodName, type Food } from '../../lib/diary';
  import { diary } from '../../lib/diary.svelte';
  import { startAmount } from '../../lib/portions';
  import { profile } from '../../lib/profile.svelte';
  import AmountPicker from '../AmountPicker.svelte';

  // How much of a food, starting from what was logged last time, with what it
  // comes to. The button says where it goes ("Add to breakfast").
  let {
    food,
    to,
    busy,
    onadd,
  }: {
    food: Food;
    /** where it goes, such as "breakfast" or "coffee" */
    to: string;
    busy: boolean;
    onadd: (grams: number, amount: string) => void;
  } = $props();

  // svelte-ignore state_referenced_locally
  const start = startAmount(food, diary.lastAmounts.get(food.id));
  let grams = $state<number | undefined>(start.grams);
  // the amount as counted, such as "2 tbsp", or empty for grams
  let amount = $state(start.amount);
  const preview = $derived(grams ? amountOf(food, grams) : null);
</script>

<div class="card">
  <p class="font-semibold">{foodName(food, profile.current?.food_language)}</p>
  {#if food.brand}<p class="text-sm text-zinc-500 dark:text-zinc-400">{food.brand}</p>{/if}
  <p class="mt-1 text-sm text-zinc-500 dark:text-zinc-400">{food.kcal} kcal per 100 g</p>
</div>
<div class="mt-4 flex flex-col gap-3">
  <AmountPicker {food} bind:grams bind:amount />
</div>
{#if preview}
  <div class="card mt-4 grid grid-cols-4 gap-2 text-center text-sm">
    <div><b class="block text-lg tabular-nums">{preview.kcal}</b>kcal</div>
    <div><b class="block text-lg tabular-nums">{preview.carbs}</b>carbs</div>
    <div><b class="block text-lg tabular-nums">{preview.protein}</b>protein</div>
    <div><b class="block text-lg tabular-nums">{preview.fat}</b>fat</div>
  </div>
{/if}
<button
  class="btn-primary mt-auto w-full"
  disabled={!grams || busy}
  onclick={() => grams && onadd(grams, amount)}
>
  {busy ? 'Adding…' : `Add to ${to}`}
</button>
