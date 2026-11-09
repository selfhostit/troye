import { describe, expect, it } from 'vitest';
import {
  amount,
  bar,
  chartTop,
  warningText,
  type InsightDay,
  type Report,
  type Totals,
} from './insights';

const totals = (kcal: number, fat = 0): Totals => ({
  kcal,
  protein: 0,
  carbs: 0,
  fat,
  fiber: 0,
  sugar: 0,
  salt: 0,
});
const day = (
  status: InsightDay['status'],
  food: Totals,
  estimate: Totals | null = null,
): InsightDay => ({
  date: '2026-11-08',
  status,
  food,
  estimate,
  budget: 2000,
});

describe('amount', () => {
  it('rounds to the nutrient', () => {
    expect(amount('kcal', 1849.6)).toBe(`${(1850).toLocaleString()} kcal`);
    expect(amount('salt', 4.56)).toBe('4.6 g');
  });
});

describe('bar', () => {
  it('stacks an estimate on what was logged', () => {
    expect(bar(day('partial', totals(400), totals(1900)), 'kcal')).toEqual({
      logged: 400,
      estimated: 1500,
    });
    expect(bar(day('missing', totals(0), totals(1900)), 'kcal')).toEqual({
      logged: 0,
      estimated: 1900,
    });
    expect(bar(day('logged', totals(2100)), 'kcal')).toEqual({ logged: 2100, estimated: 0 });
  });
});

describe('warningText', () => {
  it('names the goal it is held to', () => {
    expect(warningText({ nutrient: 'fat', kind: 'over', average: 82.4, target: 65 })).toBe(
      'Fat averaged 82 g a day, over your 65 g target.',
    );
    expect(warningText({ nutrient: 'fiber', kind: 'under', average: 14, target: 25 })).toBe(
      'Fiber averaged 14 g a day, under the 25 g guideline.',
    );
  });
});

describe('chartTop', () => {
  it('leaves room above the highest value', () => {
    const r = {
      days: [day('logged', totals(2500, 90)), day('missing', totals(0), totals(1800, 60))],
      rolling: { fat: [null, 75] },
      goals: { fat: { target: 65, min: 0, max: 71.5 } },
    } as unknown as Report;
    expect(chartTop(r, 'fat')).toBeCloseTo(99);
    expect(
      chartTop({ ...r, rolling: { kcal: [null, null] } } as unknown as Report, 'kcal'),
    ).toBeCloseTo(2750);
  });
});
