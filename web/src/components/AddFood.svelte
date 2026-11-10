<script lang="ts">
  import { onMount } from 'svelte';
  import { addOnIdeas, meals, type Entry, type Food, type Meal } from '../lib/diary';
  import { diary, lookupBarcode } from '../lib/diary.svelte';
  import {
    pickKind,
    type Favorite,
    type FavoriteChoice,
    type FavoriteItem,
  } from '../lib/favorites';
  import { favorites } from '../lib/favorites.svelte';
  import { errorMessage } from '../lib/pb';
  import type { MealPhoto } from '../lib/photos';
  import BarcodeScanner from './BarcodeScanner.svelte';
  import Sheet from './Sheet.svelte';
  import AddedStep from './add-food/AddedStep.svelte';
  import FavoriteStep from './add-food/FavoriteStep.svelte';
  import FoodAmount from './add-food/FoodAmount.svelte';
  import FoodSearch from './add-food/FoodSearch.svelte';
  import NewFood from './add-food/NewFood.svelte';
  import PhotoStep from './add-food/PhotoStep.svelte';
  import QuickAdd from './add-food/QuickAdd.svelte';

  // Adding food to a meal: a favorite, search (recent foods first), take a
  // photo, scan a barcode, make a custom food, or quick-add calories. Picking a
  // food asks how much. A favorite shows last time's picks to change before it
  // goes in; a food found for it (another kind of cheese) goes into the
  // favorite instead of the diary. Once a food is in, the sheet offers what usually goes with it (milk
  // for a coffee, an egg for a noodle soup), and those are logged as its
  // add-ons. Each step is its own component in add-food/; this sheet holds what
  // they share (the meal, the entry being added to) and moves between them.
  let {
    meal: startMeal,
    addTo,
    query: startQuery = '',
    onclose,
    onreview,
  }: {
    meal: Meal;
    /** opens it to add something to this entry */
    addTo?: Entry;
    query?: string;
    onclose: () => void;
    onreview: (p: MealPhoto) => void;
  } = $props();

  type View = 'search' | 'amount' | 'scan' | 'custom' | 'quick' | 'photo' | 'added' | 'favorite';
  let view = $state<View>('search');
  // the meal it opened for, then whichever the person picks
  // svelte-ignore state_referenced_locally
  let meal = $state<Meal>(startMeal);
  // svelte-ignore state_referenced_locally
  let query = $state(startQuery);
  // the entry being added to: given, or the food just logged
  // svelte-ignore state_referenced_locally
  let parent = $state<Entry | null>(addTo ?? null);
  // the parent as it is now in the diary, with what was added to it
  const base = $derived(parent ? (diary.entries.find((e) => e.id === parent!.id) ?? parent) : null);
  let usual = $state<Food[]>([]);
  let error = $state('');
  let busy = $state(false);

  // the food whose amount is being picked
  let food = $state<Food | null>(null);
  // the new-food form: a name to start from, and the barcode that wasn't found
  let newFood = $state({ name: '', barcode: '', notice: '' });
  // the photo step
  let photoFile = $state<File | null>(null);
  let sentPhoto = $state<MealPhoto | null>(null);
  let photoStep = $state<PhotoStep>();
  // the favorite being logged, as picked this time
  let fav = $state<Favorite | null>(null);
  let favItems = $state<FavoriteItem[]>([]);
  // as saved, with a new name once renamed
  const favNow = $derived(fav ? (favorites.list.find((f) => f.id === fav!.id) ?? fav) : null);
  // while set, the food picked goes into the favorite: as a kind of an item, or as a new item
  let into = $state<{ item: number | 'new'; from?: FavoriteChoice } | null>(null);

  onMount(() => {
    diary.loadRecent();
    favorites.load();
  });

  function pick(f: Food) {
    food = f;
    error = '';
    view = 'amount';
  }

  async function add(grams: number, amount: string) {
    if (!food) return;
    if (into) {
      putInFavorite({ food: food.id, grams, ...(amount ? { amount } : {}) });
      return;
    }
    busy = true;
    error = '';
    try {
      const e = await diary.add(meal, food, grams, amount, parent ?? undefined);
      await added(e);
    } catch (err) {
      error = errorMessage(err, "Couldn't add it. Please try again.");
    } finally {
      busy = false;
    }
  }

  function openFavorite(f: Favorite) {
    fav = f;
    favItems = f.items;
    into = null;
    view = 'favorite';
  }

  /** Finds a food for the favorite, or changes an item's amount (with its food). */
  function findFor(item: number | 'new', amountOnly = false) {
    const from = item === 'new' ? undefined : favItems[item].choices[favItems[item].pick];
    into = { item, from: amountOnly ? from : undefined };
    query = '';
    const f = from && favorites.foods.get(from.food);
    if (amountOnly && f) pick(f);
    else view = 'search';
  }

  function putInFavorite(choice: FavoriteChoice) {
    const item = into!.item;
    favItems =
      item === 'new'
        ? [...favItems, { choices: [choice], pick: 0 }]
        : favItems.map((it, i) => (i === item ? pickKind(it, choice) : it));
    favorites.remember([food!]);
    backToFavorite();
  }

  function backToFavorite() {
    into = null;
    query = '';
    view = 'favorite';
  }

  async function scanned(code: string) {
    busy = true;
    error = '';
    try {
      pick(await lookupBarcode(code));
    } catch (err) {
      newFood = { name: '', barcode: code, notice: errorMessage(err) };
      view = 'custom';
    } finally {
      busy = false;
    }
  }

  function photo(file: File) {
    photoFile = file;
    sentPhoto = null;
    view = 'photo';
  }

  const mealLabel = $derived(meals.find((m) => m.id === meal)?.label.toLowerCase() ?? '');

  /**
   * After logging: an add-on goes back to its entry, so more can go with it; a
   * food that things usually go with offers them; anything else is done.
   */
  async function added(e: Entry) {
    if (parent) {
      showAdded(parent);
      return;
    }
    const before = e.food ? await diary.usualAddOns(e.food) : [];
    if (!before.length && !addOnIdeas(e.name).length) {
      onclose();
      return;
    }
    showAdded(e, before);
  }

  async function showAdded(e: Entry, known?: Food[]) {
    parent = e;
    query = '';
    error = '';
    view = 'added';
    usual = known ?? (e.food ? await diary.usualAddOns(e.food) : []);
  }

  function addOn(words: string) {
    query = words;
    view = 'search';
  }

  const titles: Record<View, string> = {
    added: 'Added',
    search: 'Add food',
    amount: 'How much?',
    scan: 'Scan a barcode',
    custom: 'New food',
    quick: 'Quick add',
    photo: 'Meal photo',
    favorite: '',
  };
</script>

<Sheet
  title={view === 'photo' && sentPhoto
    ? 'Photo received'
    : favNow && (view === 'favorite' || (into && view === 'search'))
      ? favNow.name
      : base && view === 'search'
        ? `Add to ${base.name}`
        : addTo && view === 'added'
          ? addTo.name
          : titles[view]}
  onclose={into
    ? backToFavorite
    : view === 'added' || (view === 'search' && !base)
      ? onclose
      : view === 'search'
        ? () => showAdded(base!)
        : view === 'photo' && sentPhoto
          ? () => photoStep?.done()
          : () => (view = 'search')}
>
  {#if !base}
    <div class="mb-4 flex gap-1 overflow-x-auto" role="radiogroup" aria-label="Meal">
      {#each meals as m (m.id)}
        <button
          class="chip shrink-0 py-1.5"
          class:chosen={meal === m.id}
          role="radio"
          aria-checked={meal === m.id}
          onclick={() => (meal = m.id)}>{m.label}</button
        >
      {/each}
    </div>
  {/if}

  {#if error}
    <p class="mb-3 text-sm text-red-600 dark:text-red-400" role="alert">{error}</p>
  {/if}

  {#if view === 'favorite' && favNow}
    <FavoriteStep
      fav={favNow}
      bind:items={favItems}
      {meal}
      onamount={(i) => findFor(i, true)}
      onkind={(i) => findFor(i)}
      onfood={() => findFor('new')}
      onlogged={onclose}
      ondeleted={() => ((fav = null), (view = 'search'))}
    />
  {:else if view === 'search' && into}
    <FoodSearch
      placeholder={into.item === 'new' ? 'What else is in it?' : 'Which kind?'}
      bind:query
      onpick={pick}
      onscan={() => (view = 'scan')}
      onnew={(name) => ((newFood = { name, barcode: '', notice: '' }), (view = 'custom'))}
    />
  {:else if view === 'added' && base}
    <AddedStep {base} {usual} onpick={pick} onsearch={addOn} ondone={onclose} />
  {:else if view === 'search'}
    <FoodSearch
      {base}
      bind:query
      onpick={pick}
      onphoto={photo}
      onscan={() => (view = 'scan')}
      onnew={(name) => ((newFood = { name, barcode: '', notice: '' }), (view = 'custom'))}
      onquick={() => (view = 'quick')}
      onfavorite={openFavorite}
    />
  {:else if view === 'amount' && food}
    {#key food.id}
      <FoodAmount
        {food}
        {busy}
        from={into?.from}
        action={into && fav
          ? `Put it in ${fav.name.toLowerCase()}`
          : `Add to ${base ? base.name.toLowerCase() : mealLabel}`}
        onadd={add}
      />
    {/key}
  {:else if view === 'scan'}
    {#if busy}
      <p class="text-center text-zinc-500 dark:text-zinc-400">Looking it up…</p>
    {:else}
      <BarcodeScanner onfound={scanned} />
    {/if}
  {:else if view === 'custom'}
    <NewFood {...newFood} onsaved={pick} />
  {:else if view === 'photo' && photoFile}
    {#key photoFile}
      <PhotoStep
        bind:this={photoStep}
        bind:sent={sentPhoto}
        file={photoFile}
        {meal}
        {onreview}
        {onclose}
      />
    {/key}
  {:else if view === 'quick'}
    <QuickAdd {meal} {parent} onadded={added} />
  {/if}
</Sheet>
