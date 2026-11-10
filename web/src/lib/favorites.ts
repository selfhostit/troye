import type { RecordModel } from 'pocketbase';
import { amountOf, foodName, type Entry, type Food, type FoodLanguage } from './diary';

// Favorites: what someone eats often, such as their breakfast toast, to log in
// one go. Each item is a food with the kinds the person has had of it (Gouda
// or Emmental, a boiled or a fried egg), the one picked last time, and whether
// it was left out. The server logs the first item with the rest as its add-ons.

/** One kind of an item: a food and how much of it. */
export interface FavoriteChoice {
  food: string;
  grams: number;
  /** as counted, such as "1 slice"; empty for grams */
  amount?: string;
}

export interface FavoriteItem {
  choices: FavoriteChoice[];
  /** the choice picked last time */
  pick: number;
  /** left out last time */
  off?: boolean;
}

export interface Favorite extends RecordModel {
  user: string;
  name: string;
  items: FavoriteItem[];
  uses: number;
  last_used: string;
}

/** As many kinds of one item as a favorite keeps. */
export const maxChoices = 8;

/** A favorite's items from a logged entry and its add-ons; quick adds have no food, so they stay out. */
export function itemsFrom(entries: Pick<Entry, 'food' | 'grams' | 'amount'>[]): FavoriteItem[] {
  return entries
    .filter((e) => e.food && e.grams)
    .map((e) => ({
      choices: [{ food: e.food, grams: e.grams, ...(e.amount ? { amount: e.amount } : {}) }],
      pick: 0,
    }));
}

/** The kinds picked, of the items left in. */
export function picked(items: FavoriteItem[]): FavoriteChoice[] {
  return items.filter((it) => !it.off).map((it) => it.choices[it.pick]);
}

/** The calories of a favorite as picked; foods not loaded yet count as nothing. */
export function favoriteKcal(items: FavoriteItem[], foods: Map<string, Food>): number {
  return picked(items).reduce((s, c) => {
    const f = foods.get(c.food);
    return s + (f ? amountOf(f, c.grams).kcal : 0);
  }, 0);
}

/** What's in it, as picked: "Bread, Gouda, Egg". */
export function favoriteSummary(
  items: FavoriteItem[],
  foods: Map<string, Food>,
  lang?: FoodLanguage | '',
): string {
  return picked(items)
    .map((c) => foods.get(c.food))
    .filter((f): f is Food => !!f)
    .map((f) => foodName(f, lang))
    .join(', ');
}

/**
 * An item with a food picked in it at an amount: the same food gets the new
 * amount, another food becomes a new kind of it. The oldest kinds not picked
 * make room when it is full.
 */
export function pickKind(item: FavoriteItem, choice: FavoriteChoice): FavoriteItem {
  let choices = item.choices.filter((c) => c.food !== choice.food);
  const keep = item.choices[item.pick]?.food;
  while (choices.length >= maxChoices) {
    const drop = choices.findIndex((c) => c.food !== keep);
    choices = choices.filter((_, i) => i !== drop);
  }
  const at = item.choices.findIndex((c) => c.food === choice.food);
  choices.splice(at >= 0 ? Math.min(at, choices.length) : choices.length, 0, choice);
  return { choices, pick: choices.indexOf(choice), off: false };
}

/** Favorites whose name or foods have the words searched for. */
export function matchFavorites(
  list: Favorite[],
  q: string,
  foods: Map<string, Food>,
  lang?: FoodLanguage | '',
): Favorite[] {
  const words = q.toLowerCase().split(/\s+/).filter(Boolean);
  if (!words.length) return list;
  const text = (f: Favorite) =>
    [f.name, ...f.items.flatMap((it) => it.choices.map((c) => foods.get(c.food)))]
      .map((x) => (typeof x === 'string' ? x : x ? foodName(x, lang) : ''))
      .join(' ')
      .toLowerCase();
  return list.filter((f) => words.every((w) => text(f).includes(w)));
}

/** The usual ones first: most logged, then most recently. */
export function byUse(a: Favorite, b: Favorite): number {
  return b.uses - a.uses || (b.last_used || '').localeCompare(a.last_used || '');
}
