package insights

import (
	"math"
	"slices"
	"time"

	"github.com/selfhostit/troye/stats"
)

// Patterns that take weeks to show: nutrients drifting up or down, a weekday
// that runs over, and how much the person really burns, from what they ate
// against how their weight moved. Each is nil or empty until there is enough
// data to say it with some confidence.

// how far back the patterns look, in days before today
const (
	trendDays   = 28
	weekdayDays = 8 * 7
	burnDays    = 28
)

// Trend is a nutrient drifting over the last four weeks.
type Trend struct {
	Nutrient Nutrient `json:"nutrient"`
	// PerWeek is the change a week, from a straight line through the days.
	PerWeek float64 `json:"per_week"`
	Average float64 `json:"average"`
}

// fewest days with a value for a trend
const minTrendDays = 14

// trends are the nutrients whose straight line moved by at least a tenth of
// their average over the four weeks, the biggest move first.
func trends(days []Day) []Trend {
	window := past(days, trendDays)
	out := []Trend{}
	for _, k := range Nutrients {
		var xs, ys []float64
		for i, d := range window {
			if v, ok := d.value(); ok {
				xs = append(xs, float64(i))
				ys = append(ys, k.Of(v))
			}
		}
		if len(xs) < minTrendDays {
			continue
		}
		line, ok := stats.LinearFit(xs, ys)
		avg, _ := stats.Mean(ys)
		if !ok || avg <= 0 || math.Abs(line.Slope*trendDays) < avg/10 {
			continue
		}
		out = append(out, Trend{Nutrient: k, PerWeek: stats.Round(line.Slope*7, 1), Average: stats.Round(avg, 1)})
	}
	slices.SortStableFunc(out, func(a, b Trend) int {
		return cmpDesc(math.Abs(a.PerWeek)/a.Average, math.Abs(b.PerWeek)/b.Average)
	})
	return out
}

// Weekdays is how each day of the week compares with the person's average day.
type Weekdays struct {
	// Diffs are kcal above (or below) the average, Monday first; null for a
	// weekday with fewer than three logged days.
	Diffs []*float64 `json:"diffs"`
	// Weekday is the one furthest off, as "Saturday", when it is at least
	// 200 kcal off; empty otherwise.
	Weekday string  `json:"weekday"`
	Diff    float64 `json:"diff"`
}

const (
	minPerWeekday = 3
	minWeekdays   = 5
	notableKcal   = 200
)

// weekdays compares the weekdays over the last eight weeks, counting only
// days logged in full (estimates come from weekdays, so they would echo).
func weekdays(days []Day) *Weekdays {
	byDay := make([][]float64, 7)
	var all []float64
	for _, d := range past(days, weekdayDays) {
		if d.Status != Logged {
			continue
		}
		t, err := time.Parse(time.DateOnly, d.Date)
		if err != nil {
			continue
		}
		i := (int(t.Weekday()) + 6) % 7 // Monday is 0
		byDay[i] = append(byDay[i], d.Food.Kcal)
		all = append(all, d.Food.Kcal)
	}
	mean, ok := stats.Mean(all)
	if !ok {
		return nil
	}
	w := &Weekdays{Diffs: make([]*float64, 7)}
	seen := 0
	for i, xs := range byDay {
		if len(xs) < minPerWeekday {
			continue
		}
		m, _ := stats.Mean(xs)
		diff := math.Round(m - mean)
		w.Diffs[i] = &diff
		seen++
		if math.Abs(diff) >= notableKcal && math.Abs(diff) > math.Abs(w.Diff) {
			w.Weekday = time.Weekday((i + 1) % 7).String()
			w.Diff = diff
		}
	}
	if seen < minWeekdays {
		return nil
	}
	return w
}

// Burn is what the person really burns a day, measured from the last four
// weeks: what they ate, less the energy their change in weight stored. It
// takes in whatever logging tends to miss, so a budget built on it matches
// how they really log.
type Burn struct {
	From string `json:"from"`
	To   string `json:"to"`
	// TDEE is the measured burn, rounded to 10 kcal.
	TDEE int `json:"tdee"`
	// IntakeKcal is the average intake, estimates for gaps included.
	IntakeKcal int `json:"intake_kcal"`
	// WeightPerWeek is the change in kg a week, from a line through the weigh-ins.
	WeightPerWeek float64 `json:"weight_per_week"`
	Days          int     `json:"days"`
	Weighins      int     `json:"weighins"`
}

const (
	// the energy in a kg of body weight, the usual rule of thumb
	kcalPerKg      = 7700
	minBurnLogged  = 14
	minBurnWeighin = 4
	minBurnSpan    = 14
)

// burn measures the burn, or nil without four weigh-ins over two weeks and
// two weeks logged in full, or when the answer is out of any plausible range.
func burn(days []Day) *Burn {
	window := past(days, burnDays)
	if len(window) == 0 {
		return nil
	}
	var intake, wx, wy []float64
	logged := 0
	for i, d := range window {
		if v, ok := d.value(); ok {
			intake = append(intake, v.Kcal)
		}
		if d.Status == Logged {
			logged++
		}
		if d.WeightKg > 0 {
			wx = append(wx, float64(i))
			wy = append(wy, d.WeightKg)
		}
	}
	if logged < minBurnLogged || len(wx) < minBurnWeighin || wx[len(wx)-1]-wx[0] < minBurnSpan {
		return nil
	}
	line, ok := stats.LinearFit(wx, wy)
	if !ok {
		return nil
	}
	eaten, _ := stats.Mean(intake)
	tdee := eaten - line.Slope*kcalPerKg
	if tdee < 1000 || tdee > 5000 {
		return nil
	}
	return &Burn{
		From:          window[0].Date,
		To:            window[len(window)-1].Date,
		TDEE:          int(math.Round(tdee/10) * 10),
		IntakeKcal:    int(math.Round(eaten)),
		WeightPerWeek: stats.Round(line.Slope*7, 2),
		Days:          len(intake),
		Weighins:      len(wx),
	}
}

// past is the last n days before today (days ends with today, when it is there).
func past(days []Day, n int) []Day {
	end := len(days)
	if end > 0 && days[end-1].Status == Today {
		end--
	}
	return days[max(end-n, 0):end]
}

func cmpDesc(a, b float64) int {
	switch {
	case a > b:
		return -1
	case a < b:
		return 1
	}
	return 0
}
