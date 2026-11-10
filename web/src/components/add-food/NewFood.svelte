<script lang="ts">
  import type { Food } from '../../lib/diary';
  import { saveCustomFood } from '../../lib/diary.svelte';
  import { errorMessage } from '../../lib/pb';

  // A food of the person's own, from its label: values per 100 g and a typical
  // serving. After a barcode that wasn't found, it saves the barcode with it,
  // so the next scan finds it.
  let {
    name = '',
    barcode = '',
    notice = '',
    onsaved,
  }: {
    name?: string;
    barcode?: string;
    /** why it opened after a scan */
    notice?: string;
    onsaved: (f: Food) => void;
  } = $props();

  // svelte-ignore state_referenced_locally
  let custom = $state({
    name,
    brand: '',
    kcal: undefined as number | undefined,
    protein: undefined as number | undefined,
    carbs: undefined as number | undefined,
    fat: undefined as number | undefined,
    serving_g: undefined as number | undefined,
    serving_label: '',
  });
  let busy = $state(false);
  let error = $state('');

  async function save(e: SubmitEvent) {
    e.preventDefault();
    busy = true;
    error = '';
    try {
      const f = await saveCustomFood({
        name: custom.name.trim(),
        brand: custom.brand.trim(),
        kcal: custom.kcal ?? 0,
        protein: custom.protein ?? 0,
        carbs: custom.carbs ?? 0,
        fat: custom.fat ?? 0,
        serving_g: custom.serving_g ?? 0,
        serving_label: custom.serving_label.trim(),
        barcode,
      });
      onsaved(f);
    } catch (err) {
      error = errorMessage(err, "Couldn't save the food. Please try again.");
    } finally {
      busy = false;
    }
  }
</script>

{#if error}
  <p class="mb-3 text-sm text-red-600 dark:text-red-400" role="alert">{error}</p>
{/if}

<form class="flex flex-1 flex-col gap-3" onsubmit={save}>
  {#if barcode}
    <p class="text-sm text-zinc-600 dark:text-zinc-400">
      {notice} Copy the values from the label once, and the next scan of {barcode} finds it.
    </p>
  {/if}
  <label>Name <input required maxlength="200" bind:value={custom.name} /></label>
  <label>Brand (optional) <input maxlength="200" bind:value={custom.brand} /></label>
  <p class="text-sm font-medium">Per 100 g</p>
  <div class="grid grid-cols-2 gap-3">
    <label
      >Calories <input
        type="number"
        inputmode="decimal"
        min="0"
        max="1000"
        step="any"
        required
        bind:value={custom.kcal}
      /></label
    >
    <label
      >Carbs (g) <input
        type="number"
        inputmode="decimal"
        min="0"
        max="100"
        step="any"
        bind:value={custom.carbs}
      /></label
    >
    <label
      >Protein (g) <input
        type="number"
        inputmode="decimal"
        min="0"
        max="100"
        step="any"
        bind:value={custom.protein}
      /></label
    >
    <label
      >Fat (g) <input
        type="number"
        inputmode="decimal"
        min="0"
        max="100"
        step="any"
        bind:value={custom.fat}
      /></label
    >
  </div>
  <div class="grid grid-cols-2 gap-3">
    <label
      >Serving (g) <input
        type="number"
        inputmode="decimal"
        min="0"
        max="5000"
        step="any"
        bind:value={custom.serving_g}
      /></label
    >
    <label
      >Serving name <input
        maxlength="50"
        placeholder="1 slice"
        bind:value={custom.serving_label}
      /></label
    >
  </div>
  <button class="btn-primary mt-auto" disabled={busy}>{busy ? 'Saving…' : 'Save food'}</button>
</form>
