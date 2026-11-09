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
