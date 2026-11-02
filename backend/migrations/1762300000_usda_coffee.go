package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"

	"github.com/selfhostit/troye/foods"
)

// Espresso and black coffee get names a search finds before branded coffee
// beans and capsules, whose calories are per 100 g of coffee, not of the drink.
func init() {
	m.Register(func(app core.App) error {
		return foods.RenameUSDA(app)
	}, nil)
}
