<script lang="ts">
  import type { Entry, Meal } from '../../lib/diary';
  import { diary } from '../../lib/diary.svelte';
  import { errorMessage } from '../../lib/pb';

  // Just calories, to a meal or onto an entry.
  let { meal, parent, onadded }: { meal: Meal; parent: Entry | null; onadded: (e: Entry) => void } =
    $props();

  let quick = $state({ name: '', kcal: undefined as number | undefined });
  let busy = $state(false);
  let error = $state('');

  async function save(e: SubmitEvent) {
    e.preventDefault();
    if (!quick.kcal) return;
    busy = true;
    error = '';
    try {
      const entry = await diary.quickAdd(
        meal,
        quick.name.trim() || 'Quick add',
        quick.kcal,
        parent ?? undefined,
      );
      quick = { name: '', kcal: undefined };
      onadded(entry);
    } catch (err) {
      error = errorMessage(err, "Couldn't add it. Please try again.");
    } finally {
      busy = false;
    }
  }
</script>

{#if error}
  <p class="mb-3 text-sm text-red-600 dark:text-red-400" role="alert">{error}</p>
{/if}

<form class="flex flex-1 flex-col gap-3" onsubmit={save}>
  <p class="text-sm text-zinc-600 dark:text-zinc-400">
    For when you just know the calories, like a slice of cake at a party.
  </p>
  <label
    >Calories <input
      type="number"
      inputmode="numeric"
      min="1"
      max="20000"
      required
      bind:value={quick.kcal}
    /></label
  >
  <label>What was it? (optional) <input maxlength="200" bind:value={quick.name} /></label>
  <button class="btn-primary mt-auto" disabled={busy || !quick.kcal}
    >{busy ? 'Adding…' : 'Add'}</button
  >
</form>
