import { describe, expect, it } from 'vitest';
import type { Food } from './diary';
import {
  countText,
  guessAmount,
  householdUnits,
  readAmount,
  servingUnit,
  startAmount,
  unitsOf,
} from './portions';

const food = (name: string, extra: Partial<Food> = {}) =>
  ({ name, serving_g: 0, serving_label: '', ...extra }) as Food;

describe('household units', () => {
  it('suits the food', () => {
    expect(householdUnits('Hummus, plain').map((u) => u.one)).toEqual(['tbsp', 'tsp']);
    expect(householdUnits('Pindakaas')[0].one).toBe('tbsp');
    expect(householdUnits('Milk, semi-skimmed')[0].one).toBe('splash');
    expect(householdUnits('Wholemeal bread')[0]).toEqual({
      one: 'slice',
      many: 'slices',
      grams: 35,
    });
    expect(householdUnits('Almonds')[0].one).toBe('handful');
    expect(householdUnits('Banana')[0].one).toBe('piece');
    expect(householdUnits('Zzz')).toEqual([]);
  });

  it("doesn't trip over words inside words", () => {
    expect(householdUnits('Egg, boiled')[0].one).toBe('egg');
    expect(householdUnits('Peanut butter')[0].one).toBe('tbsp');
    expect(householdUnits('Oat milk')[0].one).toBe('splash');
    expect(householdUnits('Chocolate, dark')[0].one).toBe('square');
    expect(householdUnits('Pineapple, raw')[0].one).toBe('slice');
    expect(householdUnits('Bananas, raw')[0].one).toBe('piece');
    expect(householdUnits('Sausage, Italian')[0].one).toBe('sausage');
    expect(householdUnits("APPLEBEE'S, Double Crunch Shrimp")[0].one).toBe('shrimp');
    expect(householdUnits('Sinaasappel')[0]).toEqual({ one: 'piece', many: 'pieces', grams: 150 });
    expect(householdUnits('Grapes, red, seedless, raw')[0].one).toBe('handful');
    expect(householdUnits('Zzz')).toEqual([]);
  });

  it('looks at the food before its details', () => {
    expect(householdUnits('Croissants, butter')[0].one).toBe('piece');
    expect(householdUnits('Crackers, milk')[0].one).toBe('piece');
    expect(householdUnits('Pancakes, buttermilk')[0].one).toBe('piece');
  });

  it('counts what people count and slices fruit', () => {
    expect(householdUnits('Meatballs, frozen, Italian style')[0]).toEqual({
      one: 'meatball',
      many: 'meatballs',
      grams: 30,
    });
    expect(householdUnits('Gehaktballetjes')[0].one).toBe('meatball');
    expect(householdUnits('Chicken, nuggets')[0].one).toBe('nugget');
    expect(householdUnits('Strawberries, raw')[0].many).toBe('strawberries');
    for (const fruit of [
      'Apples, raw, with skin',
      'Pears, raw',
      'Mangos, raw',
      'Peaches, yellow, raw',
    ]) {
      expect(householdUnits(fruit).map((u) => u.one)).toEqual(['piece', 'slice']);
    }
    expect(householdUnits('Watermelon, raw')[0].one).toBe('slice');
  });
});

describe('units of a food', () => {
  it('puts its own serving first', () => {
    const bread = food('Bread', { serving_g: 30, serving_label: '1 slice' });
    expect(unitsOf(bread)).toEqual([{ one: 'slice', many: 'slices', grams: 30 }]);
    const hummus = food('Hummus', { serving_g: 50, serving_label: '1 pot' });
    expect(unitsOf(hummus).map((u) => u.one)).toEqual(['pot', 'tbsp', 'tsp']);
  });

  it('puts a measure after the household units', () => {
    const apple = food('Apples, raw', { serving_g: 125, serving_label: '1 cup, chopped' });
    expect(unitsOf(apple).map((u) => u.one)).toEqual(['piece', 'slice', 'cup']);
    const meatballs = food('Meatballs, frozen', { serving_g: 85, serving_label: '3 oz' });
    expect(unitsOf(meatballs)[0].one).toBe('meatball');
    // a serving of the same name keeps its place and takes its weight
    const coffee = food('Coffee, brewed', { serving_g: 240, serving_label: '1 cup' });
    expect(unitsOf(coffee)).toEqual([
      { one: 'cup', many: 'cups', grams: 240 },
      { one: 'mug', many: 'mugs', grams: 300 },
    ]);
  });

  it('reads a whole food of a size as a piece', () => {
    const apple = food('Apples, raw', { serving_g: 182, serving_label: '1 medium (3" dia)' });
    expect(unitsOf(apple)[0]).toEqual({ one: 'piece', many: 'pieces', grams: 182 });
    const egg = food('Egg, white, raw', { serving_g: 33, serving_label: '1 large' });
    expect(unitsOf(egg)).toEqual([{ one: 'egg', many: 'eggs', grams: 33 }]);
    const dates = food('Dates, medjool', { serving_g: 24, serving_label: '1 date, pitted' });
    expect(unitsOf(dates)).toEqual([{ one: 'date', many: 'dates', grams: 24 }]);
  });

  it('counts a serving without a simple name as servings', () => {
    expect(servingUnit(food('x', { serving_g: 60, serving_label: '2 slices' }))?.many).toBe(
      'servings (2 slices)',
    );
    expect(servingUnit(food('x', { serving_g: 0, serving_label: '1 slice' }))).toBeNull();
    expect(servingUnit(food('x', { serving_g: 25, serving_label: '1 Biscuit' }))?.many).toBe(
      'biscuits',
    );
  });

  it('writes and reads amounts', () => {
    const units = unitsOf(food('Hummus'));
    expect(countText(units[0], 1)).toBe('1 tbsp');
    expect(readAmount('2.5 tbsp', units)).toEqual({ unit: units[0], count: 2.5 });
    expect(readAmount('2 Tablespoons', units)).toBeNull();
    expect(readAmount('', units)).toBeNull();
  });

  it('starts from the last amount, else one unit, else 100 g', () => {
    expect(startAmount(food('Hummus'), { grams: 45, amount: '3 tbsp' })).toEqual({
      grams: 45,
      amount: '3 tbsp',
    });
    expect(startAmount(food('Hummus'), { grams: 30, amount: '' }).amount).toBe('2 tbsp');
    expect(startAmount(food('Hummus'))).toEqual({ grams: 15, amount: '1 tbsp' });
    expect(startAmount(food('Zzz'))).toEqual({ grams: 100, amount: '' });
  });

  it('guesses a count for grams logged before units', () => {
    const bread = food('Bread', { serving_g: 35, serving_label: '1 slice' });
    expect(guessAmount(bread, 70)).toBe('2 slices');
    expect(guessAmount(bread, 52.5)).toBe('1.5 slices');
    expect(guessAmount(bread, 50)).toBe('');
  });
});

describe('chicken breast', () => {
  it('starts every chicken breast at a piece, whatever its USDA serving', () => {
    for (const serving of [
      { serving_g: 85, serving_label: '3 oz' },
      { serving_g: 140, serving_label: '1 cup, chopped or diced' },
      { serving_g: 52, serving_label: '1 unit (yield from 1 lb ready-to-cook chicken)' },
      { serving_g: 0, serving_label: '' },
    ]) {
      expect(startAmount(food('Chicken breast, roasted', serving))).toEqual({
        grams: 150,
        amount: '1 piece',
      });
    }
  });
});
