// Package daily loads a person's diary as one row per day: what they ate,
// against what they planned to eat, with the day's exercise, water and
// weigh-in. It is the one place that adds up entries, so the week summary,
// charts, insights and reminders all count the same way.
//
//	s, err := daily.Load(app, userID, from, to)
//	for _, d := range s.Days {
//		if d.Logged() && d.Food.Fat > float64(s.Targets.FatG) { ... }
//	}
package daily

import (
	"database/sql"
	"errors"
	"slices"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// Nutrients are the totals of a day's food. Mass is in grams.
type Nutrients struct {
	Kcal    float64 `json:"kcal"`
	Protein float64 `json:"protein"`
	Carbs   float64 `json:"carbs"`
	Fat     float64 `json:"fat"`
	Fiber   float64 `json:"fiber"`
	Sugar   float64 `json:"sugar"`
	Salt    float64 `json:"salt"`
}

func (n *Nutrients) add(r *core.Record) {
	n.Kcal += r.GetFloat("kcal")
	n.Protein += r.GetFloat("protein")
	n.Carbs += r.GetFloat("carbs")
	n.Fat += r.GetFloat("fat")
	n.Fiber += r.GetFloat("fiber")
	n.Sugar += r.GetFloat("sugar")
	n.Salt += r.GetFloat("salt")
}

// Targets are the daily goals from the person's plan; zero when they have no profile.
type Targets struct {
	Kcal     int `json:"kcal"`
	ProteinG int `json:"protein_g"`
	CarbsG   int `json:"carbs_g"`
	FatG     int `json:"fat_g"`
}

// Day is one calendar day of the diary.
type Day struct {
	// YYYY-MM-DD
	Date string    `json:"date"`
	Food Nutrients `json:"food"`
	// Entries counts the diary entries, add-ons included.
	Entries int `json:"entries"`
	// Meals are the meals with something logged, in the order of the day.
	Meals []string `json:"meals"`
	// ExerciseKcal is what the day's exercise burned above resting.
	ExerciseKcal int `json:"exercise_kcal"`
	// Budget is the plan's calorie budget plus the day's exercise.
	Budget  int `json:"budget"`
	WaterML int `json:"water_ml"`
	// WeightKg is the day's weigh-in, 0 without one.
	WeightKg float64 `json:"weight_kg"`
}

// Logged reports whether any food was logged on the day.
func (d Day) Logged() bool { return d.Entries > 0 }

// Series is a run of consecutive days, oldest first.
type Series struct {
	Targets Targets `json:"targets"`
	Days    []Day   `json:"days"`
	// PriorWeightKg is the latest weigh-in before the first day, 0 without one.
	PriorWeightKg float64 `json:"prior_weight_kg"`
}

// meals in the order of the day
var meals = []string{"breakfast", "lunch", "dinner", "snacks"}

// Load reads one person's days from `from` to `to`, both included, in the
// time zone of `from`. Days with nothing logged are there too, with zeros.
func Load(app core.App, user string, from, to time.Time) (Series, error) {
	from = dayOf(from)
	to = dayOf(to.In(from.Location()))
	var s Series
	if to.Before(from) {
		return s, errors.New("daily: the range ends before it starts")
	}

	p, err := app.FindFirstRecordByFilter("profiles", "user = {:user}", dbx.Params{"user": user})
	switch {
	case err == nil:
		s.Targets = Targets{
			Kcal:     p.GetInt("calorie_budget"),
			ProteinG: p.GetInt("protein_g"),
			CarbsG:   p.GetInt("carbs_g"),
			FatG:     p.GetInt("fat_g"),
		}
	case !errors.Is(err, sql.ErrNoRows):
		return s, err
	}

	byDate := map[string]*Day{}
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		s.Days = append(s.Days, Day{Date: d.Format(time.DateOnly), Budget: s.Targets.Kcal})
	}
	for i := range s.Days {
		byDate[s.Days[i].Date] = &s.Days[i]
	}

	inRange := "user = {:user} && date >= {:from} && date <= {:to}"
	params := dbx.Params{"user": user, "from": s.Days[0].Date, "to": s.Days[len(s.Days)-1].Date}

	entries, err := app.FindRecordsByFilter("entries", inRange, "", 0, 0, params)
	if err != nil {
		return s, err
	}
	for _, e := range entries {
		d := byDate[e.GetString("date")]
		d.Food.add(e)
		d.Entries++
		if m := e.GetString("meal"); !slices.Contains(d.Meals, m) {
			d.Meals = append(d.Meals, m)
		}
	}
	for i := range s.Days {
		slices.SortFunc(s.Days[i].Meals, func(a, b string) int {
			return slices.Index(meals, a) - slices.Index(meals, b)
		})
	}

	exercises, err := app.FindRecordsByFilter("exercises", inRange, "", 0, 0, params)
	if err != nil {
		return s, err
	}
	for _, x := range exercises {
		d := byDate[x.GetString("date")]
		d.ExerciseKcal += x.GetInt("kcal")
		d.Budget += x.GetInt("kcal")
	}

	water, err := app.FindRecordsByFilter("water", inRange, "", 0, 0, params)
	if err != nil {
		return s, err
	}
	for _, w := range water {
		byDate[w.GetString("date")].WaterML = w.GetInt("ml")
	}

	weighins, err := app.FindRecordsByFilter("weighins", inRange, "", 0, 0, params)
	if err != nil {
		return s, err
	}
	for _, w := range weighins {
		byDate[w.GetString("date")].WeightKg = w.GetFloat("weight_kg")
	}
	before, err := app.FindRecordsByFilter("weighins", "user = {:user} && date < {:from}", "-date", 1, 0, params)
	if err != nil {
		return s, err
	}
	if len(before) == 1 {
		s.PriorWeightKg = before[0].GetFloat("weight_kg")
	}
	return s, nil
}

// LoggedDates are the days with food logged from `since` on, for streaks and
// gaps that look further back than a Series needs to.
func LoggedDates(app core.App, user string, since time.Time) (map[string]bool, error) {
	var dates []string
	err := app.RecordQuery("entries").
		Select("date").
		Distinct(true).
		AndWhere(dbx.HashExp{"user": user}).
		AndWhere(dbx.NewExp("date >= {:since}", dbx.Params{"since": since.Format(time.DateOnly)})).
		Column(&dates)
	if err != nil {
		return nil, err
	}
	logged := make(map[string]bool, len(dates))
	for _, d := range dates {
		logged[d] = true
	}
	return logged, nil
}

func dayOf(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}
