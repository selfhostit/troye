package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

// The burn measured from the diary and the weigh-ins (backend/insights),
// once the person chooses to plan with it instead of the formula's estimate.
// 0 means the formula.
func init() {
	m.Register(func(app core.App) error {
		profiles, err := app.FindCollectionByNameOrId("profiles")
		if err != nil {
			return err
		}
		profiles.Fields.Add(&core.NumberField{Name: "measured_tdee", OnlyInt: true, Min: types.Pointer(0.0), Max: types.Pointer(6000.0)})
		return app.Save(profiles)
	}, func(app core.App) error {
		profiles, err := app.FindCollectionByNameOrId("profiles")
		if err != nil {
			return err
		}
		profiles.Fields.RemoveByName("measured_tdee")
		return app.Save(profiles)
	})
}
