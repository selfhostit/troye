<script lang="ts">
  import { amount, offerBudget, trendText, weekdayText, type Report } from '../lib/insights';
  import { errorMessage } from '../lib/pb';
  import { formatPace } from '../lib/profile';
  import { profile } from '../lib/profile.svelte';

  // What takes weeks to show: your real daily burn from eating and weighing
  // (with the budget it gives), nutrients drifting, and a weekday that runs
  // over. Each card waits until there is enough data to say it.
  let { report, onchange }: { report: Report; onchange: () => void } = $props();

  const units = $derived(profile.current?.units ?? 'metric');
  const burn = $derived(report.burn);
  const advice = $derived(report.budget_advice);
  const offer = $derived(offerBudget(advice, burn));
  const top = $derived(
    Math.max(...(report.weekdays?.diffs ?? []).map((d) => Math.abs(d ?? 0)), 100),
  );
  const days = ['M', 'T', 'W', 'T', 'F', 'S', 'S'];

  let busy = $state(false);
  let error = $state('');

  async function plan(tdee: number) {
    busy = true;
    error = '';
    try {
      await profile.setMeasuredBurn(tdee);
      onchange();
    } catch (err) {
      error = errorMessage(err, "Couldn't change your budget.");
    } finally {
      busy = false;
    }
  }

  function weightText(kgPerWeek: number) {
    if (Math.abs(kgPerWeek) < 0.05) return 'your weight held steady';
    return `you ${kgPerWeek < 0 ? 'lost' : 'gained'} ${formatPace(Math.round(Math.abs(kgPerWeek) * 100) / 100, units)}`;
  }
</script>

{#if burn}
  <section class="card flex flex-col gap-2 text-sm">
    <h2 class="font-semibold">Your real burn</h2>
    <p>
      Over the last 4 weeks you ate about {amount('kcal', burn.intake_kcal)} a day and {weightText(
        burn.weight_per_week,
      )}. That means you burn about <b>{amount('kcal', burn.tdee)}</b> a day{#if advice}, where the
        formula guessed {amount('kcal', advice.formula_tdee)}{/if}.
    </p>
    {#if advice && offer}
      <p class="text-zinc-600 dark:text-zinc-400">
        Planning with it would make your budget {amount('kcal', advice.suggested)} instead of {amount(
          'kcal',
          advice.current,
        )}, at the same pace.
      </p>
      <button
        class="btn-primary self-start px-4 py-2 text-sm"
        disabled={busy}
        onclick={() => plan(burn.tdee)}>Use {amount('kcal', advice.suggested)}</button
      >
    {/if}
    {#if advice?.measured_tdee}
      <p class="text-zinc-600 dark:text-zinc-400">
        Your budget uses a measured burn of {amount('kcal', advice.measured_tdee)}.
        <button
          class="font-semibold text-brand-600 dark:text-brand-400"
          disabled={busy}
          onclick={() => plan(0)}>Use the formula again</button
        >
      </p>
    {/if}
    {#if error}<p class="text-red-600 dark:text-red-400" role="alert">{error}</p>{/if}
  </section>
{/if}

{#if report.trends.length}
  <section class="card flex flex-col gap-2 text-sm">
    <h2 class="font-semibold">Trends over 4 weeks</h2>
    {#each report.trends as t (t.nutrient)}
      <div class="flex justify-between gap-4">
        <span>{trendText(t)}</span>
        <span class="text-zinc-500 dark:text-zinc-400">avg {amount(t.nutrient, t.average)}</span>
      </div>
    {/each}
  </section>
{/if}

{#if report.weekdays}
  <section class="card flex flex-col gap-3 text-sm">
    <h2 class="font-semibold">Your week</h2>
    <p>{weekdayText(report.weekdays)}</p>
    <div class="grid h-20 grid-cols-7 items-center gap-2" aria-hidden="true">
      {#each report.weekdays.diffs as d, i (i)}
        <div class="relative h-full">
          <div
            class="absolute inset-x-0 top-1/2 border-t border-zinc-200 dark:border-zinc-700"
          ></div>
          {#if d !== null}
            <div
              class="absolute inset-x-1 rounded-sm {d > 0
                ? 'bg-orange-400'
                : 'bg-zinc-300 dark:bg-zinc-600'}"
              style={d > 0
                ? `bottom: 50%; height: ${(d / top) * 50}%`
                : `top: 50%; height: ${(-d / top) * 50}%`}
            ></div>
          {/if}
        </div>
      {/each}
    </div>
    <div class="grid grid-cols-7 gap-2 text-center text-xs text-zinc-500 dark:text-zinc-400">
      {#each days as d, i (i)}<span>{d}</span>{/each}
    </div>
  </section>
{/if}
