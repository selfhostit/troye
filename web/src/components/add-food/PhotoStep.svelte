<script lang="ts">
  import { onDestroy } from 'svelte';
  import { meals, type Meal } from '../../lib/diary';
  import { diary } from '../../lib/diary.svelte';
  import { errorMessage } from '../../lib/pb';
  import { statusText, type MealPhoto } from '../../lib/photos';
  import { photos } from '../../lib/photos.svelte';
  import AnalyserCard from '../AnalyserCard.svelte';

  // A meal photo: sent the moment it is taken, so it is never lost, and then
  // where it is and whether the PC is there to analyse it. A note for the
  // analyser, or another meal, is saved on it when the person is done.
  let {
    file,
    meal,
    sent = $bindable(null),
    onreview,
    onclose,
  }: {
    file: File;
    meal: Meal;
    /** the queued photo, once it is sent */
    sent?: MealPhoto | null;
    onreview: (p: MealPhoto) => void;
    onclose: () => void;
  } = $props();

  // svelte-ignore state_referenced_locally
  const preview = URL.createObjectURL(file);
  onDestroy(() => URL.revokeObjectURL(preview));
  let note = $state('');
  let busy = $state(false);
  let error = $state('');
  // the queued photo as it changes, so it shows being analysed
  const live = $derived(sent ? (photos.pending.find((p) => p.id === sent!.id) ?? sent) : null);

  async function send() {
    busy = true;
    error = '';
    try {
      sent = await photos.add(file, meal, diary.day, '');
    } catch (err) {
      error = errorMessage(err, "Couldn't send the photo. Please try again.");
    } finally {
      busy = false;
    }
  }
  send();

  /** Saves a changed note or meal on the sent photo, then closes. */
  export async function done() {
    const p = live;
    const text = note.trim();
    if (p && p.status !== 'logged' && (text !== p.note || meal !== p.meal)) {
      busy = true;
      try {
        await photos.edit(p, { note: text, meal });
      } catch (err) {
        error = errorMessage(err, "Couldn't save the note.");
        busy = false;
        return;
      }
    }
    onclose();
  }
</script>

{#if error}
  <p class="mb-3 text-sm text-red-600 dark:text-red-400" role="alert">{error}</p>
{/if}

<div class="flex flex-1 flex-col gap-3">
  <img src={preview} alt="Your meal" class="max-h-64 w-full rounded-2xl object-cover" />
  {#if live}
    <div class="card flex items-center gap-4" role="status">
      <img
        src={live.status === 'ready'
          ? '/brand/plato-cheer.svg'
          : live.status === 'failed'
            ? '/brand/plato.svg'
            : '/brand/plato-thinking.svg'}
        alt=""
        class="size-14 shrink-0"
      />
      <div class="min-w-0">
        <p class="font-medium">{statusText(live, photos.analyser.online)}</p>
        <p class="text-sm text-zinc-500 dark:text-zinc-400">
          {#if live.status === 'ready'}
            Check what it found before it goes in your diary.
          {:else if live.status === 'failed'}
            {live.error}
          {:else}
            It's saved under {meals.find((m) => m.id === live.meal)?.label.toLowerCase()} in your diary.
            You'll get a notification when it's ready to check.
          {/if}
        </p>
      </div>
    </div>
    {#if live.status === 'ready' || live.status === 'failed'}
      <button class="btn-primary" onclick={() => onreview(live)}>Check it now</button>
    {:else}
      <AnalyserCard always />
      <label
        >Anything it can't see? (optional)
        <input maxlength="200" placeholder="Cooked in butter, no sauce" bind:value={note} /></label
      >
    {/if}
    <button class="btn-secondary mt-auto" disabled={busy} onclick={done}>Done</button>
  {:else if busy}
    <p class="text-sm text-zinc-600 dark:text-zinc-400" role="status">Sending your photo…</p>
  {:else}
    <button class="btn-primary mt-auto" onclick={send}>Try again</button>
  {/if}
</div>
