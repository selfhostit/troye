<script lang="ts">
  import { foodName, type Entry, type Food } from '../../lib/diary';
  import { diary, searchFoods } from '../../lib/diary.svelte';
  import { errorMessage } from '../../lib/pb';
  import { startAmount } from '../../lib/portions';
  import { profile } from '../../lib/profile.svelte';

  // Finding a food: search (recent foods when nothing is typed), or another way
  // in: a photo, a barcode, a new food, or a quick add. When adding to an entry
  // (base), it asks what went with it, and the photo is left out.
  let {
    base,
    query = $bindable(''),
    onpick,
    onphoto,
    onscan,
    onnew,
    onquick,
  }: {
    base: Entry | null;
    query: string;
    onpick: (f: Food) => void;
    onphoto: (file: File) => void;
    onscan: () => void;
    onnew: (name: string) => void;
    onquick: () => void;
  } = $props();

  let results = $state<Food[]>([]);
  let searching = $state(false);
  let error = $state('');
  const name = (f: Food) => foodName(f, profile.current?.food_language);

  let timer: ReturnType<typeof setTimeout>;
  $effect(() => {
    const q = query;
    clearTimeout(timer);
    if (q.trim().length < 2) {
      results = [];
      return;
    }
    searching = true;
    timer = setTimeout(async () => {
      try {
        results = await searchFoods(q);
        error = '';
      } catch (err) {
        if ((err as { isAbort?: boolean }).isAbort) return;
        error = errorMessage(err);
      } finally {
        searching = false;
      }
    }, 250);
  });

  function pickedPhoto(e: Event) {
    const file = (e.currentTarget as HTMLInputElement).files?.[0];
    if (file) onphoto(file);
  }

  const shown = $derived(query.trim().length >= 2 ? results : diary.recent);
</script>

{#snippet foodRow(f: Food)}
  {@const usual = startAmount(f, diary.lastAmounts.get(f.id))}
  <li>
    <button
      class="flex w-full items-center justify-between gap-3 px-1 py-3 text-left"
      onclick={() => onpick(f)}
    >
      <span class="min-w-0">
        <span class="line-clamp-2 font-medium">{name(f)}</span>
        <span class="block truncate text-sm text-zinc-500 dark:text-zinc-400">
          {[f.brand, usual.amount ? `${usual.amount} (${usual.grams} g)` : `${usual.grams} g`]
            .filter(Boolean)
            .join(' · ')}
        </span>
      </span>
      <span class="shrink-0 text-right tabular-nums">
        <span class="block text-sm text-zinc-700 dark:text-zinc-200">
          {Math.round((f.kcal * usual.grams) / 100)} kcal
        </span>
        <span class="block text-xs text-zinc-500 dark:text-zinc-400"
          >{Math.round(f.kcal)} / 100 g</span
        >
      </span>
    </button>
  </li>
{/snippet}

{#if error}
  <p class="mb-3 text-sm text-red-600 dark:text-red-400" role="alert">{error}</p>
{/if}

<!-- svelte-ignore a11y_autofocus -->
<input
  type="search"
  placeholder={base ? `What did you add to ${base.name.toLowerCase()}?` : 'Search foods'}
  aria-label="Search foods"
  autocomplete="off"
  autofocus={!!base}
  bind:value={query}
/>
{#if !base}
  <label class="btn-primary mt-3 cursor-pointer flex-row gap-2">
    <svg viewBox="0 0 24 24" class="size-5" fill="none" stroke="currentColor" stroke-width="1.8"
      ><path
        d="M4 8h3l2-3h6l2 3h3v11H4z M12 17a4 4 0 1 0 0-8 4 4 0 0 0 0 8"
        stroke-linejoin="round"
      /></svg
    >
    Take a photo
    <input
      class="sr-only"
      type="file"
      accept="image/*"
      capture="environment"
      onchange={pickedPhoto}
    />
  </label>
{/if}
<div class="mt-2 grid grid-cols-3 gap-2 text-sm">
  <button class="btn-secondary px-2" onclick={onscan}>Scan barcode</button>
  <button class="btn-secondary px-2" onclick={() => onnew(query)}>New food</button>
  <button class="btn-secondary px-2" onclick={onquick}>Quick add</button>
</div>

<h3 class="mt-5 text-sm font-medium text-zinc-500 dark:text-zinc-400">
  {query.trim().length >= 2 ? (searching ? 'Searching…' : 'Results') : 'Recent'}
</h3>
{#if shown.length}
  <ul class="divide-y divide-zinc-200 dark:divide-zinc-800">
    {#each shown as f (f.id)}{@render foodRow(f)}{/each}
  </ul>
{:else if query.trim().length >= 2 && !searching}
  <p class="mt-2 text-sm text-zinc-500 dark:text-zinc-400">
    Nothing found. Try another word, or add it as a new food.
  </p>
{:else if query.trim().length < 2}
  <p class="mt-2 text-sm text-zinc-500 dark:text-zinc-400">
    Foods you log show up here, so the next time is one tap.
  </p>
{/if}
