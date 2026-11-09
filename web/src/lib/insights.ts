import { pb } from './pb';

// What GET /api/app/insights returns (backend/insights): the last days with
// missing ones estimated, 7-day rolling averages and the week's warnings,
// plus what the nutrition screen works out from it to draw.

export type Nutrient = 'kcal' | 'protein' | 'carbs' | 'fat' | 'fiber' | 'sugar' | 'salt';
export type Totals = Record<Nutrient, number>;
export type DayStatus = 'logged' | 'partial' | 'missing' | 'today';

export interface InsightDay {
  date: string;
  status: DayStatus;
  food: Totals;
  /** what a missing or partial day likely held; null without enough history */
  estimate: Totals | null;
  budget: number;
}

export interface Goal {
  target: number;
  /** a warning fires below min or above max; 0 means no bound */
  min: number;
  max: number;
}

export interface InsightWeek {
  from: string;
  to: string;
  logged: number;
  estimated: number;
  missing: number;
  average: Totals;
  budget: number;
}

export interface Warning {
  nutrient: Nutrient;
  kind: 'over' | 'under';
  average: number;
  target: number;
}

export interface Report {
  today: string;
  goals: Partial<Record<Nutrient, Goal>>;
  days: InsightDay[];
  rolling: Record<Nutrient, (number | null)[]>;
  week: InsightWeek;
  warnings: Warning[];
}

export const nutrients: { id: Nutrient; label: string; unit: 'kcal' | 'g'; decimals: number }[] = [
  { id: 'kcal', label: 'Calories', unit: 'kcal', decimals: 0 },
  { id: 'protein', label: 'Protein', unit: 'g', decimals: 0 },
  { id: 'carbs', label: 'Carbs', unit: 'g', decimals: 0 },
  { id: 'fat', label: 'Fat', unit: 'g', decimals: 0 },
  { id: 'fiber', label: 'Fiber', unit: 'g', decimals: 0 },
  { id: 'sugar', label: 'Sugar', unit: 'g', decimals: 0 },
  { id: 'salt', label: 'Salt', unit: 'g', decimals: 1 },
];

export const insightRanges = [
  { id: '2w', label: '2W', days: 14 },
  { id: '1m', label: '1M', days: 30 },
  { id: '3m', label: '3M', days: 91 },
] as const;
export type InsightRange = (typeof insightRanges)[number]['id'];

export function loadInsights(days: number): Promise<Report> {
  return pb.send<Report>(`/api/app/insights?days=${days}`, { requestKey: null });
}

const meta = (n: Nutrient) => nutrients.find((x) => x.id === n)!;

/** An amount with its unit, such as "1,850 kcal" or "4.5 g". */
export function amount(n: Nutrient, v: number): string {
  const m = meta(n);
  const p = 10 ** m.decimals;
  return `${(Math.round(v * p) / p).toLocaleString()} ${m.unit}`;
}

/** What to draw for a day: the logged amount, plus an estimate on top where one fills the gap. */
export function bar(d: InsightDay, n: Nutrient): { logged: number; estimated: number } {
  const logged = d.status === 'missing' ? 0 : d.food[n];
  const estimated = d.estimate ? Math.max(d.estimate[n] - logged, 0) : 0;
  return { logged, estimated };
}

/** A warning as a sentence, such as "Fat averaged 82 g a day, over your 65 g target." */
export function warningText(w: Warning): string {
  const m = meta(w.nutrient);
  const goal =
    w.nutrient === 'kcal'
      ? `your ${amount('kcal', w.target)} budget`
      : planned.includes(w.nutrient)
        ? `your ${amount(w.nutrient, w.target)} target`
        : `the ${amount(w.nutrient, w.target)} guideline`;
  return `${m.label} averaged ${amount(w.nutrient, w.average)} a day, ${w.kind} ${goal}.`;
}

// the plan sets these; the others follow general guidance (WHO, EFSA)
const planned: Nutrient[] = ['protein', 'carbs', 'fat'];

/** Advice for a warning, one short sentence. */
export function warningTip(w: Warning): string {
  const tips: Record<Nutrient, [over: string, under: string]> = {
    kcal: ['A few days ran over budget.', 'Eating far under budget is hard to keep up.'],
    protein: ['', 'Eggs, yoghurt, beans or fish at each meal help.'],
    carbs: ['Swap some bread, pasta or sweets for vegetables.', ''],
    fat: ['Watch oils, cheese, fried food and spreads.', ''],
    fiber: ['', 'Wholegrains, beans, fruit and vegetables add fiber.'],
    sugar: ['Sweet drinks and snacks are the usual source.', ''],
    salt: ['Bread, cheese, cured meat and ready meals carry most salt.', ''],
  };
  return tips[w.nutrient][w.kind === 'over' ? 0 : 1];
}

/** The chart's top: the highest bar, rolling value or target, with headroom. */
export function chartTop(r: Report, n: Nutrient): number {
  const values = r.days.flatMap((d, i) => {
    const b = bar(d, n);
    return [b.logged + b.estimated, r.rolling[n][i] ?? 0, n === 'kcal' ? d.budget : 0];
  });
  const goal = r.goals[n];
  return Math.max(...values, goal?.target ?? 0, 1) * 1.1;
}
