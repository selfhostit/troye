package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

// Favorites: what someone eats often, logged in one go. Each item is a food
// with the kinds the person has had of it (Gouda or Emmental), the one they
// picked last time, and whether they left it out. Logging a favorite saves the
// picks for next time and puts the first item in the diary with the rest as
// its add-ons, so they show, move and go together like milk in a coffee.

const (
	maxFavoriteItems = 20
	maxChoices       = 8
)

// FavoriteItem is one food in a favorite, in the kinds the person has had.
type FavoriteItem struct {
	Choices []FavoriteChoice `json:"choices"`
	// the choice they had last time
	Pick int `json:"pick"`
	// left out last time
	Off bool `json:"off,omitempty"`
}

// FavoriteChoice is one kind of an item: a food and how much of it.
type FavoriteChoice struct {
	Food  string  `json:"food"`
	Grams float64 `json:"grams"`
	// as counted, such as "1 slice"; empty for grams
	Amount string `json:"amount,omitempty"`
}

var isoDay = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

func registerFavorites(app core.App) {
	check := func(e *core.RecordEvent) error {
		if _, err := checkFavorite(e.App, e.Record.GetString("user"), e.Record.Get("items")); err != nil {
			return apis.NewBadRequestError(err.Error(), nil)
		}
		return e.Next()
	}
	app.OnRecordCreate("favorites").BindFunc(check)
	app.OnRecordUpdate("favorites").BindFunc(check)

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.POST("/api/app/favorites/{id}/log", logFavorite).Bind(apis.RequireAuth("users"))
		return se.Next()
	})
}

// checkFavorite reads a favorite's items and checks that each food is one the
// person can log, with an amount, and that something is left in.
func checkFavorite(app core.App, user string, raw any) ([]FavoriteItem, error) {
	var items []FavoriteItem
	data, err := json.Marshal(raw)
	if err == nil {
		err = json.Unmarshal(data, &items)
	}
	if err != nil {
		return nil, errors.New("couldn't read the foods in this favorite")
	}
	if len(items) == 0 || len(items) > maxFavoriteItems {
		return nil, fmt.Errorf("a favorite has 1 to %d foods", maxFavoriteItems)
	}
	for _, it := range items {
		if len(it.Choices) == 0 || len(it.Choices) > maxChoices {
			return nil, fmt.Errorf("each food in a favorite has 1 to %d kinds", maxChoices)
		}
		if it.Pick < 0 || it.Pick >= len(it.Choices) {
			return nil, errors.New("pick one of the kinds of each food")
		}
		for _, c := range it.Choices {
			if c.Grams <= 0 || c.Grams > 5000 || len(c.Amount) > 50 {
				return nil, errors.New("each food in a favorite needs an amount")
			}
			food, err := app.FindRecordById("foods", c.Food)
			if err != nil {
				return nil, errors.New("a food in this favorite no longer exists")
			}
			if owner := food.GetString("owner"); owner != "" && owner != user {
				return nil, errors.New("that food belongs to someone else")
			}
		}
	}
	return items, nil
}

// favoriteEntries are the diary items for a favorite as picked: the first food
// left in, then the others as its add-ons.
func favoriteEntries(items []FavoriteItem) []LogItem {
	var out []LogItem
	for _, it := range items {
		if it.Off {
			continue
		}
		c := it.Choices[it.Pick]
		out = append(out, LogItem{Food: c.Food, Grams: c.Grams, Amount: c.Amount, AddOn: len(out) > 0})
	}
	return out
}

// logFavorite logs a favorite as picked this time, and keeps the picks (and
// any kind added this time) for next time. It answers with the new entries.
func logFavorite(e *core.RequestEvent) error {
	var body struct {
		Date  string         `json:"date"`
		Meal  string         `json:"meal"`
		Items []FavoriteItem `json:"items"`
	}
	if err := e.BindBody(&body); err != nil {
		return e.BadRequestError("Could not read what to log.", err)
	}
	fav, err := e.App.FindRecordById("favorites", e.Request.PathValue("id"))
	if err != nil || fav.GetString("user") != e.Auth.Id {
		return e.NotFoundError("", err)
	}
	if !isoDay.MatchString(body.Date) {
		return e.BadRequestError("Which day is it for?", nil)
	}
	items, err := checkFavorite(e.App, e.Auth.Id, body.Items)
	if err != nil {
		return e.BadRequestError(err.Error(), nil)
	}
	log := favoriteEntries(items)
	if len(log) == 0 {
		return e.BadRequestError("Leave at least one food in.", nil)
	}

	var logged []*core.Record
	err = e.App.RunInTransaction(func(tx core.App) error {
		var err error
		if logged, err = logEntries(tx, e.Auth.Id, body.Date, body.Meal, log); err != nil {
			return err
		}
		fav.Set("items", items)
		fav.Set("uses", fav.GetInt("uses")+1)
		fav.Set("last_used", types.NowDateTime())
		return tx.Save(fav)
	})
	if err != nil {
		return e.BadRequestError("Couldn't log it. Check the foods and amounts.", err)
	}
	if errs := e.App.ExpandRecords(logged, []string{"food"}, nil); len(errs) > 0 {
		e.App.Logger().Warn("favorites: expanding foods", "errors", errs)
	}
	return e.JSON(http.StatusOK, logged)
}
