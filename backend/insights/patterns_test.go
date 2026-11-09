package insights_test

import (
	"testing"
	"time"

	"github.com/selfhostit/troye/daily"
	"github.com/selfhostit/troye/insights"
)

func weekday(date string) time.Weekday {
	d, _ := time.Parse(time.DateOnly, date)
	return d.Weekday()
}

func TestTrends(t *testing.T) {
	today := "2026-11-09"
	// fat climbs 1 g a day over the last four weeks; the rest holds
	s := series(today, 70, func(i int, _ string) *daily.Nutrients {
		n := usual
		n.Fat = 60 + float64(max(i-41, 0))
		return &n
	})
	r := insights.Analyze(s, today, 14)
	if len(r.Trends) != 1 || r.Trends[0].Nutrient != insights.Fat || r.Trends[0].PerWeek != 7 {
		t.Fatalf("trends %+v", r.Trends)
	}

	// a few days logged aren't a trend
	s = series(today, 70, func(i int, _ string) *daily.Nutrients {
		if i < 60 {
			return nil
		}
		n := usual
		n.Fat = float64(i)
		return &n
	})
	if r := insights.Analyze(s, today, 14); len(r.Trends) != 0 {
		t.Errorf("trends from ten days %+v", r.Trends)
	}
}

func TestWeekdays(t *testing.T) {
	today := "2026-11-09"
	s := series(today, 70, func(_ int, date string) *daily.Nutrients {
		n := usual
		if weekday(date) == time.Saturday {
			n.Kcal += 700
		}
		return &n
	})
	w := insights.Analyze(s, today, 14).Weekdays
	if w == nil || w.Weekday != "Saturday" || w.Diff != 600 || *w.Diffs[5] != 600 || *w.Diffs[0] != -100 {
		t.Fatalf("weekdays %+v", w)
	}

	// an even week has no notable day
	s = series(today, 70, func(int, string) *daily.Nutrients { return &usual })
	if w := insights.Analyze(s, today, 14).Weekdays; w == nil || w.Weekday != "" {
		t.Errorf("even week %+v", w)
	}
	// nor does a week with nothing logged
	s = series(today, 70, func(int, string) *daily.Nutrients { return nil })
	if w := insights.Analyze(s, today, 14).Weekdays; w != nil {
		t.Errorf("empty weeks %+v", w)
	}
}

func TestBurn(t *testing.T) {
	today := "2026-11-09"
	// eating 1,900 a day and losing 0.5 kg a week: 1,900 + 0.5 x 7,700 / 7 = 2,450
	s := series(today, 70, func(int, string) *daily.Nutrients { return &usual })
	for i := range s.Days {
		if i%3 == 0 {
			s.Days[i].WeightKg = 85 - float64(i)*0.5/7
		}
	}
	b := insights.Analyze(s, today, 14).Burn
	if b == nil || b.TDEE != 2450 || b.IntakeKcal != 1900 || b.WeightPerWeek != -0.5 || b.Days != 28 || b.Weighins != 9 {
		t.Fatalf("burn %+v", b)
	}
	if b.From != "2026-10-12" || b.To != "2026-11-08" {
		t.Errorf("window %s to %s", b.From, b.To)
	}

	// weigh-ins over one week only aren't enough
	for i := range s.Days {
		if i < 60 {
			s.Days[i].WeightKg = 0
		}
	}
	if b := insights.Analyze(s, today, 14).Burn; b != nil {
		t.Errorf("burn from a week of weigh-ins %+v", b)
	}
}
