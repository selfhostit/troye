package app

import (
	"errors"
	"math"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/selfhostit/troye/foods"
)

// The food diary: entries get their totals from their food, and add-ons (milk
// in a coffee) follow the entry they belong to. Food search, barcodes and the
// food tables are in backend/foods.

func registerDiary(app core.App) {
	app.OnRecordCreate("entries").BindFunc(func(e *core.RecordEvent) error {
		if err := fitAddOn(e.App, e.Record); err != nil {
			return err
		}
		if err := fillEntry(e.App, e.Record); err != nil {
			return err
		}
		return e.Next()
	})
	app.OnRecordUpdate("entries").BindFunc(func(e *core.RecordEvent) error {
		if err := fitAddOn(e.App, e.Record); err != nil {
			return err
		}
		if err := fillEntry(e.App, e.Record); err != nil {
			return err
		}
		if err := e.Next(); err != nil {
			return err
		}
		return moveAddOns(e.App, e.Record)
	})
}

// LogItem is one thing to put in the diary: a food and how much of it, or
// just a name and calories.
type LogItem struct {
	Food  string
	Grams float64
	// as counted, such as "2 tbsp"; empty for grams
	Amount string
	Name   string
	Kcal   float64
	// AddOn puts it on the first item, like the cheese on a toast
	AddOn bool
}

// logEntries puts several items in one person's diary on a day and meal, such
// as the foods of a checked meal photo. Each goes through the same hooks as an
// entry made one at a time, so totals and names are filled in the same way.
// Run it in a transaction so that the items go in together or not at all.
func logEntries(app core.App, user, date, meal string, items []LogItem) ([]*core.Record, error) {
	entries, err := app.FindCollectionByNameOrId("entries")
	if err != nil {
		return nil, err
	}
	out := make([]*core.Record, 0, len(items))
	for _, it := range items {
		entry := core.NewRecord(entries)
		entry.Set("user", user)
		entry.Set("date", date)
		entry.Set("meal", meal)
		if it.Food != "" {
			entry.Set("food", it.Food)
			entry.Set("grams", it.Grams)
			entry.Set("amount", it.Amount)
		} else {
			entry.Set("name", it.Name)
			entry.Set("kcal", it.Kcal)
		}
		if it.AddOn && len(out) > 0 {
			entry.Set("parent", out[0].Id)
		}
		if err := app.Save(entry); err != nil {
			return nil, err
		}
		out = append(out, entry)
	}
	return out, nil
}

// fitAddOn checks that an add-on hangs off one of the same person's entries,
// one level deep, and puts it on its parent's day and meal.
func fitAddOn(app core.App, entry *core.Record) error {
	parentID := entry.GetString("parent")
	if parentID == "" {
		return nil
	}
	if parentID == entry.Id {
		return errors.New("an entry can't be its own add-on")
	}
	parent, err := app.FindRecordById("entries", parentID)
	if err != nil || parent.GetString("user") != entry.GetString("user") {
		return errors.New("that entry isn't in your diary")
	}
	if parent.GetString("parent") != "" {
		return errors.New("an add-on can't have add-ons of its own")
	}
	if !entry.IsNew() {
		n, err := app.CountRecords("entries", dbx.HashExp{"parent": entry.Id})
		if err != nil {
			return err
		}
		if n > 0 {
			return errors.New("an entry with add-ons can't be an add-on")
		}
	}
	entry.Set("date", parent.GetString("date"))
	entry.Set("meal", parent.GetString("meal"))
	return nil
}

// moveAddOns keeps an entry's add-ons on its day and meal when it moves.
func moveAddOns(app core.App, entry *core.Record) error {
	addOns, err := app.FindAllRecords("entries", dbx.HashExp{"parent": entry.Id})
	if err != nil {
		return err
	}
	for _, a := range addOns {
		if a.GetString("date") == entry.GetString("date") && a.GetString("meal") == entry.GetString("meal") {
			continue
		}
		a.Set("date", entry.GetString("date"))
		a.Set("meal", entry.GetString("meal"))
		if err := app.Save(a); err != nil {
			return err
		}
	}
	return nil
}

// fillEntry works out an entry's totals from its food and grams. An entry
// without a food is a quick add, and keeps the calories it was given.
func fillEntry(app core.App, entry *core.Record) error {
	foodID := entry.GetString("food")
	if foodID == "" {
		return nil
	}
	food, err := app.FindRecordById("foods", foodID)
	if err != nil {
		return errors.New("that food no longer exists")
	}
	if owner := food.GetString("owner"); owner != "" && owner != entry.GetString("user") {
		return errors.New("that food belongs to someone else")
	}
	if entry.GetString("name") == "" {
		lang := ""
		if p, err := app.FindFirstRecordByFilter("profiles", "user = {:user}", dbx.Params{"user": entry.GetString("user")}); err == nil {
			lang = p.GetString("food_language")
		}
		entry.Set("name", foods.Name(food, lang))
	}
	factor := entry.GetFloat("grams") / 100
	for _, field := range []string{"kcal", "protein", "carbs", "fat", "fiber", "sugar", "salt"} {
		v := food.GetFloat(field) * factor
		switch field {
		case "kcal":
			v = math.Round(v)
		case "salt":
			v = math.Round(v*100) / 100
		default:
			v = math.Round(v*10) / 10
		}
		entry.Set(field, v)
	}
	return nil
}
