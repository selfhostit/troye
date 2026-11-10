package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Reminders to log (backend/reminders): each person can turn kinds off on
// their profile, and the reminders collection records what was sent, so a
// kind goes out at most once a day. Only the server reads it.
func init() {
	kinds := []string{"evening", "gap"}
	m.Register(func(app core.App) error {
		profiles, err := app.FindCollectionByNameOrId("profiles")
		if err != nil {
			return err
		}
		profiles.Fields.Add(&core.SelectField{Name: "reminders_off", Values: kinds, MaxSelect: len(kinds)})
		if err := app.Save(profiles); err != nil {
			return err
		}

		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}
		reminders := core.NewBaseCollection("reminders")
		reminders.Fields.Add(
			&core.RelationField{Name: "user", CollectionId: users.Id, Required: true, MaxSelect: 1, CascadeDelete: true},
			&core.SelectField{Name: "kind", Required: true, Values: kinds, MaxSelect: 1},
			&core.TextField{Name: "day", Required: true, Pattern: `^\d{4}-\d{2}-\d{2}$`},
			&core.AutodateField{Name: "created", OnCreate: true},
		)
		reminders.AddIndex("idx_reminders_user_kind_day", true, "user, kind, day", "")
		return app.Save(reminders)
	}, func(app core.App) error {
		reminders, err := app.FindCollectionByNameOrId("reminders")
		if err == nil {
			if err := app.Delete(reminders); err != nil {
				return err
			}
		}
		profiles, err := app.FindCollectionByNameOrId("profiles")
		if err != nil {
			return err
		}
		profiles.Fields.RemoveByName("reminders_off")
		return app.Save(profiles)
	})
}
