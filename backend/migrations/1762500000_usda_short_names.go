package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"

	"github.com/selfhostit/troye/foods"
)

// USDA foods get the names people call them, "Chicken breast, grilled" for
// "Chicken, broiler or fryers, breast, skinless, boneless, meat only, cooked,
// grilled", so search results read at a glance. The full names stay
// searchable as aliases.
func init() {
	m.Register(func(app core.App) error {
		return foods.RefreshUSDA(app)
	}, nil)
}
