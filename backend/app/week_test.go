package app_test

import (
	"math"
	"strconv"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/selfhostit/troye/app"
)

func save(t testing.TB, a core.App, collection string, data map[string]any) *core.Record {
	t.Helper()
	c, err := a.FindCollectionByNameOrId(collection)
	if err != nil {
		t.Fatal(err)
	}
	r := core.NewRecord(c)
	r.Load(data)
	if err := a.Save(r); err != nil {
		t.Fatal(err)
	}
	return r
}

func TestWeekSummary(t *testing.T) {
	a := newApp(t)
	defer a.Cleanup()
	ada, _ := addUser(t, a, "ada@example.com", "owner")
	p := addProfile(t, a, ada.Id)
	// weigh-ins on fixed days, without the one onboarding logged on the real today
	for _, w := range weighins(t, a, ada.Id) {
		if err := a.Delete(w); err != nil {
			t.Fatal(err)
		}
	}
	save(t, a, "weighins", map[string]any{"user": ada.Id, "date": "2026-10-12", "weight_kg": 80.8})
	save(t, a, "weighins", map[string]any{"user": ada.Id, "date": "2026-10-23", "weight_kg": 80})
	budget := reload(t, a, p).GetInt("calorie_budget")
	today := time.Date(2026, 10, 24, 12, 0, 0, 0, time.Local) // a Saturday
	monday := app.Monday(today)
	if monday.Format(time.DateOnly) != "2026-10-19" {
		t.Fatalf("Monday %s", monday)
	}

	for _, d := range []string{"2026-10-17", "2026-10-18", "2026-10-19", "2026-10-20", "2026-10-22", "2026-10-23", "2026-10-24"} {
		save(t, a, "entries", map[string]any{"user": ada.Id, "date": d, "meal": "lunch", "name": "Lunch", "kcal": 1000})
	}
	save(t, a, "entries", map[string]any{"user": ada.Id, "date": "2026-10-20", "meal": "dinner", "name": "Pizza night", "kcal": float64(budget - 500)})
	save(t, a, "exercises", map[string]any{"user": ada.Id, "date": "2026-10-20", "kind": "run", "minutes": 60})
	save(t, a, "water", map[string]any{"user": ada.Id, "date": "2026-10-19", "ml": 1750})

	w, err := app.WeekSummary(a, ada.Id, monday, today)
	if err != nil {
		t.Fatal(err)
	}
	tue := w.Days[1]
	if tue.Exercise != 704 || tue.Budget != budget+704 || tue.Kcal != float64(budget+500) {
		t.Errorf("Tuesday %+v", tue)
	}
	if w.Days[0].WaterML != 1750 || w.Days[2].Logged {
		t.Errorf("Monday %+v, Wednesday %+v", w.Days[0], w.Days[2])
	}
	if w.DaysLogged != 5 || w.UnderBudget != 5 || w.WeightChange != -0.8 || !w.HasWeight {
		t.Errorf("week %+v", w)
	}
	// Wednesday has nothing, so the streak runs Thursday to Saturday
	if w.Streak != 3 {
		t.Errorf("streak %d", w.Streak)
	}

	msg, ok := app.WeeklyMessage(w, "metric")
	if !ok || msg.Title != "Great week: 5 days within budget" || msg.Body != "5 of 7 days logged · "+thousandsOf(int(math.Round(float64(4000+budget+500)/5)))+" kcal a day · −0.8 kg" {
		t.Errorf("message %+v", msg)
	}
	if _, ok := app.WeeklyMessage(app.Week{}, "metric"); ok {
		t.Error("a message for an empty week")
	}
}

// thousandsOf formats like the message does: 1,234.
func thousandsOf(n int) string {
	s := strconv.Itoa(n)
	if len(s) <= 3 {
		return s
	}
	return s[:len(s)-3] + "," + s[len(s)-3:]
}
