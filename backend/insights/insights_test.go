package insights_test

import (
	"testing"
	"time"

	"github.com/selfhostit/troye/daily"
	"github.com/selfhostit/troye/insights"
)

var targets = daily.Targets{Kcal: 2000, ProteinG: 100, CarbsG: 250, FatG: 65}

// series makes `n` days ending on `today`, each filled in by fill (nil = nothing logged).
func series(today string, n int, fill func(i int, date string) *daily.Nutrients) daily.Series {
	end, _ := time.Parse(time.DateOnly, today)
	s := daily.Series{Targets: targets}
	for i := range n {
		date := end.AddDate(0, 0, i-n+1).Format(time.DateOnly)
		d := daily.Day{Date: date, Budget: targets.Kcal}
		if f := fill(i, date); f != nil {
			d.Food = *f
			d.Entries = 3
			d.Meals = []string{"breakfast", "lunch", "dinner"}
		}
		s.Days = append(s.Days, d)
	}
	return s
}

var usual = daily.Nutrients{Kcal: 1900, Protein: 105, Carbs: 230, Fat: 60, Fiber: 27, Sugar: 40, Salt: 4.5}

func TestAnalyzeImputesMissingDays(t *testing.T) {
	// 10 weeks of usual days, with Sundays at 2,500 kcal, then two days skipped
	today := "2026-11-09" // a Monday
	s := series(today, 70, func(i int, date string) *daily.Nutrients {
		if date == "2026-11-07" || date == "2026-11-08" {
			return nil
		}
		n := usual
		if d, _ := time.Parse(time.DateOnly, date); d.Weekday() == time.Sunday {
			n.Kcal = 2500
		}
		return &n
	})
	r := insights.Analyze(s, today, 14)

	if len(r.Days) != 14 || r.Days[13].Date != today || r.Days[13].Status != insights.Today {
		t.Fatalf("days %+v", r.Days[13])
	}
	sat, sun := r.Days[11], r.Days[12]
	if sat.Status != insights.Missing || sat.Estimate == nil || sat.Estimate.Kcal != 1900 {
		t.Errorf("Saturday %+v", sat)
	}
	// the same weekday, so the Sunday habit carries over
	if sun.Status != insights.Missing || sun.Estimate == nil || sun.Estimate.Kcal != 2500 {
		t.Errorf("Sunday %+v %+v", sun, sun.Estimate)
	}
	if w := r.Week; w.From != "2026-11-02" || w.To != "2026-11-08" || w.Logged != 5 || w.Estimated != 2 || w.Missing != 0 {
		t.Errorf("week %+v", w)
	}
	if len(r.Warnings) != 0 {
		t.Errorf("warnings for a usual week %+v", r.Warnings)
	}
	if len(r.Rolling[insights.Kcal]) != 14 || r.Rolling[insights.Kcal][13] != nil || r.Rolling[insights.Kcal][12] == nil {
		t.Errorf("rolling %v", r.Rolling[insights.Kcal])
	}
}

func TestAnalyzeWithoutHistory(t *testing.T) {
	today := "2026-11-09"
	s := series(today, 70, func(i int, _ string) *daily.Nutrients {
		if i < 66 {
			return nil
		}
		return &usual
	})
	r := insights.Analyze(s, today, 14)
	for _, d := range r.Days[:10] {
		if d.Status != insights.Missing || d.Estimate != nil {
			t.Fatalf("estimated with no history %+v", d)
		}
	}
	// three logged days aren't enough for a rolling average
	if r.Rolling[insights.Kcal][12] != nil {
		t.Errorf("rolling %v", *r.Rolling[insights.Kcal][12])
	}
	if r.Week.Logged != 3 || r.Week.Missing != 4 || len(r.Warnings) != 0 {
		t.Errorf("week %+v, warnings %+v", r.Week, r.Warnings)
	}
}

func TestAnalyzePartialDays(t *testing.T) {
	today := "2026-11-09"
	s := series(today, 70, func(int, string) *daily.Nutrients { return &usual })
	// yesterday only breakfast was logged
	y := &s.Days[68]
	y.Food = daily.Nutrients{Kcal: 400, Protein: 15, Carbs: 60, Fat: 10, Sugar: 70}
	y.Meals = []string{"breakfast"}
	r := insights.Analyze(s, today, 14)
	d := r.Days[12]
	if d.Status != insights.Partial || d.Estimate == nil {
		t.Fatalf("yesterday %+v", d)
	}
	// the estimate never drops what was logged
	if d.Estimate.Kcal != 1900 || d.Estimate.Sugar != 70 {
		t.Errorf("estimate %+v", d.Estimate)
	}
	if r.Week.Estimated != 1 || r.Week.Logged != 6 {
		t.Errorf("week %+v", r.Week)
	}
}

func TestAnalyzeWarnings(t *testing.T) {
	today := "2026-11-09"
	heavy := usual
	heavy.Fat, heavy.Protein, heavy.Salt = 95, 60, 5.2
	s := series(today, 70, func(int, string) *daily.Nutrients { return &heavy })
	r := insights.Analyze(s, today, 14)

	if len(r.Warnings) != 2 {
		t.Fatalf("warnings %+v", r.Warnings)
	}
	// fat is 46% over, protein 40% under: fat first
	fat, protein := r.Warnings[0], r.Warnings[1]
	if fat.Nutrient != insights.Fat || fat.Kind != insights.Over || fat.Average != 95 || fat.Target != 65 {
		t.Errorf("fat %+v", fat)
	}
	if protein.Nutrient != insights.Protein || protein.Kind != insights.Under || protein.Target != 100 {
		t.Errorf("protein %+v", protein)
	}
	// salt is over 5 g but within the leeway
	if g := r.Goals[insights.Salt]; g.Target != 5 || g.Max != 5.5 {
		t.Errorf("salt goal %+v", g)
	}
}

func TestAnalyzeCaloriesAgainstExercise(t *testing.T) {
	today := "2026-11-09"
	big := usual
	big.Kcal = 2400
	s := series(today, 70, func(int, string) *daily.Nutrients { return &big })
	if w := insights.Analyze(s, today, 14).Warnings; len(w) != 1 || w[0].Nutrient != insights.Kcal || w[0].Target != 2000 {
		t.Fatalf("warnings %+v", w)
	}
	// with 500 kcal of exercise a day, 2,400 is within budget
	for i := range s.Days {
		s.Days[i].Budget = 2500
	}
	if w := insights.Analyze(s, today, 14).Warnings; len(w) != 0 {
		t.Errorf("warnings with exercise %+v", w)
	}
}

func TestRange(t *testing.T) {
	today := time.Date(2026, 11, 9, 0, 0, 0, 0, time.UTC)
	from, to := insights.Range(today, 14)
	if from.Format(time.DateOnly) != "2026-09-01" || !to.Equal(today) {
		t.Errorf("range %s to %s", from, to)
	}
}
