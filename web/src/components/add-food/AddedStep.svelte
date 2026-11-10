<script lang="ts">
  import { addOnIdeas, entryAmount, foodName, meals, type Entry, type Food } from '../../lib/diary';
  import { diary } from '../../lib/diary.svelte';
  import { itemsFrom } from '../../lib/favorites';
  import { favorites } from '../../lib/favorites.svelte';
  import { errorMessage } from '../../lib/pb';
  import { profile } from '../../lib/profile.svelte';

  // What was just logged, with what went with it, and what usually does: what
  // this person added to the food before, then ideas to search for (milk for a
  // coffee, an egg for a noodle soup). It can be saved as a favorite, to log
  // the same again in one go.
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

  // saving it as a favorite: closed, naming it, or saved under a name
  let saving = $state<'no' | 'naming' | 'busy' | 'saved'>('no');
  // svelte-ignore state_referenced_locally
  let favName = $state(base.name);
  let error = $state('');
  const items = $derived(itemsFrom([base, ...addOns]));

  async function saveFavorite(e: SubmitEvent) {
    e.preventDefault();
    saving = 'busy';
    error = '';
    try {
      await favorites.create(favName.trim(), items);
      saving = 'saved';
    } catch (err) {
      error = errorMessage(err, "Couldn't save it. Please try again.");
      saving = 'naming';
    }
  }
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

  {#if error}
    <p class="text-sm text-red-600 dark:text-red-400" role="alert">{error}</p>
  {/if}
  {#if saving === 'saved'}
    <p class="text-sm text-zinc-600 dark:text-zinc-400" role="status">
      <span class="text-amber-500" aria-hidden="true">★</span> Saved as {favName.trim()}. It's at
      the top of Add food next time.
    </p>
  {:else if saving !== 'no'}
    <form class="flex items-end gap-2" onsubmit={saveFavorite}>
      <label class="flex-1"
        >Name the favorite
        <!-- svelte-ignore a11y_autofocus -->
        <input required maxlength="100" autofocus bind:value={favName} /></label
      >
      <button class="btn-primary" disabled={saving === 'busy'}>Save</button>
    </form>
  {:else if items.length}
    <button class="btn-secondary" onclick={() => (saving = 'naming')}
      ><span class="text-amber-500" aria-hidden="true">★</span> Save as favorite</button
    >
  {/if}

  <button class="btn-primary mt-auto" onclick={ondone}>Done</button>
</div>
