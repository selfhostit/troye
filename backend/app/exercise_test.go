package app_test

import (
	"testing"

	"github.com/selfhostit/troye/app"
)

func TestExerciseKcal(t *testing.T) {
	// 45 minutes with the dog at 80 kg: (3 - 1) x 80 x 0.75
	if got := app.ExerciseKcal("dog_walk", 45, 80); got != 120 {
		t.Errorf("dog walk %d", got)
	}
	if got := app.ExerciseKcal("unknown", 60, 0); got != 245 {
		t.Errorf("unknown kind at the default weight %d", got)
	}
}

func TestExerciseCalories(t *testing.T) {
	a := newApp(t)
	defer a.Cleanup()
	ada, _ := addUser(t, a, "ada@example.com", "owner")
	addProfile(t, a, ada.Id) // 80 kg

	run := save(t, a, "exercises", map[string]any{"user": ada.Id, "date": "2026-10-24", "kind": "run", "minutes": 30})
	if run.GetInt("kcal") != 352 {
		t.Errorf("run %d kcal", run.GetInt("kcal"))
	}
	// longer moves the calories
	run = reload(t, a, run)
	run.Set("minutes", 60)
	if err := a.Save(run); err != nil {
		t.Fatal(err)
	}
	if run.GetInt("kcal") != 704 {
		t.Errorf("an hour's run %d kcal", run.GetInt("kcal"))
	}
	// the person's own number from a watch stays
	own := save(t, a, "exercises", map[string]any{"user": ada.Id, "date": "2026-10-24", "kind": "cycle", "minutes": 40, "kcal": 410})
	if own.GetInt("kcal") != 410 {
		t.Errorf("own calories became %d", own.GetInt("kcal"))
	}
}
