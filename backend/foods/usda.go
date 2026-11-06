package foods

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"
	"github.com/selfhostit/food"
)

// USDAAttribution is shown with the USDA foods. The data is in the public
// domain; USDA asks to be cited.
const USDAAttribution = food.USDAAttribution

// Table is a food table: its edition and its foods.
type Table = food.Table

// BundledUSDA reads the USDA table that ships with the food module.
func BundledUSDA() (Table, error) { return food.BundledUSDA() }

// RenameUSDA updates the foods the food module renames, "Espresso" for
// "Beverages, coffee, brewed, espresso, restaurant-prepared", in a table
// imported before they had those names.
func RenameUSDA(app core.App) error {
	t, err := BundledUSDA()
	if err != nil {
		return err
	}
	_, err = Store(app, "usda", food.USDARenamed(t))
	return err
}

// RefreshUSDA stores the whole bundled USDA table again, over the foods it
// already has, for when Troye changes how it stores them, such as their names.
func RefreshUSDA(app core.App) error {
	t, err := BundledUSDA()
	if err != nil {
		return err
	}
	_, err = Store(app, "usda", t.Foods)
	return err
}

// ImportUSDA stores the bundled USDA table, unless that edition is already in.
func ImportUSDA(app core.App) error {
	t, err := BundledUSDA()
	if err != nil {
		return fmt.Errorf("reading the bundled USDA table: %w", err)
	}
	if status, err := app.FindFirstRecordByData(TablesCollection, "name", "usda"); err == nil &&
		status.GetString("version") == t.Version && status.GetInt("foods") > 0 {
		return nil
	}
	return importTable(app, "usda", t.Version, each(t.Foods))
}
