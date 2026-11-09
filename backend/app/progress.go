package app

import (
	"net/http"
	"strconv"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

	"github.com/selfhostit/troye/daily"
	"github.com/selfhostit/troye/insights"
)

// The progress screen's nutrition: GET /api/app/insights?days=30 returns the
// last days with missing ones estimated, 7-day rolling averages, the
// warnings for the week and the longer patterns (see backend/insights), and
// what the measured burn would make the budget. Choosing that budget is a
// profile update setting measured_tdee, which the plan then uses.

func registerProgress(app core.App) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.GET("/api/app/insights", insightsRoute).Bind(apis.RequireAuth("users"))
		return se.Next()
	})
}

// the most days a report covers
const maxInsightDays = 182

func insightsRoute(e *core.RequestEvent) error {
	days := 30
	if s := e.Request.URL.Query().Get("days"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n < 7 || n > maxInsightDays {
			return e.BadRequestError("Ask for 7 to 182 days.", nil)
		}
		days = n
	}
	r, err := InsightReport(e.App, e.Auth.Id, time.Now(), days)
	if err != nil {
		return e.InternalServerError("", err)
	}
	return e.JSON(http.StatusOK, r)
}

// Progress is the insights report with what the measured burn means for the plan.
type Progress struct {
	insights.Report
	Budget *BudgetAdvice `json:"budget_advice"`
}

// BudgetAdvice compares the plan's budget with one built on the measured burn.
type BudgetAdvice struct {
	// FormulaTDEE is the burn the formula estimates from the plan's answers.
	FormulaTDEE int `json:"formula_tdee"`
	// MeasuredTDEE is the burn the plan uses now, 0 while it uses the formula.
	MeasuredTDEE int `json:"measured_tdee"`
	// Suggested is the budget the measured burn gives, at the plan's pace.
	Suggested int `json:"suggested"`
	Current   int `json:"current"`
}

// InsightReport analyses a person's last `days` days up to today.
func InsightReport(app core.App, user string, today time.Time, days int) (Progress, error) {
	from, to := insights.Range(today, days)
	s, err := daily.Load(app, user, from, to)
	if err != nil {
		return Progress{}, err
	}
	p := Progress{Report: insights.Analyze(s, today.Format(time.DateOnly), days)}
	if p.Burn == nil {
		return p, nil
	}
	profile, err := app.FindFirstRecordByFilter("profiles", "user = {:user}", dbx.Params{"user": user})
	if err != nil {
		return p, nil // no plan to compare with
	}
	p.Budget = budgetAdvice(planFor(profile), p.Burn.TDEE, profile.GetInt("calorie_budget"), today)
	return p, nil
}

// budgetAdvice works the plan out with the formula and with the measured
// burn; nil when either can't be worked out.
func budgetAdvice(in PlanInput, measured, current int, today time.Time) *BudgetAdvice {
	inUse := in.MeasuredTDEE
	in.MeasuredTDEE = 0
	formula, err := ComputePlan(in, today)
	if err != nil {
		return nil
	}
	in.MeasuredTDEE = measured
	suggested, err := ComputePlan(in, today)
	if err != nil {
		return nil
	}
	return &BudgetAdvice{FormulaTDEE: formula.TDEE, MeasuredTDEE: inUse, Suggested: suggested.Budget, Current: current}
}
