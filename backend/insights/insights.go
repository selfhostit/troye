// Package insights reads a person's diary for patterns: which days are
// missing or only half logged and what they likely held, 7-day rolling
// averages per nutrient, and warnings when the last week's average is off
// target. It only does arithmetic on a daily.Series: loading the days and
// telling the person are the caller's job.
package insights

import (
	"math"
	"slices"
	"time"

	"github.com/selfhostit/troye/daily"
	"github.com/selfhostit/troye/stats"
)

// Nutrient names one of the totals in daily.Nutrients.
type Nutrient string

const (
	Kcal    Nutrient = "kcal"
	Protein Nutrient = "protein"
	Carbs   Nutrient = "carbs"
	Fat     Nutrient = "fat"
	Fiber   Nutrient = "fiber"
	Sugar   Nutrient = "sugar"
	Salt    Nutrient = "salt"
)

// Nutrients in the order they are shown.
var Nutrients = []Nutrient{Kcal, Protein, Carbs, Fat, Fiber, Sugar, Salt}

// Of is the nutrient's total in n.
func (k Nutrient) Of(n daily.Nutrients) float64 {
	switch k {
	case Kcal:
		return n.Kcal
	case Protein:
		return n.Protein
	case Carbs:
		return n.Carbs
	case Fat:
		return n.Fat
	case Fiber:
		return n.Fiber
	case Sugar:
		return n.Sugar
	case Salt:
		return n.Salt
	}
	return 0
}

func (k Nutrient) set(n *daily.Nutrients, v float64) {
	switch k {
	case Kcal:
		n.Kcal = v
	case Protein:
		n.Protein = v
	case Carbs:
		n.Carbs = v
	case Fat:
		n.Fat = v
	case Fiber:
		n.Fiber = v
	case Sugar:
		n.Sugar = v
	case Salt:
		n.Salt = v
	}
}

// Status says how complete a day's diary is.
type Status string

const (
	// Logged is a day that looks fully logged.
	Logged Status = "logged"
	// Partial is a past day with one meal or less and under half the budget.
	Partial Status = "partial"
	// Missing is a past day with nothing logged.
	Missing Status = "missing"
	// Today is still going, so it is never judged.
	Today Status = "today"
)

// Day is one day of the report.
type Day struct {
	Date   string          `json:"date"`
	Status Status          `json:"status"`
	Food   daily.Nutrients `json:"food"`
	// Estimate is what a missing or partial day likely held, from the
	// person's own complete days; nil without enough of them.
	Estimate *daily.Nutrients `json:"estimate"`
	Budget   int              `json:"budget"`
	// WeightKg is the day's weigh-in, 0 without one.
	WeightKg float64 `json:"weight_kg"`
}

// value is the day's totals for averaging: the estimate when there is one,
// the food when the day is complete, and false otherwise.
func (d Day) value() (daily.Nutrients, bool) {
	switch {
	case d.Estimate != nil:
		return *d.Estimate, true
	case d.Status == Logged:
		return d.Food, true
	}
	return daily.Nutrients{}, false
}

// Goal is what a nutrient is held to. A warning fires when the 7-day average
// goes above Max or below Min; zero means no bound that way.
type Goal struct {
	Target float64 `json:"target"`
	Min    float64 `json:"min"`
	Max    float64 `json:"max"`
}

// Guidance for the nutrients the plan doesn't set, a day, for adults: free
// sugars under 10% of energy (about 50 g), salt under 5 g (both WHO), and at
// least 25 g of fiber (EFSA).
const (
	sugarG = 50
	saltG  = 5
	fiberG = 25
)

// Leeway before a warning: an average more than 10% over a ceiling or 20%
// under a floor, so an ordinary week doesn't nag.
const (
	over  = 1.1
	under = 0.8
)

func ceiling(target float64) Goal { return Goal{Target: target, Max: target * over} }
func floor(target float64) Goal   { return Goal{Target: target, Min: target * under} }

// Goals for each nutrient from a plan's targets and an average daily budget
// (exercise included). Nutrients the plan leaves at zero have no goal.
func Goals(t daily.Targets, budget float64) map[Nutrient]Goal {
	g := map[Nutrient]Goal{
		Fiber: floor(fiberG),
		Sugar: ceiling(sugarG),
		Salt:  ceiling(saltG),
	}
	if budget > 0 {
		g[Kcal] = Goal{Target: budget, Min: budget * under, Max: budget * over}
	}
	if t.ProteinG > 0 {
		g[Protein] = floor(float64(t.ProteinG))
	}
	if t.CarbsG > 0 {
		g[Carbs] = ceiling(float64(t.CarbsG))
	}
	if t.FatG > 0 {
		g[Fat] = ceiling(float64(t.FatG))
	}
	return g
}

// Week sums up the last seven days before today.
type Week struct {
	From string `json:"from"`
	To   string `json:"to"`
	// Logged, Estimated and Missing count the days: complete, filled in
	// with an estimate, and without either.
	Logged    int `json:"logged"`
	Estimated int `json:"estimated"`
	Missing   int `json:"missing"`
	// Average is over the logged and estimated days.
	Average daily.Nutrients `json:"average"`
	// Budget is the average budget over the same days.
	Budget float64 `json:"budget"`
}

// Kind says which way a warning goes.
type Kind string

const (
	Over  Kind = "over"
	Under Kind = "under"
)

// Warning is a nutrient whose average over the last week is off its goal.
type Warning struct {
	Nutrient Nutrient `json:"nutrient"`
	Kind     Kind     `json:"kind"`
	Average  float64  `json:"average"`
	Target   float64  `json:"target"`
}

// Report is what the progress screen shows.
type Report struct {
	Today string `json:"today"`
	// Goals use the plan's own calorie budget, without exercise.
	Goals map[Nutrient]Goal `json:"goals"`
	Days  []Day             `json:"days"`
	// Rolling holds, for each nutrient, the 7-day average ending on each day
	// of Days; null where fewer than four of those days have a value, and on today.
	Rolling  map[Nutrient][]*float64 `json:"rolling"`
	Week     Week                    `json:"week"`
	Warnings []Warning               `json:"warnings"`
	// Trends, Weekdays and Burn look further back; see patterns.go.
	Trends   []Trend   `json:"trends"`
	Weekdays *Weekdays `json:"weekdays"`
	Burn     *Burn     `json:"burn"`
}

// History is how many days before the first one shown a series needs, for
// estimates and the rolling average.
const History = 8 * 7

const (
	rollingDays = 7
	// fewest values for an average to mean something
	minValues = 4
	// fewest days actually logged before warning about a week
	minLogged = 3
)

// Analyze reads a series ending today, which should start History days
// before the first of the `days` days to show.
func Analyze(s daily.Series, today string, days int) Report {
	all := classify(s, today)
	estimate(all)

	first := max(len(all)-days, 0)
	r := Report{
		Today:    today,
		Goals:    Goals(s.Targets, float64(s.Targets.Kcal)),
		Days:     all[first:],
		Rolling:  map[Nutrient][]*float64{},
		Warnings: []Warning{},
	}
	for _, k := range Nutrients {
		line := make([]*float64, 0, len(r.Days))
		for i := first; i < len(all); i++ {
			line = append(line, rolling(all, i, k))
		}
		r.Rolling[k] = line
	}

	r.Week = week(all)
	if r.Week.Logged >= minLogged && r.Week.Logged+r.Week.Estimated >= minValues {
		r.Warnings = warnings(r.Week, Goals(s.Targets, r.Week.Budget))
	}
	r.Trends = trends(all)
	r.Weekdays = weekdays(all)
	r.Burn = burn(all)
	return r
}

func classify(s daily.Series, today string) []Day {
	out := make([]Day, 0, len(s.Days))
	for _, d := range s.Days {
		if d.Date > today {
			break
		}
		day := Day{Date: d.Date, Food: d.Food, Budget: d.Budget, WeightKg: d.WeightKg}
		switch {
		case d.Date == today:
			day.Status = Today
		case !d.Logged():
			day.Status = Missing
		case s.Targets.Kcal > 0 && len(d.Meals) <= 1 && d.Food.Kcal < float64(s.Targets.Kcal)/2:
			day.Status = Partial
		default:
			day.Status = Logged
		}
		out = append(out, day)
	}
	return out
}

// estimate fills in missing and partial days with the median of the same
// weekday over the 8 weeks before, or failing that of the 28 days before,
// counting only complete days. A partial day keeps what it logged where
// that is more than the estimate.
func estimate(days []Day) {
	for i := range days {
		d := &days[i]
		if d.Status != Missing && d.Status != Partial {
			continue
		}
		var weekday, recent []daily.Nutrients
		for j := i - 1; j >= 0 && j >= i-History; j-- {
			if days[j].Status != Logged {
				continue
			}
			if (i-j)%7 == 0 {
				weekday = append(weekday, days[j].Food)
			}
			if i-j <= 28 {
				recent = append(recent, days[j].Food)
			}
		}
		var from []daily.Nutrients
		switch {
		case len(weekday) >= 2:
			from = weekday
		case len(recent) >= 3:
			from = recent
		default:
			continue
		}
		e := median(from)
		if d.Status == Partial {
			for _, k := range Nutrients {
				k.set(&e, math.Max(k.Of(e), k.Of(d.Food)))
			}
		}
		d.Estimate = &e
	}
}

func median(list []daily.Nutrients) daily.Nutrients {
	var out daily.Nutrients
	xs := make([]float64, len(list))
	for _, k := range Nutrients {
		for i, n := range list {
			xs[i] = k.Of(n)
		}
		m, _ := stats.Median(xs)
		k.set(&out, stats.Round(m, 1))
	}
	return out
}

// rolling is the average of a nutrient over the seven days ending at days[i].
func rolling(days []Day, i int, k Nutrient) *float64 {
	if days[i].Status == Today {
		return nil
	}
	var xs []float64
	for j := max(i-rollingDays+1, 0); j <= i; j++ {
		if v, ok := days[j].value(); ok {
			xs = append(xs, k.Of(v))
		}
	}
	if len(xs) < minValues {
		return nil
	}
	m, _ := stats.Mean(xs)
	m = stats.Round(m, 1)
	return &m
}

// week sums up the seven days before today.
func week(days []Day) Week {
	var w Week
	window := past(days, rollingDays)
	if len(window) == 0 {
		return w
	}
	w.From, w.To = window[0].Date, window[len(window)-1].Date

	var values []daily.Nutrients
	var budgets []float64
	for _, d := range window {
		v, ok := d.value()
		switch {
		case !ok:
			w.Missing++
			continue
		case d.Status == Logged:
			w.Logged++
		default:
			w.Estimated++
		}
		values = append(values, v)
		budgets = append(budgets, float64(d.Budget))
	}
	xs := make([]float64, len(values))
	for _, k := range Nutrients {
		for i, v := range values {
			xs[i] = k.Of(v)
		}
		m, _ := stats.Mean(xs)
		k.set(&w.Average, stats.Round(m, 1))
	}
	b, _ := stats.Mean(budgets)
	w.Budget = math.Round(b)
	return w
}

// warnings are the nutrients off their goal, the furthest off first (as a
// share of the target).
func warnings(w Week, goals map[Nutrient]Goal) []Warning {
	type scored struct {
		Warning
		by float64
	}
	var list []scored
	for _, k := range Nutrients {
		g, ok := goals[k]
		if !ok || g.Target <= 0 {
			continue
		}
		avg := k.Of(w.Average)
		switch {
		case g.Max > 0 && avg > g.Max:
			list = append(list, scored{Warning{k, Over, avg, g.Target}, avg/g.Target - 1})
		case g.Min > 0 && avg < g.Min:
			list = append(list, scored{Warning{k, Under, avg, g.Target}, 1 - avg/g.Target})
		}
	}
	slices.SortStableFunc(list, func(a, b scored) int { return cmpDesc(a.by, b.by) })
	out := make([]Warning, len(list))
	for i, s := range list {
		out[i] = s.Warning
	}
	return out
}

// Range is the days to load for a report of `days` days ending today.
func Range(today time.Time, days int) (from, to time.Time) {
	return today.AddDate(0, 0, -(days - 1 + History)), today
}
