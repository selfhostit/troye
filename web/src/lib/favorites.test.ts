import { describe, expect, it } from 'vitest';
import type { Food } from './diary';
import {
  byUse,
  favoriteKcal,
  favoriteSummary,
  itemsFrom,
  matchFavorites,
  maxChoices,
  pickKind,
  type Favorite,
  type FavoriteItem,
} from './favorites';

const food = (id: string, kcal: number, extra: Partial<Food> = {}) =>
  ({ id, name: id, kcal, protein: 0, carbs: 0, fat: 0, ...extra }) as Food;
const foods = new Map(
  [food('bread', 250), food('gouda', 356, { name_nl: 'Goudse kaas' }), food('egg', 155)].map(
    (f) => [f.id, f],
  ),
);
const toast: FavoriteItem[] = [
  { choices: [{ food: 'bread', grams: 40, amount: '1 slice' }], pick: 0 },
  { choices: [{ food: 'gouda', grams: 20 }], pick: 0 },
  { choices: [{ food: 'egg', grams: 50 }], pick: 0, off: true },
];
const fav = (name: string, extra: Partial<Favorite> = {}) =>
  ({ id: name, name, items: toast, uses: 0, last_used: '', ...extra }) as Favorite;

describe('favorites', () => {
  it('is made from an entry and its add-ons, leaving out quick adds', () => {
    expect(
      itemsFrom([
        { food: 'bread', grams: 40, amount: '1 slice' },
        { food: 'gouda', grams: 20, amount: '' },
        { food: '', grams: 0 },
      ]),
    ).toEqual([
      { choices: [{ food: 'bread', grams: 40, amount: '1 slice' }], pick: 0 },
      { choices: [{ food: 'gouda', grams: 20 }], pick: 0 },
    ]);
  });

  it('adds up and lists what is picked, not what is left out', () => {
    expect(favoriteKcal(toast, foods)).toBe(100 + 71);
    expect(favoriteSummary(toast, foods, 'nl')).toBe('bread, Goudse kaas');
  });

  it('keeps a new kind of an item and picks it', () => {
    const egg = toast[2];
    const fried = pickKind(egg, { food: 'fried egg', grams: 60 });
    expect(fried).toEqual({
      choices: [
        { food: 'egg', grams: 50 },
        { food: 'fried egg', grams: 60 },
      ],
      pick: 1,
      off: false,
    });
    // the same food again changes its amount in its place
    expect(pickKind(fried, { food: 'egg', grams: 100 })).toEqual({
      choices: [
        { food: 'egg', grams: 100 },
        { food: 'fried egg', grams: 60 },
      ],
      pick: 0,
      off: false,
    });
  });

  it('drops the oldest kind not picked when an item is full', () => {
    const full: FavoriteItem = {
      choices: Array.from({ length: maxChoices }, (_, i) => ({ food: `f${i}`, grams: 10 })),
      pick: 0,
    };
    const next = pickKind(full, { food: 'new', grams: 10 });
    expect(next.choices).toHaveLength(maxChoices);
    expect(next.choices.map((c) => c.food)).toContain('f0');
    expect(next.choices.map((c) => c.food)).not.toContain('f1');
    expect(next.choices[next.pick].food).toBe('new');
  });

  it('finds favorites by name or by what is in them', () => {
    const list = [fav('Breakfast toast'), fav('Lunch', { items: [toast[0]] })];
    expect(matchFavorites(list, 'toast', foods).map((f) => f.name)).toEqual(['Breakfast toast']);
    expect(matchFavorites(list, 'egg', foods).map((f) => f.name)).toEqual(['Breakfast toast']);
    expect(matchFavorites(list, 'bread', foods)).toHaveLength(2);
  });

  it('puts the usual ones first', () => {
    const list = [
      fav('a', { uses: 1, last_used: '2026-11-01' }),
      fav('b', { uses: 3 }),
      fav('c', { uses: 1, last_used: '2026-11-09' }),
    ];
    expect(list.sort(byUse).map((f) => f.name)).toEqual(['b', 'c', 'a']);
  });
});
