package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

// Favorites: what someone eats often, such as their breakfast toast, to log in
// one go. Its items are the foods in it, each with the kinds the person has
// had (Gouda or Emmental, a boiled or a fried egg), which one they had last
// time, and whether they left it out. backend/app/favorites.go checks them and
// logs a favorite as an entry with the rest as its add-ons.
func init() {
	m.Register(func(app core.App) error {
		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}
		favorites := core.NewBaseCollection("favorites")
		favorites.Fields.Add(
			&core.RelationField{Name: "user", CollectionId: users.Id, Required: true, MaxSelect: 1, CascadeDelete: true},
			&core.TextField{Name: "name", Required: true, Max: 100},
			// [{choices: [{food, grams, amount}], pick, off}]
			&core.JSONField{Name: "items", Required: true, MaxSize: 20000},
			// how often it was logged, to show the usual ones first
			&core.NumberField{Name: "uses", OnlyInt: true, Min: types.Pointer(0.0)},
			&core.DateField{Name: "last_used"},
			&core.AutodateField{Name: "created", OnCreate: true},
			&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
		)
		favorites.AddIndex("idx_favorites_user", false, "user", "")
		own := types.Pointer(`user = @request.auth.id`)
		favorites.ListRule, favorites.ViewRule, favorites.DeleteRule = own, own, own
		favorites.CreateRule = types.Pointer(`@request.auth.collectionName = "users" && @request.body.user = @request.auth.id`)
		favorites.UpdateRule = types.Pointer(`user = @request.auth.id && (@request.body.user:isset = false || @request.body.user = @request.auth.id)`)
		return app.Save(favorites)
	}, func(app core.App) error {
		favorites, err := app.FindCollectionByNameOrId("favorites")
		if err != nil {
			return err
		}
		return app.Delete(favorites)
	})
}
