import type { Entry, Food, Meal } from './diary';
import { diary } from './diary.svelte';
import { byUse, type Favorite, type FavoriteItem } from './favorites';
import { pb } from './pb';
import { session } from './session.svelte';

// The person's favorites, the usual ones first, with the foods in them, which
// the screens need for names, units and calories.
class Favorites {
  list = $state<Favorite[]>([]);
  foods = $state(new Map<string, Food>());

  async load() {
    try {
      const list = await pb.collection('favorites').getFullList<Favorite>({
        filter: pb.filter('user = {:user}', { user: session.user!.id }),
        requestKey: null,
      });
      await this.loadFoods(list.flatMap((f) => f.items));
      this.list = list.sort(byUse);
    } catch {
      // favorites are a shortcut; search still works
    }
  }

  /** Loads the foods of these items that aren't loaded yet. */
  private async loadFoods(items: FavoriteItem[]) {
    const ids = [...new Set(items.flatMap((it) => it.choices.map((c) => c.food)))].filter(
      (id) => !this.foods.has(id),
    );
    if (!ids.length) return;
    const found: Food[] = [];
    for (let i = 0; i < ids.length; i += 50) {
      const part = ids.slice(i, i + 50);
      found.push(
        ...(await pb.collection('foods').getFullList<Food>({
          filter: part.map((id) => pb.filter('id = {:id}', { id })).join(' || '),
          requestKey: null,
        })),
      );
    }
    this.remember(found);
  }

  /** Keeps foods for the screens, such as one just picked as a new kind. */
  remember(foods: Food[]) {
    const next = new Map(this.foods);
    for (const f of foods) next.set(f.id, f);
    this.foods = next;
  }

  private put(f: Favorite) {
    this.list = [...this.list.filter((x) => x.id !== f.id), f].sort(byUse);
  }

  async create(name: string, items: FavoriteItem[]): Promise<Favorite> {
    const f = await pb
      .collection('favorites')
      .create<Favorite>({ user: session.user!.id, name, items });
    this.put(f);
    return f;
  }

  async update(fav: Favorite, changes: Partial<Pick<Favorite, 'name' | 'items'>>) {
    this.put(await pb.collection('favorites').update<Favorite>(fav.id, changes));
  }

  async remove(fav: Favorite) {
    await pb.collection('favorites').delete(fav.id);
    this.list = this.list.filter((x) => x.id !== fav.id);
  }

  /** Logs a favorite as picked this time, which the server keeps as the picks for next time. */
  async log(fav: Favorite, items: FavoriteItem[], meal: Meal): Promise<Entry[]> {
    const entries = await pb.send<Entry[]>(`/api/app/favorites/${fav.id}/log`, {
      method: 'POST',
      body: { date: diary.day, meal, items },
      requestKey: null,
    });
    diary.took(entries);
    this.put({
      ...fav,
      items,
      uses: fav.uses + 1,
      last_used: new Date().toISOString(),
    });
    return entries;
  }
}

export const favorites = new Favorites();
