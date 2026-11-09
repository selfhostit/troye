<script lang="ts">
  import { amount, bar, chartTop, type Nutrient, type Report } from '../lib/insights';
  import { shortDate, ticks } from '../lib/weight';

  // One nutrient day by day: what was logged as a bar, an estimate for a
  // missing or half-logged day as a dashed bar on top, the 7-day average as
  // the line, and the target (for calories, each day's budget) as a dashed
  // line. Touching or hovering picks a day; the arrow keys step through them.
  let { report, nutrient }: { report: Report; nutrient: Nutrient } = $props();

  let width = $state(320);
  const height = 200;
  const pad = { top: 14, right: 12, bottom: 24, left: 40 };

  const days = $derived(report.days);
  const rolling = $derived(report.rolling[nutrient]);
  const goal = $derived(report.goals[nutrient]);
  const yTicks = $derived(ticks(0, chartTop(report, nutrient)));
  const yMax = $derived(yTicks.at(-1)!);

  const slot = $derived((width - pad.left - pad.right) / Math.max(days.length, 1));
  const barW = $derived(Math.max(Math.min(slot * 0.7, 18), 2));
  const cx = (i: number) => pad.left + slot * (i + 0.5);
  const y = (v: number) => pad.top + (1 - v / (yMax || 1)) * (height - pad.top - pad.bottom);

  const line = $derived.by(() => {
    let d = '';
    let pen = false;
    rolling.forEach((v, i) => {
      if (v === null) {
        pen = false;
        return;
      }
      d += `${pen ? 'L' : 'M'}${cx(i).toFixed(1)},${y(v).toFixed(1)}`;
      pen = true;
    });
    return d;
  });
  const xLabels = $derived(
    days.length > 1 ? [0, Math.floor(days.length / 2), days.length - 1] : [0],
  );

  let picked = $state<number | null>(null);
  const pick = $derived(picked !== null ? days[picked] : null);

  function nearest(e: PointerEvent) {
    const box = (e.currentTarget as SVGElement).getBoundingClientRect();
    const i = Math.floor((e.clientX - box.left - pad.left) / slot);
    picked = Math.min(Math.max(i, 0), days.length - 1);
  }

  function key(e: KeyboardEvent) {
    if (e.key !== 'ArrowLeft' && e.key !== 'ArrowRight') return;
    e.preventDefault();
    const step = e.key === 'ArrowLeft' ? -1 : 1;
    picked = Math.min(Math.max((picked ?? days.length) + step, 0), days.length - 1);
  }

  const statusText = {
    logged: '',
    partial: 'Half logged',
    missing: 'Not logged',
    today: 'Today so far',
  };
</script>

<div class="relative" bind:clientWidth={width}>
  <!-- svelte-ignore a11y_no_noninteractive_tabindex, a11y_no_noninteractive_element_interactions -->
  <svg
    {width}
    {height}
    class="block touch-pan-y outline-none select-none"
    role="img"
    aria-label="Chart of each day. Use the arrow keys to read each day."
    tabindex="0"
    onpointermove={nearest}
    onpointerdown={nearest}
    onpointerleave={(e) => e.pointerType === 'mouse' && (picked = null)}
    onkeydown={key}
    onblur={() => (picked = null)}
  >
    {#each yTicks as t (t)}
      <line
        x1={pad.left}
        x2={width - pad.right}
        y1={y(t)}
        y2={y(t)}
        class="stroke-zinc-100 dark:stroke-zinc-800"
      />
      <text
        x={pad.left - 6}
        y={y(t)}
        dy="0.32em"
        text-anchor="end"
        class="fill-zinc-500 text-[11px] tabular-nums dark:fill-zinc-400">{t.toLocaleString()}</text
      >
    {/each}
    {#each xLabels as i, n (i)}
      <text
        x={cx(i)}
        y={height - 6}
        text-anchor={xLabels.length === 1
          ? 'middle'
          : n === 0
            ? 'start'
            : n === 2
              ? 'end'
              : 'middle'}
        class="fill-zinc-500 text-[11px] dark:fill-zinc-400"
        >{shortDate(days[i].date, report.today)}</text
      >
    {/each}

    {#each days as d, i (d.date)}
      {@const b = bar(d, nutrient)}
      {#if b.logged > 0}
        <rect
          x={cx(i) - barW / 2}
          y={y(b.logged)}
          width={barW}
          height={y(0) - y(b.logged)}
          rx="2"
          class={picked === i
            ? 'fill-zinc-500 dark:fill-zinc-300'
            : 'fill-zinc-300 dark:fill-zinc-600'}
        />
      {/if}
      {#if b.estimated > 0}
        <rect
          x={cx(i) - barW / 2 + 0.5}
          y={y(b.logged + b.estimated)}
          width={barW - 1}
          height={y(b.logged) - y(b.logged + b.estimated)}
          rx="2"
          fill="none"
          stroke-dasharray="3 2"
          class="stroke-zinc-400 dark:stroke-zinc-500"
        />
      {/if}
      {#if nutrient === 'kcal' && d.budget > 0}
        <line
          x1={cx(i) - slot / 2}
          x2={cx(i) + slot / 2}
          y1={y(d.budget)}
          y2={y(d.budget)}
          stroke-dasharray="3 3"
          class="stroke-zinc-400 dark:stroke-zinc-500"
        />
      {/if}
    {/each}

    {#if nutrient !== 'kcal' && goal}
      <line
        x1={pad.left}
        x2={width - pad.right}
        y1={y(goal.target)}
        y2={y(goal.target)}
        stroke-dasharray="4 4"
        class="stroke-zinc-400 dark:stroke-zinc-500"
      />
    {/if}

    <path
      d={line}
      fill="none"
      stroke-width="2"
      stroke-linejoin="round"
      stroke-linecap="round"
      class="stroke-orange-500"
    />
  </svg>

  {#if pick && picked !== null}
    {@const b = bar(pick, nutrient)}
    <div
      class="pointer-events-none absolute top-0 rounded-lg border border-zinc-200 bg-white px-2.5 py-1.5 text-xs shadow-sm dark:border-zinc-700 dark:bg-zinc-800"
      style="left: {cx(picked) > width / 2 ? cx(picked) - 152 : cx(picked) + 12}px; width: 140px"
      aria-live="polite"
    >
      <b class="block text-sm tabular-nums">{amount(nutrient, b.logged)}</b>
      {#if statusText[pick.status]}
        <span class="block text-zinc-500 dark:text-zinc-400">{statusText[pick.status]}</span>
      {/if}
      {#if b.estimated > 0}
        <span class="block text-zinc-500 dark:text-zinc-400"
          >likely {amount(nutrient, b.logged + b.estimated)}</span
        >
      {/if}
      {#if rolling[picked] !== null}
        <span class="flex items-center gap-1.5 text-zinc-500 dark:text-zinc-400">
          <span class="inline-block h-0.5 w-3 rounded bg-orange-500"></span>
          7 days {amount(nutrient, rolling[picked]!)}
        </span>
      {/if}
      <span class="block text-zinc-500 dark:text-zinc-400"
        >{shortDate(pick.date, report.today)}</span
      >
    </div>
  {/if}
</div>
