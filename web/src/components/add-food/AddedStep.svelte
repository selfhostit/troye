<script lang="ts">
  import { addOnIdeas, entryAmount, foodName, meals, type Entry, type Food } from '../../lib/diary';
  import { diary } from '../../lib/diary.svelte';
  import { profile } from '../../lib/profile.svelte';

  // What was just logged, with what went with it, and what usually does: what
  // this person added to the food before, then ideas to search for (milk for a
  // coffee, an egg for a noodle soup).
  let {
    base,
    usual,
    onpick,
    onsearch,
    ondone,
  }: {
    base: Entry;
    /** what this person added to the food before, most used first */
    usual: Food[];
    onpick: (f: Food) => void;
    /** searches for an add-on, with these words or none */
    onsearch: (words: string) => void;
    ondone: () => void;
  } = $props();

  const name = (f: Food) => foodName(f, profile.current?.food_language);
  const addOns = $derived(diary.addOnsOf(base));
  const ideas = $derived(
    addOnIdeas(base.name).filter(
      (w) =>
        ![...usual.map(name), ...addOns.map((a) => a.name)].some((n) =>
          n.toLowerCase().includes(w),
        ),
    ),
  );
</script>

<div class="flex flex-1 flex-col gap-4">
  <div class="card flex items-center gap-4">
    <img src="/brand/plato-cheer.svg" alt="" class="size-12 shrink-0" />
    <div class="min-w-0 flex-1">
      <p class="truncate font-semibold">{base.name}</p>
      <p class="text-sm text-zinc-500 dark:text-zinc-400">
        {[entryAmount(base), meals.find((m) => m.id === base.meal)?.label]
          .filter(Boolean)
          .join(' · ')}
      </p>
    </div>
    <span class="shrink-0 font-semibold tabular-nums">
      {Math.round(base.kcal + addOns.reduce((s, a) => s + a.kcal, 0))} kcal
    </span>
  </div>
  {#if addOns.length}
    <ul class="-mt-2 space-y-1 px-1 text-sm text-zinc-600 dark:text-zinc-400">
      {#each addOns as a (a.id)}
        <li class="flex justify-between gap-3">
          <span class="truncate">+ {a.name} · {entryAmount(a)}</span>
          <span class="shrink-0 tabular-nums">{Math.round(a.kcal)}</span>
        </li>
      {/each}
    </ul>
  {/if}

  <div>
    <h3 class="text-sm font-medium">Anything with it?</h3>
    <div class="mt-2 flex flex-wrap gap-2">
      {#each usual.filter((f) => !addOns.some((a) => a.food === f.id)).slice(0, 4) as f (f.id)}
        <button class="chip" onclick={() => onpick(f)}>+ {name(f)}</button>
      {/each}
      {#each ideas.slice(0, 6 - Math.min(usual.length, 4)) as idea (idea)}
        <button class="chip" onclick={() => onsearch(idea)}>+ {idea}</button>
      {/each}
      <button
        class="chip font-medium text-brand-600 dark:text-brand-400"
        onclick={() => onsearch('')}>+ Something else</button
      >
    </div>
  </div>

  <button class="btn-primary mt-auto" onclick={ondone}>Done</button>
</div>
