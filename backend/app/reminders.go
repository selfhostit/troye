package app

import (
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/selfhostit/troye/jobs"
	"github.com/selfhostit/troye/reminders"
)

// Reminders to log (see backend/reminders): the evening one at 21:00, and a
// check every hour for anyone who hasn't logged in 8 hours of eating time.
func registerReminders(app core.App) {
	jobs.MustAdd(app, "evening-reminder", "0 21 * * *", func(app core.App) error {
		return reminders.Send(app, reminders.Evening, time.Now())
	})
	jobs.MustAdd(app, "gap-reminder", "0 * * * *", func(app core.App) error {
		return reminders.Send(app, reminders.Gap, time.Now())
	})
}
