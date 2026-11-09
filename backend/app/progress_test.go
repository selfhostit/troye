package app_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/tests"

	"github.com/selfhostit/troye/app"
	"github.com/selfhostit/troye/insights"
)

func TestInsightReport(t *testing.T) {
	a := newApp(t)
	defer a.Cleanup()
	ada, _ := addUser(t, a, "ada@example.com", "owner")
	addProfile(t, a, ada.Id)
	bob, _ := addUser(t, a, "bob@example.com", "member")

	today := time.Date(2026, 11, 9, 12, 0, 0, 0, time.Local)
	for i := 1; i <= 7; i++ {
		day := today.AddDate(0, 0, -i).Format(time.DateOnly)
		for _, meal := range []string{"breakfast", "lunch", "dinner"} {
			save(t, a, "entries", map[string]any{"user": ada.Id, "date": day, "meal": meal, "name": meal, "kcal": 600, "protein": 30, "fat": 35})
		}
	}
	save(t, a, "entries", map[string]any{"user": bob.Id, "date": "2026-11-08", "meal": "lunch", "name": "Not Ada's", "kcal": 5000, "fat": 500})

	r, err := app.InsightReport(a, ada.Id, today, 14)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Days) != 14 || r.Days[13].Date != "2026-11-09" || r.Week.Logged != 7 {
		t.Fatalf("report %+v", r.Week)
	}
	if r.Week.Average.Fat != 105 {
		t.Errorf("fat %v", r.Week.Average.Fat)
	}
	if len(r.Warnings) == 0 || r.Warnings[0].Nutrient != insights.Fat {
		t.Errorf("warnings %+v", r.Warnings)
	}
}

func TestBudgetAdvice(t *testing.T) {
	a := newApp(t)
	defer a.Cleanup()
	ada, _ := addUser(t, a, "ada@example.com", "owner")
	p := addProfile(t, a, ada.Id) // 80 kg, losing 0.5 kg a week
	for _, w := range weighins(t, a, ada.Id) {
		if err := a.Delete(w); err != nil {
			t.Fatal(err)
		}
	}
	// four weeks of 2,000 kcal a day while losing 0.5 kg a week: a burn of 2,550
	today := time.Now()
	for i := 1; i <= 28; i++ {
		day := today.AddDate(0, 0, -i).Format(time.DateOnly)
		for _, meal := range []string{"breakfast", "lunch", "dinner"} {
			save(t, a, "entries", map[string]any{"user": ada.Id, "date": day, "meal": meal, "name": meal, "kcal": 2000.0 / 3})
		}
		if i%4 == 0 {
			save(t, a, "weighins", map[string]any{"user": ada.Id, "date": day, "weight_kg": 80 + float64(i)*0.5/7})
		}
	}

	r, err := app.InsightReport(a, ada.Id, today, 30)
	if err != nil {
		t.Fatal(err)
	}
	if r.Burn == nil || r.Burn.TDEE != 2550 {
		t.Fatalf("burn %+v", r.Burn)
	}
	b := r.Budget
	// 2,550 less 550 for the pace
	if b == nil || b.Suggested != 2000 || b.MeasuredTDEE != 0 || b.FormulaTDEE < 2000 || b.Current != reload(t, a, p).GetInt("calorie_budget") {
		t.Fatalf("advice %+v", b)
	}

	// choosing it moves the plan
	p = reload(t, a, p)
	p.Set("measured_tdee", r.Burn.TDEE)
	if err := a.Save(p); err != nil {
		t.Fatal(err)
	}
	if got := reload(t, a, p).GetInt("calorie_budget"); got != 2000 {
		t.Errorf("budget %d", got)
	}
	if r, _ := app.InsightReport(a, ada.Id, today, 30); r.Budget == nil || r.Budget.MeasuredTDEE != 2550 || r.Budget.Current != 2000 {
		t.Errorf("advice in use %+v", r.Budget)
	}
}

func TestInsightsRoute(t *testing.T) {
	scenarios := []struct {
		name, url string
		auth      bool
		status    int
		content   string
	}{
		{"needs a sign-in", "/api/app/insights", false, http.StatusUnauthorized, "authorization token"},
		{"a month by default", "/api/app/insights", true, http.StatusOK, `"warnings":[]`},
		{"a range it takes", "/api/app/insights?days=91", true, http.StatusOK, `"rolling"`},
		{"not a year", "/api/app/insights?days=365", true, http.StatusBadRequest, "7 to 182 days"},
	}
	for _, sc := range scenarios {
		a := newApp(t)
		ada, token := addUser(t, a, "ada@example.com", "owner")
		addProfile(t, a, ada.Id)
		s := tests.ApiScenario{
			Name:           sc.name,
			Method:         http.MethodGet,
			URL:            sc.url,
			ExpectedStatus: sc.status,
			TestAppFactory: func(testing.TB) *tests.TestApp { return a },
		}
		if sc.auth {
			s.Headers = map[string]string{"Authorization": token}
		}
		s.ExpectedContent = []string{sc.content}
		s.Test(t)
	}
}
