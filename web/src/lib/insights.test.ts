import { describe, expect, it } from 'vitest';
import {
  amount,
  bar,
  chartTop,
  offerBudget,
  trendText,
  warningText,
  weekdayText,
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
  weight_kg: 0,
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

describe('patterns', () => {
  it('words a trend', () => {
    expect(trendText({ nutrient: 'fat', per_week: 7, average: 70 })).toBe('Fat up 7 g a week');
    expect(trendText({ nutrient: 'fiber', per_week: -2.4, average: 20 })).toBe(
      'Fiber down 2 g a week',
    );
  });
  it('words the weekday pattern', () => {
    expect(weekdayText({ diffs: [], weekday: 'Saturday', diff: 600 })).toBe(
      'Saturdays run about 600 kcal over your average day.',
    );
    expect(weekdayText({ diffs: [], weekday: '', diff: 0 })).toMatch(/even/);
  });
  it('offers the measured budget once it differs', () => {
    const burn = {
      from: '',
      to: '',
      tdee: 2550,
      intake_kcal: 2000,
      weight_per_week: -0.5,
      days: 28,
      weighins: 7,
    };
    const advice = { formula_tdee: 2075, measured_tdee: 0, suggested: 2000, current: 1520 };
    expect(offerBudget(advice, burn)).toBe(true);
    expect(offerBudget({ ...advice, measured_tdee: 2550 }, burn)).toBe(false);
    expect(offerBudget({ ...advice, suggested: 1540 }, burn)).toBe(false);
    expect(offerBudget(null, burn)).toBe(false);
  });
});
