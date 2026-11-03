import type { Food } from './diary';

// Amounts the way people think of them: a tablespoon of hummus, a slice of
// bread, a glass of milk, a handful of nuts. Nobody weighs their hummus, so a
// food offers its own serving and the household units that suit it, and grams
// stay one tap away.

/** A unit of a food, such as a tablespoon of 15 g. */
export interface Unit {
  /** the unit's name for one and for more, such as "slice" and "slices" */
  one: string;
  many: string;
  grams: number;
}

const unit = (one: string, grams: number, many = one + 's'): Unit => ({ one, many, grams });
const tbsp = (grams = 15) => unit('tbsp', grams, 'tbsp');
const tsp = (grams = 5) => unit('tsp', grams, 'tsp');
const piece = (grams: number) => unit('piece', grams);

// Typical weights, rounded; the first rule a food's name matches wins, so the
// specific ones (peanut butter) come before the general ones (butter). Names
// are matched in English, Dutch and French, as the food tables have them.
const rules: [RegExp, Unit[]][] = [
  [
    /hummus|houmous|peanut butter|pindakaas|beurre de cacahu|tahini|tahin|nutella|choco(late)? spread|chocopasta|guacamole|tzatziki|pesto|cream cheese|smeerkaas|fromage frais|cottage/,
    [tbsp(), tsp()],
  ],
  [
    /mayo|ketchup|mustard|mosterd|moutarde|sauce|\bsaus\b|dressing|vinaigrette|soy|soja|sambal|sriracha|salsa/,
    [tbsp(), tsp()],
  ],
  [/jam|confiture|honey|honing|miel|syrup|siroop|sirop|stroop/, [tsp(7), tbsp(21)]],
  [/\boil\b|\bolie\b|huile/, [tbsp(14), tsp(5)]],
  [/butter|boter|beurre|margarine|ghee/, [tsp(5), tbsp(14)]],
  [/sugar|suiker|sucre|cocoa|cacao|flour|bloem|farine/, [tsp(4), tbsp(12)]],
  [/espresso|ristretto/, [unit('shot', 30)]],
  [
    /coffee|koffie|café|cafe|cappuccino|latte|\btea\b|thee|\bthé/,
    [unit('cup', 200), unit('mug', 300)],
  ],
  [/milk|melk|lait/, [unit('splash', 30, 'splashes'), unit('glass', 250, 'glasses'), tbsp()]],
  [/wine|wijn|\bvin\b/, [unit('glass', 150, 'glasses')]],
  [/beer|bier|bière|biere|pils/, [unit('glass', 250, 'glasses'), unit('bottle', 330)]],
  [
    /juice|\bsap\b|\bjus\b|soda|\bcola\b|lemonade|limonade|smoothie|kefir|drink|drank|boisson|\bwater\b/,
    [unit('glass', 250, 'glasses'), unit('can', 330)],
  ],
  [/soup|soep|soupe|ramen|pho\b|broth|bouillon/, [unit('bowl', 300), unit('cup', 240)]],
  [
    /yogh?urt|yaourt|skyr|kwark|quark|pudding|custard|vla/,
    [unit('pot', 125), tbsp(), unit('bowl', 200)],
  ],
  // things people count: meatballs, nuggets, croquettes, dumplings
  [/meatball|balletje|polpett/, [unit('meatball', 30)]],
  [/gehaktbal|boulette/, [piece(100)]],
  [/nugget/, [unit('nugget', 18)]],
  [/bitterbal/, [piece(20)]],
  [/kroket|croquet|frikandel/, [piece(70)]],
  [/fish (stick|finger)|fishstick|visstick|poisson pané/, [unit('stick', 28)]],
  [/falafel/, [piece(20)]],
  [/dumpling|gyoza|dim sum|wonton|bao\b/, [piece(25)]],
  [/loempia/, [piece(150)]],
  [/spring roll|nem\b/, [piece(60)]],
  [/samosa/, [piece(50)]],
  [/sushi|maki|nigiri/, [piece(30)]],
  [/\bwings?\b|kippenvleugel|aile de poulet/, [unit('wing', 35)]],
  [/drumstick(?! leaves| pods)|kippenbout|pilon/, [unit('drumstick', 75)]],
  [/chicken breast|kipfilet|blanc de poulet/, [piece(150), unit('slice', 20)]],
  [
    /shrimp|prawn|garnaal|garnalen|crevette|scampi/,
    [unit('shrimp', 8, 'shrimp'), unit('handful', 60)],
  ],
  [/scallop|coquille|sint-jakobsschelp/, [piece(20)]],
  [/\bolives?\b|olijf|olijven/, [unit('olive', 4)]],
  [/pancake|pannenkoek|crêpe|crepe|donut|doughnut|croissant/, [piece(60)]],
  [/pain au chocolat|chocoladebroodje/, [piece(70)]],
  [/rice cake|rijstwafel|galette de riz/, [unit('cake', 9)]],
  [/stroopwafel/, [piece(30)]],
  [/waffle|wafel|gaufre/, [piece(75)]],
  [/muffin/, [piece(60)]],
  [/bagel/, [piece(90)]],
  [/tortilla|\bwraps?\b|\bpita\b|naan/, [piece(60)]],
  [
    /\boats?\b|haver|avoine|muesli|granola|cereal|cornflakes|porridge|pap\b/,
    [unit('bowl', 50), tbsp(10)],
  ],
  [/rice|rijst|\briz\b|quinoa|couscous|bulgur/, [unit('portion', 180), unit('cup', 160), tbsp(15)]],
  [
    /pasta|spaghetti|penne|macaroni|noodle|nouille|noedel|tagliatelle|fusilli|ravioli|tortellini|lasagn/,
    [unit('plate', 250), unit('cup', 140)],
  ],
  [
    /\bnuts?\b|walnut|hazelnut|noot|noten|noix|almond|amandel|amande|cashew|pistach|peanut|pinda|\bseeds?\b|zaad|zaden|graine|raisin|rozijn/,
    [unit('handful', 30), tbsp(10)],
  ],
  [/crisps|chips|popcorn|pretzel/, [unit('handful', 20), unit('bag', 45)]],
  [/cracker|beschuit|rusk|biscotte/, [piece(10)]],
  [/bell pepper|peppers?, bell|poivron/, [piece(160)]],
  // fruit, by the piece and in slices
  [
    /cherry tomato|cherrytomaat|snoeptomaat|tomates? cerises?/,
    [unit('cherry tomato', 15, 'cherry tomatoes')],
  ],
  [/tomato|tomaat|tomate/, [piece(120), unit('cherry tomato', 15, 'cherry tomatoes')]],
  [/strawberr|aardbei|fraise/, [unit('strawberry', 12, 'strawberries'), unit('handful', 70)]],
  [/cherr|\bkersen\b|\bkers\b|cerise/, [unit('handful', 70), unit('cherry', 8, 'cherries')]],
  [/grape(?!fruit)|druif|druiven|raisin/, [unit('handful', 70), unit('grape', 5)]],
  [/berr|\bbessen\b|baie/, [unit('handful', 70), unit('cup', 150)]],
  [/\bdates?\b|dadel|datte/, [unit('date', 10)]],
  [/\bfigs?\b|vijg|figue/, [piece(50)]],
  [/pineapple|\bananas\b/, [unit('slice', 80), unit('cup', 165)]],
  [/watermelon|watermeloen|pastèque/, [unit('slice', 280), unit('cup', 150)]],
  [/melon|meloen|cantaloupe|honeydew/, [unit('slice', 120), unit('cup', 160)]],
  [/mango|mangue/, [piece(200), unit('slice', 35)]],
  [/grapefruit|pompelmoes|pamplemousse/, [unit('half', 120, 'halves'), piece(240)]],
  [/\bapples?\b|\bappels?\b|\bpommes?\b(?! de terre)/, [piece(180), unit('slice', 20)]],
  [/\bpears?\b|\bpeer\b|poire/, [piece(170), unit('slice', 20)]],
  [/orange|sinaas/, [piece(150), unit('segment', 15)]],
  [/mandarin|clementine|tangerine/, [piece(70), unit('segment', 8)]],
  [/lemon|citroen|citron|\blimes?\b|limoen/, [unit('wedge', 8), piece(60)]],
  [/kiwi/, [piece(75)]],
  [/peach|perzik|pêche|nectarine/, [piece(150), unit('slice', 20)]],
  [/plum|pruim|prune|apricot|abrikoos|abricot/, [piece(50)]],
  [/banana|banaan|banane/, [piece(120)]],
  [/avocado|avocat/, [unit('half', 75, 'halves'), piece(150)]],
  [/bread|brood|pain|toast|baguette|stokbrood|sandwich|boterham/, [unit('slice', 35)]],
  [/pizza/, [unit('slice', 100)]],
  [/cake|taart|tarte|gâteau|gateau|\bpie\b|brownie/, [unit('slice', 80)]],
  [/cookie|biscuit|koek|speculoos/, [piece(15)]],
  [/praline|bonbon|truffle|truffe/, [piece(12)]],
  [/chocolate|chocolade|chocolat/, [unit('square', 5), unit('bar', 100)]],
  [/mozzarella/, [unit('ball', 125), unit('slice', 20)]],
  [/cheese|kaas|fromage|parmesan|feta/, [unit('slice', 20), tbsp(7)]],
  [/\bham\b|salami|bacon|spek|jambon|chorizo|turkey breast|kalkoen/, [unit('slice', 15)]],
  [/sausage|worst|saucisse|bratwurst|frankfurter|knakworst|chipolata/, [unit('sausage', 60)]],
  [/\beggs?\b|\bei\b|eieren|œuf|oeuf/, [unit('egg', 50)]],
  [/carrot|wortel|carotte/, [piece(60)]],
  [/potato|aardappel|pomme de terre/, [piece(150)]],
  [/fries|frites|friet|patat/, [unit('portion', 150), unit('handful', 50)]],
  [/onion|ui\b|uien|oignon/, [piece(110)]],
  [/cucumber|komkommer|concombre/, [unit('slice', 7), piece(300)]],
  [/zucchini|courgette/, [piece(200)]],
  [/mushroom|champignon/, [unit('mushroom', 18), unit('handful', 50)]],
  [/broccoli|cauliflower|bloemkool|chou-fleur/, [unit('floret', 12), unit('portion', 150)]],
  [
    /salad|salade|\bsla\b|spinach|spinazie|épinard|lettuce/,
    [unit('bowl', 80), unit('handful', 30)],
  ],
];

/**
 * The household units that suit a food, from its name. The food itself, before
 * the first comma, is tried first: "Croissants, butter" is a croissant and
 * "Crackers, milk" a cracker.
 */
export function householdUnits(name: string): Unit[] {
  const n = name.toLowerCase();
  const head = n.split(',')[0];
  return (rules.find(([re]) => re.test(head)) ?? rules.find(([re]) => re.test(n)))?.[1] ?? [];
}

// a serving named by a measure rather than by the food: "1 cup, chopped", "1 oz"
const measures = new Set(['cup', 'oz', 'tbsp', 'tsp', 'fl', 'ml', 'g', 'gram', 'grams', 'serving']);
// a serving that is one whole food of a size: "1 medium (3" dia)"
const sizes = new Set([
  'small',
  'medium',
  'large',
  'extra',
  'jumbo',
  'super',
  'whole',
  'fruit',
  'each',
  'item',
]);

/**
 * A food's serving as a unit: "1 slice" becomes slice and slices, "1 date,
 * pitted" a date, and "1 medium (3" dia)" a piece.
 */
export function servingUnit(food: Pick<Food, 'serving_g' | 'serving_label'>): Unit | null {
  if (!(food.serving_g > 0)) return null;
  const label = food.serving_label.trim();
  // one word, alone or before a comma, a bracket or an amount: "1 link 3 oz"
  const simple = label.match(/^1 ([a-z]+)(?:$|,|\s*\(|\s+\d)/i)?.[1]?.toLowerCase();
  if (simple && sizes.has(simple)) return piece(food.serving_g);
  if (simple && !/s$/.test(simple)) return unit(simple, food.serving_g, plural(simple));
  // "2 slices (60 g)" or no label: counted as servings of that
  const what = label || `${food.serving_g} g`;
  return { one: `serving (${what})`, many: `servings (${what})`, grams: food.serving_g };
}

function plural(word: string): string {
  if (/(s|sh|ch|x)$/.test(word)) return word + 'es';
  if (/[^aeiou]y$/.test(word)) return word.slice(0, -1) + 'ies';
  return word + 's';
}

/**
 * The units to offer for a food: its own serving first, then household units
 * of other names. A serving that is only a measure, such as a cup of chopped
 * apple, comes after the household units, so an apple starts as a piece.
 */
export function unitsOf(food: Pick<Food, 'name' | 'serving_g' | 'serving_label'>): Unit[] {
  let serving = servingUnit(food);
  const household = householdUnits(food.name);
  if (!serving) return household;
  // "1 large" of an egg is an egg
  if (serving.one === 'piece' && household[0]?.one === 'egg')
    serving = { ...household[0], grams: serving.grams };
  // a household unit of the same name takes the serving's weight, in its place
  if (household.some((u) => u.one === serving.one))
    return household.map((u) => (u.one === serving.one ? serving : u));
  const more = household.filter((u) => u.grams !== serving.grams);
  return measures.has(serving.one.split(' ')[0]) && more.length
    ? [...more, serving]
    : [serving, ...more];
}

/** "1 tbsp", "2.5 slices". */
export function countText(u: Unit, n: number): string {
  return `${n} ${n === 1 ? u.one : u.many}`;
}

/** Reads an amount such as "2 tbsp" back into one of the units, or null. */
export function readAmount(amount: string, units: Unit[]): { unit: Unit; count: number } | null {
  const m = amount.trim().match(/^(\d+(?:\.\d+)?) (.+)$/);
  if (!m) return null;
  const found = units.find((u) => u.one === m[2] || u.many === m[2]);
  return found ? { unit: found, count: parseFloat(m[1]) } : null;
}

/** A count's grams, to a tenth of a gram. */
export function gramsOf(u: Unit, n: number): number {
  return Math.round(u.grams * n * 10) / 10;
}

/**
 * Where the amount starts for a food: what was logged last time (in units when it
 * comes to a whole or half number of them), else one of its
 * serving or first household unit, else 100 g.
 */
export function startAmount(
  food: Pick<Food, 'name' | 'serving_g' | 'serving_label'>,
  last?: { grams: number; amount: string },
): { grams: number; amount: string } {
  if (last?.grams)
    return { grams: last.grams, amount: last.amount || guessAmount(food, last.grams) };
  const u = unitsOf(food)[0];
  return u ? { grams: u.grams, amount: countText(u, 1) } : { grams: 100, amount: '' };
}

/** An amount in grams as a count of the food's first unit when it is a whole or half number of them, else "". */
export function guessAmount(
  food: Pick<Food, 'name' | 'serving_g' | 'serving_label'>,
  grams: number,
): string {
  const u = unitsOf(food)[0];
  if (!u || !grams) return '';
  const n = grams / u.grams;
  return n >= 0.5 && Math.abs(n * 2 - Math.round(n * 2)) < 0.01
    ? countText(u, Math.round(n * 2) / 2)
    : '';
}
