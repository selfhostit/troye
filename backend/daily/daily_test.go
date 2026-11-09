package daily_test

import (
	"os"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"

	"github.com/selfhostit/troye/daily"
	_ "github.com/selfhostit/troye/migrations"
)

// template is a data dir with every migration applied, which each test app
// starts from as a copy.
var template string

func TestMain(m *testing.M) {
	empty, err := os.MkdirTemp("", "troye-daily-")
	if err != nil {
		panic(err)
	}
	migrated, err := tests.NewTestAppWithConfig(core.BaseAppConfig{DataDir: empty})
	if err != nil {
		panic(err)
	}
	template = migrated.DataDir()
	if err := migrated.ResetBootstrapState(); err != nil {
		panic(err)
	}
	code := m.Run()
	_ = os.RemoveAll(empty)
	_ = os.RemoveAll(template)
	os.Exit(code)
}

func newApp(t testing.TB) *tests.TestApp {
	a, err := tests.NewTestAppWithConfig(core.BaseAppConfig{DataDir: template})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(a.Cleanup)
	return a
}

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

func user(t testing.TB, a core.App, email string) string {
	users, _ := a.FindCollectionByNameOrId("users")
	u := core.NewRecord(users)
	u.SetEmail(email)
	u.SetPassword("password123")
	u.Set("role", "member")
	if err := a.Save(u); err != nil {
		t.Fatal(err)
	}
	return u.Id
}

func day(s string) time.Time {
	d, _ := time.ParseInLocation(time.DateOnly, s, time.Local)
	return d
}

func TestLoad(t *testing.T) {
	a := newApp(t)
	ada := user(t, a, "ada@example.com")
	bob := user(t, a, "bob@example.com")
	save(t, a, "profiles", map[string]any{"user": ada, "calorie_budget": 1800, "protein_g": 90, "carbs_g": 225, "fat_g": 60})

	save(t, a, "entries", map[string]any{"user": ada, "date": "2026-10-20", "meal": "dinner", "name": "Pasta", "kcal": 700, "protein": 25, "carbs": 90, "fat": 20, "fiber": 6, "sugar": 8, "salt": 1.5})
	save(t, a, "entries", map[string]any{"user": ada, "date": "2026-10-20", "meal": "breakfast", "name": "Oats", "kcal": 350, "protein": 12, "carbs": 55, "fat": 7, "fiber": 8, "sugar": 4, "salt": 0.1})
	save(t, a, "entries", map[string]any{"user": ada, "date": "2026-10-20", "meal": "breakfast", "name": "Milk", "kcal": 60, "protein": 4, "carbs": 5, "fat": 2})
	save(t, a, "entries", map[string]any{"user": ada, "date": "2026-10-25", "meal": "lunch", "name": "Outside the range", "kcal": 500})
	save(t, a, "entries", map[string]any{"user": bob, "date": "2026-10-20", "meal": "lunch", "name": "Someone else's", "kcal": 900})
	save(t, a, "exercises", map[string]any{"user": ada, "date": "2026-10-21", "kind": "run", "minutes": 30, "kcal": 300})
	save(t, a, "exercises", map[string]any{"user": ada, "date": "2026-10-21", "kind": "walk", "minutes": 30, "kcal": 100})
	save(t, a, "water", map[string]any{"user": ada, "date": "2026-10-22", "ml": 1500})
	save(t, a, "weighins", map[string]any{"user": ada, "date": "2026-10-10", "weight_kg": 81})
	save(t, a, "weighins", map[string]any{"user": ada, "date": "2026-10-15", "weight_kg": 80.6})
	save(t, a, "weighins", map[string]any{"user": ada, "date": "2026-10-22", "weight_kg": 80.2})

	s, err := daily.Load(a, ada, day("2026-10-20"), day("2026-10-24"))
	if err != nil {
		t.Fatal(err)
	}
	if s.Targets != (daily.Targets{Kcal: 1800, ProteinG: 90, CarbsG: 225, FatG: 60}) {
		t.Errorf("targets %+v", s.Targets)
	}
	if len(s.Days) != 5 || s.Days[0].Date != "2026-10-20" || s.Days[4].Date != "2026-10-24" {
		t.Fatalf("days %+v", s.Days)
	}

	tue := s.Days[0]
	want := daily.Nutrients{Kcal: 1110, Protein: 41, Carbs: 150, Fat: 29, Fiber: 14, Sugar: 12, Salt: 1.6}
	if tue.Food != want || tue.Entries != 3 || !tue.Logged() || tue.Budget != 1800 {
		t.Errorf("Tuesday %+v", tue)
	}
	if len(tue.Meals) != 2 || tue.Meals[0] != "breakfast" || tue.Meals[1] != "dinner" {
		t.Errorf("meals %v", tue.Meals)
	}
	if wed := s.Days[1]; wed.Logged() || wed.ExerciseKcal != 400 || wed.Budget != 2200 {
		t.Errorf("Wednesday %+v", wed)
	}
	if thu := s.Days[2]; thu.WaterML != 1500 || thu.WeightKg != 80.2 {
		t.Errorf("Thursday %+v", thu)
	}
	if s.PriorWeightKg != 80.6 {
		t.Errorf("prior weight %v", s.PriorWeightKg)
	}
}

func TestLoadWithoutProfile(t *testing.T) {
	a := newApp(t)
	ada := user(t, a, "ada@example.com")
	s, err := daily.Load(a, ada, day("2026-10-20"), day("2026-10-20"))
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Days) != 1 || s.Days[0].Budget != 0 || s.Targets != (daily.Targets{}) {
		t.Errorf("series %+v", s)
	}
	if _, err := daily.Load(a, ada, day("2026-10-20"), day("2026-10-19")); err == nil {
		t.Error("a backwards range loaded")
	}
}

func TestLoggedDates(t *testing.T) {
	a := newApp(t)
	ada := user(t, a, "ada@example.com")
	for _, d := range []string{"2026-09-01", "2026-10-01", "2026-10-01", "2026-10-02"} {
		save(t, a, "entries", map[string]any{"user": ada, "date": d, "meal": "lunch", "name": "Lunch", "kcal": 500})
	}
	logged, err := daily.LoggedDates(a, ada, day("2026-09-15"))
	if err != nil {
		t.Fatal(err)
	}
	if len(logged) != 2 || !logged["2026-10-01"] || !logged["2026-10-02"] {
		t.Errorf("logged %v", logged)
	}
}
