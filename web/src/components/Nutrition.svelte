<script lang="ts">
  import {
    amount,
    insightRanges,
    loadInsights,
    nutrients,
    warningText,
    warningTip,
    type InsightRange,
    type Nutrient,
    type Report,
  } from '../lib/insights';
  import { errorMessage } from '../lib/pb';
  import NutrientChart from './NutrientChart.svelte';

  // What you ate over time: warnings when last week's average was off
  // target, one nutrient's chart, and the week's averages. Days you didn't
  // log are estimated from your usual days, and shown as estimates.
  let range = $state<InsightRange>('1m');
  let nutrient = $state<Nutrient>('kcal');
  let report = $state<Report | null>(null);
  let error = $state('');

  $effect(() => {
    const r = range;
    error = '';
    loadInsights(insightRanges.find((x) => x.id === r)!.days)
      .then((rep) => r === range && (report = rep))
      .catch((err) => (error = errorMessage(err, "Couldn't load your nutrition.")));
  });

  const week = $derived(report?.week ?? null);
  const counted = $derived(week ? week.logged + week.estimated : 0);
  const goalOf = (n: Nutrient) =>
    n === 'kcal' ? (week?.budget ?? 0) : (report?.goals[n]?.target ?? 0);
  const weekText = $derived.by(() => {
    if (!week) return '';
    const parts = [`${week.logged} logged`];
    if (week.estimated) parts.push(`${week.estimated} estimated`);
    if (week.missing) parts.push(`${week.missing} missing`);
    return parts.join(', ');
  });
</script>

<header>
  <h1 class="text-2xl font-bold tracking-tight">Nutrition</h1>
</header>

{#if error}
  <p
    class="rounded-xl bg-red-50 px-4 py-2 text-sm text-red-700 dark:bg-red-500/10 dark:text-red-300"
    role="alert"
  >
    {error}
  </p>
{/if}

{#if report}
  {#each report.warnings as w (w.nutrient)}
    <section
      class="card flex flex-col gap-1 border-orange-200 bg-orange-50 py-4 dark:border-orange-900 dark:bg-orange-950/40"
      role="status"
    >
      <p class="font-semibold">{warningText(w)}</p>
      <p class="text-sm text-zinc-600 dark:text-zinc-400">
        Over the last 7 days. {warningTip(w)}
      </p>
    </section>
  {/each}

  <div class="flex gap-1.5 overflow-x-auto pb-1" role="radiogroup" aria-label="Nutrient">
    {#each nutrients as n (n.id)}
      <button
        class="chip shrink-0 py-1.5"
        class:chosen={nutrient === n.id}
        role="radio"
        aria-checked={nutrient === n.id}
        onclick={() => (nutrient = n.id)}>{n.label}</button
      >
    {/each}
  </div>

  <div class="flex gap-1" role="radiogroup" aria-label="Time range">
    {#each insightRanges as r (r.id)}
      <button
        class="chip flex-1 px-0 py-1.5"
        class:chosen={range === r.id}
        role="radio"
        aria-checked={range === r.id}
        onclick={() => (range = r.id)}>{r.label}</button
      >
    {/each}
  </div>

  <section class="card px-3 pb-3">
    <NutrientChart {report} {nutrient} />
    <div class="mt-2 flex flex-wrap gap-x-4 gap-y-1 px-2 text-xs text-zinc-500 dark:text-zinc-400">
      <span class="flex items-center gap-1.5"
        ><span class="inline-block size-2 rounded-sm bg-zinc-300 dark:bg-zinc-600"
        ></span>Logged</span
      >
      <span class="flex items-center gap-1.5"
        ><span
          class="inline-block size-2 rounded-sm border border-dashed border-zinc-400 dark:border-zinc-500"
        ></span>Estimated</span
      >
      <span class="flex items-center gap-1.5"
        ><span class="inline-block h-0.5 w-3 rounded bg-orange-500"></span>7-day average</span
      >
      <span class="flex items-center gap-1.5"
        ><span class="inline-block w-3 border-t border-dashed border-zinc-400"></span>{nutrient ===
        'kcal'
          ? 'Budget'
          : 'Target'}</span
      >
    </div>
  </section>

  <section class="card flex flex-col gap-3 text-sm">
    <div class="flex items-baseline justify-between gap-4">
      <h2 class="font-semibold">Last 7 days</h2>
      <span class="text-zinc-500 dark:text-zinc-400">{weekText}</span>
    </div>
    {#if counted >= 4}
      {#each nutrients as n (n.id)}
        <div class="flex justify-between gap-4">
          <span class="text-zinc-500 dark:text-zinc-400">{n.label}</span>
          <span class="text-right font-medium tabular-nums">
            {amount(n.id, week!.average[n.id])}
            {#if goalOf(n.id)}
              <span class="font-normal text-zinc-500 dark:text-zinc-400"
                >/ {amount(n.id, goalOf(n.id))}</span
              >
            {/if}
          </span>
        </div>
      {/each}
      {#if week!.estimated}
        <p class="text-xs text-zinc-500 dark:text-zinc-400">
          Days you didn't fully log are estimated from your usual days. Logging them makes this more
          accurate.
        </p>
      {/if}
    {:else}
      <p class="text-zinc-500 dark:text-zinc-400">
        Log at least four days in a week to see your averages and warnings.
      </p>
    {/if}
  </section>
{/if}
