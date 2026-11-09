package app

import (
	"net/http"
	"strconv"
	"time"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

	"github.com/selfhostit/troye/daily"
	"github.com/selfhostit/troye/insights"
)

// The progress screen's nutrition: GET /api/app/insights?days=30 returns the
// last days with missing ones estimated, 7-day rolling averages and the
// warnings for the week (see backend/insights).

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

// InsightReport analyses a person's last `days` days up to today.
func InsightReport(app core.App, user string, today time.Time, days int) (insights.Report, error) {
	from, to := insights.Range(today, days)
	s, err := daily.Load(app, user, from, to)
	if err != nil {
		return insights.Report{}, err
	}
	return insights.Analyze(s, today.Format(time.DateOnly), days), nil
}
