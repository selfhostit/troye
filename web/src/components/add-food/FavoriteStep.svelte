<script lang="ts">
  import { amountOf, entryAmount, foodName, meals, type Food, type Meal } from '../../lib/diary';
  import { favoriteKcal, type Favorite, type FavoriteItem } from '../../lib/favorites';
  import { favorites } from '../../lib/favorites.svelte';
  import { errorMessage } from '../../lib/pb';
  import { profile } from '../../lib/profile.svelte';

  // A favorite, with last time's picks already chosen: swap a kind (Emmental
  // for Gouda), leave something out, change an amount, or put another food in,
  // then log it. What's picked is kept for next time. Edit renames it, takes
  // foods out of it, or deletes it.
  let {
    fav,
    items = $bindable(),
    meal,
    onamount,
    onkind,
    onfood,
    onlogged,
    ondeleted,
  }: {
    fav: Favorite;
    /** the favorite as picked this time */
    items: FavoriteItem[];
    meal: Meal;
    /** changes the amount of an item */
    onamount: (i: number) => void;
    /** finds another kind of an item */
    onkind: (i: number) => void;
    /** finds a food to put in it */
    onfood: () => void;
    onlogged: () => void;
    ondeleted: () => void;
  } = $props();

  let busy = $state(false);
  let error = $state('');
  let editing = $state(false);
  // svelte-ignore state_referenced_locally
  let name = $state(fav.name);

  const lang = $derived(profile.current?.food_language);
  const food = (id: string): Food | undefined => favorites.foods.get(id);
  const label = (id: string) => {
    const f = food(id);
    return f ? foodName(f, lang) : '…';
  };
  const kcal = $derived(favoriteKcal(items, favorites.foods));
  const mealLabel = $derived(meals.find((m) => m.id === meal)?.label.toLowerCase());
  const left = $derived(items.filter((it) => !it.off).length);

  function set(i: number, change: Partial<FavoriteItem>) {
    items = items.map((it, j) => (j === i ? { ...it, ...change } : it));
  }

  async function log() {
    busy = true;
    error = '';
    try {
      await favorites.log(fav, items, meal);
      onlogged();
    } catch (err) {
      error = errorMessage(err, "Couldn't add it. Please try again.");
    } finally {
      busy = false;
    }
  }

  async function save() {
    busy = true;
    error = '';
    try {
      await favorites.update(fav, { name: name.trim() || fav.name, items });
      editing = false;
    } catch (err) {
      error = errorMessage(err, "Couldn't save it. Please try again.");
    } finally {
      busy = false;
    }
  }

  async function remove() {
    if (!confirm(`Delete ${fav.name}? What you logged stays in your diary.`)) return;
    busy = true;
    try {
      await favorites.remove(fav);
      ondeleted();
    } catch (err) {
      error = errorMessage(err, "Couldn't delete it. Please try again.");
      busy = false;
    }
  }
</script>

{#if error}
  <p class="mb-3 text-sm text-red-600 dark:text-red-400" role="alert">{error}</p>
{/if}

<div class="flex flex-1 flex-col gap-3">
  {#if editing}
    <label>Name <input required maxlength="100" bind:value={name} /></label>
  {/if}

  <ul class="card divide-y divide-zinc-100 p-0 dark:divide-zinc-800">
    {#each items as it, i (i)}
      {@const c = it.choices[it.pick]}
      {@const f = food(c.food)}
      <li class="px-4 py-3" class:opacity-50={it.off && !editing}>
        <div class="flex items-center gap-3">
          {#if editing}
            <button
              class="-ml-1 rounded-full p-1 text-zinc-500 hover:bg-zinc-100 disabled:opacity-30 dark:hover:bg-zinc-800"
              aria-label="Take {label(c.food)} out"
              disabled={items.length === 1}
              onclick={() => (items = items.filter((_, j) => j !== i))}
            >
              <svg viewBox="0 0 24 24" class="size-5" fill="none" stroke="currentColor"
                ><path d="M6 6l12 12M18 6L6 18" stroke-width="2" stroke-linecap="round" /></svg
              >
            </button>
          {:else}
            <input
              type="checkbox"
              class="size-5 shrink-0 accent-brand-600"
              aria-label="Have {label(c.food)}"
              checked={!it.off}
              onchange={(e) => set(i, { off: !e.currentTarget.checked })}
            />
          {/if}
          <button
            class="flex min-w-0 flex-1 items-center justify-between gap-3 text-left"
            disabled={it.off}
            onclick={() => onamount(i)}
          >
            <span class="min-w-0">
              <span class="block truncate font-medium">{label(c.food)}</span>
              <span class="text-sm text-zinc-500 dark:text-zinc-400"
                >{entryAmount({ amount: c.amount, grams: c.grams })}</span
              >
            </span>
            <span class="shrink-0 text-sm tabular-nums">
              {f ? amountOf(f, c.grams).kcal : ''}
            </span>
          </button>
        </div>
        {#if !it.off && !editing}
          <div class="mt-2 ml-8 flex flex-wrap gap-1.5" role="radiogroup" aria-label="Kind">
            {#if it.choices.length > 1}
              {#each it.choices as k, j (k.food)}
                <button
                  class="chip px-2.5 py-1 text-sm"
                  class:chosen={j === it.pick}
                  role="radio"
                  aria-checked={j === it.pick}
                  onclick={() => set(i, { pick: j })}>{label(k.food)}</button
                >
              {/each}
            {/if}
            <button
              class="chip px-2.5 py-1 text-sm text-brand-600 dark:text-brand-400"
              onclick={() => onkind(i)}>{it.choices.length > 1 ? '+ other' : 'Swap'}</button
            >
          </div>
        {/if}
      </li>
    {/each}
  </ul>

  <button class="btn-secondary" onclick={onfood}>+ Put another food in</button>

  {#if editing}
    <button class="btn-primary mt-auto" disabled={busy} onclick={save}
      >{busy ? 'Saving…' : 'Save changes'}</button
    >
    <button class="btn-secondary text-red-600 dark:text-red-400" disabled={busy} onclick={remove}
      >Delete favorite</button
    >
  {:else}
    <button class="mx-auto text-sm text-zinc-500 underline" onclick={() => (editing = true)}
      >Rename or take foods out</button
    >
    <button class="btn-primary mt-auto" disabled={busy || !left} onclick={log}>
      {busy ? 'Adding…' : `Add to ${mealLabel} · ${kcal} kcal`}
    </button>
  {/if}
</div>
