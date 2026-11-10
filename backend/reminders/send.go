package reminders

import (
	"database/sql"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/selfhostit/troye/daily"
	"github.com/selfhostit/troye/push"
)

// Collection records each reminder sent: a kind goes to a person at most
// once a day. Only the server reads it.
const Collection = "reminders"

// Send sends a kind of reminder to everyone it is due for now, and records it.
func Send(app core.App, k Kind, now time.Time) error {
	profiles, err := app.FindRecordsByFilter("profiles", "onboarded = true", "", 0, 0)
	if err != nil {
		return err
	}
	day := now.Format(time.DateOnly)
	for _, profile := range profiles {
		p, err := person(app, profile, k, now)
		if err != nil {
			return err
		}
		msg, ok := Due(k, p, now)
		if !ok {
			continue
		}
		// recorded first, so a run that fails halfway never sends twice
		if err := record(app, profile.GetString("user"), k, day); err != nil {
			return err
		}
		msg.Tag += "-" + day
		push.ToUser(app, profile.GetString("user"), msg)
	}
	return nil
}

// person gathers what the policy needs about a profile's owner.
func person(app core.App, profile *core.Record, k Kind, now time.Time) (Person, error) {
	user := profile.GetString("user")
	p := Person{
		Window:     WindowFor(profile.GetString("fasting"), profile.GetString("eating_start")),
		FastingDay: slices.Contains(profile.GetStringSlice("fasting_days"), strings.ToLower(now.Weekday().String()[:3])),
	}
	for _, off := range profile.GetStringSlice("reminders_off") {
		p.Off = append(p.Off, Kind(off))
	}

	_, err := app.FindFirstRecordByFilter(Collection, "user = {:user} && kind = {:kind} && day = {:day}",
		dbx.Params{"user": user, "kind": string(k), "day": now.Format(time.DateOnly)})
	switch {
	case err == nil:
		p.Sent = true
		return p, nil
	case !errors.Is(err, sql.ErrNoRows):
		return p, err
	}

	s, err := daily.Load(app, user, now, now)
	if err != nil {
		return p, err
	}
	p.Today = s.Days[0]

	last, err := app.FindRecordsByFilter("entries", "user = {:user}", "-created", 1, 0, dbx.Params{"user": user})
	if err != nil {
		return p, err
	}
	if len(last) == 1 {
		p.LastEntry = last[0].GetDateTime("created").Time().In(now.Location())
	}
	return p, nil
}

func record(app core.App, user string, k Kind, day string) error {
	col, err := app.FindCollectionByNameOrId(Collection)
	if err != nil {
		return err
	}
	r := core.NewRecord(col)
	r.Set("user", user)
	r.Set("kind", string(k))
	r.Set("day", day)
	return app.Save(r)
}
