package app

import (
	"math"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// Exercise calories are what the activity burns on top of resting, since the
// daily budget already counts resting and everyday activity: (MET - 1) x kg x
// hours, with METs from the Compendium of Physical Activities. They are added
// to that day's budget (see backend/daily).

// mets is how hard each kind of exercise is, in METs.
var mets = map[string]float64{
	"walk":     3.5,
	"dog_walk": 3.0,
	"run":      9.8,
	"cycle":    7.5,
	"swim":     7.0,
	"strength": 5.0,
	"yoga":     2.5,
	"hike":     6.0,
	"sport":    7.0,
	"other":    4.5,
}

// ExerciseKcal is the calories an exercise burns above resting.
func ExerciseKcal(kind string, minutes, kg float64) int {
	met, ok := mets[kind]
	if !ok {
		met = mets["other"]
	}
	if kg <= 0 {
		kg = 70
	}
	return int(math.Round((met - 1) * kg * minutes / 60))
}

func registerExercise(app core.App) {
	app.OnRecordCreate("exercises").BindFunc(func(e *core.RecordEvent) error {
		if e.Record.GetInt("kcal") == 0 {
			fillExercise(e.App, e.Record)
		}
		return e.Next()
	})
	app.OnRecordUpdate("exercises").BindFunc(func(e *core.RecordEvent) error {
		old := e.Record.Original()
		changed := old.GetString("kind") != e.Record.GetString("kind") || old.GetInt("minutes") != e.Record.GetInt("minutes")
		// new minutes move the calories, unless the person typed their own
		if e.Record.GetInt("kcal") == 0 || changed && e.Record.GetInt("kcal") == old.GetInt("kcal") {
			fillExercise(e.App, e.Record)
		}
		return e.Next()
	})
}

func fillExercise(app core.App, r *core.Record) {
	kg := 0.0
	if p, err := app.FindFirstRecordByFilter("profiles", "user = {:user}", dbx.Params{"user": r.GetString("user")}); err == nil {
		kg = p.GetFloat("weight_kg")
	}
	r.Set("kcal", ExerciseKcal(r.GetString("kind"), r.GetFloat("minutes"), kg))
}
