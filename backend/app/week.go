package app

import (
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

	"github.com/selfhostit/troye/daily"
	"github.com/selfhostit/troye/jobs"
	"github.com/selfhostit/troye/push"
)

// The weekly summary: GET /api/app/week for the week screen, and a push on
// Monday morning about how last week went. The days come from backend/daily;
// this file only sums them up and words the message.

func registerWeek(app core.App) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.GET("/api/app/week", weekRoute).Bind(apis.RequireAuth("users"))
		return se.Next()
	})

	// Monday morning: how last week went
	jobs.MustAdd(app, "weekly-summary", "0 9 * * 1", func(app core.App) error {
		return sendWeeklySummaries(app, time.Now())
	})
}

// Day is one day of a week's summary.
type Day struct {
	Date     string  `json:"date"`
	Kcal     float64 `json:"kcal"`
	Exercise int     `json:"exercise"`
	// the calorie budget plus that day's exercise
	Budget  int  `json:"budget"`
	WaterML int  `json:"water_ml"`
	Logged  bool `json:"logged"`
}

// Week sums up seven days from a Monday.
type Week struct {
	Start      string `json:"start"`
	Days       []Day  `json:"days"`
	DaysLogged int    `json:"days_logged"`
	// the average over the days with food logged
	AvgKcal     int `json:"avg_kcal"`
	UnderBudget int `json:"under_budget"`
	// the latest weigh-in of the week against the one before the week; 0 without both
	WeightChange float64 `json:"weight_change"`
	HasWeight    bool    `json:"has_weight"`
	// days in a row with food logged, up to today
	Streak int `json:"streak"`
}

// Monday is the Monday of the week a day is in.
func Monday(day time.Time) time.Time {
	offset := (int(day.Weekday()) + 6) % 7
	return time.Date(day.Year(), day.Month(), day.Day()-offset, 0, 0, 0, 0, day.Location())
}

// WeekSummary sums up a person's week from its Monday.
func WeekSummary(app core.App, user string, monday, today time.Time) (Week, error) {
	s, err := daily.Load(app, user, monday, monday.AddDate(0, 0, 6))
	if err != nil {
		return Week{Start: monday.Format(time.DateOnly)}, err
	}
	w := summarize(s)
	w.Streak, err = streak(app, user, today)
	return w, err
}

// summarize sums up a week's days, all but the streak.
func summarize(s daily.Series) Week {
	w := Week{Start: s.Days[0].Date}
	var total float64
	lastWeight := 0.0
	for _, d := range s.Days {
		w.Days = append(w.Days, Day{
			Date:     d.Date,
			Kcal:     d.Food.Kcal,
			Exercise: d.ExerciseKcal,
			Budget:   d.Budget,
			WaterML:  d.WaterML,
			Logged:   d.Logged(),
		})
		if d.WeightKg > 0 {
			lastWeight = d.WeightKg
		}
		if !d.Logged() {
			continue
		}
		w.DaysLogged++
		total += d.Food.Kcal
		if d.Food.Kcal <= float64(d.Budget) {
			w.UnderBudget++
		}
	}
	if w.DaysLogged > 0 {
		w.AvgKcal = int(math.Round(total / float64(w.DaysLogged)))
	}
	if lastWeight > 0 && s.PriorWeightKg > 0 {
		w.HasWeight = true
		w.WeightChange = math.Round((lastWeight-s.PriorWeightKg)*10) / 10
	}
	return w
}

// streak counts the days in a row with food logged, ending today, or
// yesterday while today is still empty.
func streak(app core.App, user string, today time.Time) (int, error) {
	logged, err := daily.LoggedDates(app, user, today.AddDate(0, 0, -400))
	if err != nil {
		return 0, err
	}
	day := today
	if !logged[day.Format(time.DateOnly)] {
		day = day.AddDate(0, 0, -1)
	}
	n := 0
	for logged[day.Format(time.DateOnly)] {
		n++
		day = day.AddDate(0, 0, -1)
	}
	return n, nil
}

func weekRoute(e *core.RequestEvent) error {
	now := time.Now()
	day := now
	if s := e.Request.URL.Query().Get("day"); s != "" {
		d, err := time.ParseInLocation(time.DateOnly, s, now.Location())
		if err != nil {
			return e.BadRequestError("Give the day as YYYY-MM-DD.", nil)
		}
		day = d
	}
	w, err := WeekSummary(e.App, e.Auth.Id, Monday(day), now)
	if err != nil {
		return e.InternalServerError("", err)
	}
	return e.JSON(http.StatusOK, w)
}

// sendWeeklySummaries tells everyone who logged last week how it went.
func sendWeeklySummaries(app core.App, now time.Time) error {
	profiles, err := app.FindRecordsByFilter("profiles", "onboarded = true", "", 0, 0)
	if err != nil {
		return err
	}
	lastMonday := Monday(now).AddDate(0, 0, -7)
	for _, p := range profiles {
		w, err := WeekSummary(app, p.GetString("user"), lastMonday, now)
		if err != nil {
			return err
		}
		if msg, ok := WeeklyMessage(w, p.GetString("units")); ok {
			push.ToUser(app, p.GetString("user"), msg)
		}
	}
	return nil
}

// WeeklyMessage is the Monday notification for a week, if anything was logged.
func WeeklyMessage(w Week, units string) (push.Message, bool) {
	if w.DaysLogged == 0 {
		return push.Message{}, false
	}
	parts := []string{
		fmt.Sprintf("%d of 7 days logged", w.DaysLogged),
		fmt.Sprintf("%s kcal a day", thousands(w.AvgKcal)),
	}
	if w.HasWeight {
		parts = append(parts, signedWeight(w.WeightChange, units))
	}
	title := "Your week in Troye"
	switch {
	case w.UnderBudget >= 5:
		title = fmt.Sprintf("Great week: %d days within budget", w.UnderBudget)
	case w.DaysLogged == 7:
		title = "You logged every day last week"
	}
	return push.Message{Title: title, Body: strings.Join(parts, " · "), URL: "/#week", Tag: "weekly-" + w.Start}, true
}

func thousands(n int) string {
	s := fmt.Sprint(n)
	if len(s) <= 3 {
		return s
	}
	return s[:len(s)-3] + "," + s[len(s)-3:]
}

func signedWeight(kg float64, units string) string {
	unit := "kg"
	if units == "imperial" {
		kg, unit = kg/0.45359237, "lb"
	}
	kg = math.Round(kg*10) / 10
	switch {
	case kg < 0:
		return fmt.Sprintf("−%.1f %s", -kg, unit)
	case kg > 0:
		return fmt.Sprintf("+%.1f %s", kg, unit)
	}
	return "weight steady"
}
