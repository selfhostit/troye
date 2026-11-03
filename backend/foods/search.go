package foods

import (
	"sort"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
	"github.com/selfhostit/food"
)

// how many matches are ranked; the database orders them roughly first
const candidates = 400

// Search finds foods whose names (in English, Dutch or French), other names or
// brand contain every word of q, in any order and without minding accents. The
// best come first: foods the person logs often, their own foods, names that
// start with the words, whole words, the generic tables before branded
// products, and short names.
func Search(app core.App, userID, q string, limit int) ([]*core.Record, error) {
	words := food.Words(q)
	if len(words) == 0 {
		return []*core.Record{}, nil
	}
	found, err := matches(app, userID, words)
	if err != nil {
		return nil, err
	}
	if len(found) == 0 {
		// "apples", "pommes" and "eieren" find "apple", "pomme" and "eier"
		if stems := food.Stemmed(words); stems != nil {
			words = stems
			if found, err = matches(app, userID, words); err != nil {
				return nil, err
			}
		}
	}

	uses := usage(app, userID)
	scores := make(map[string]float64, len(found))
	for _, r := range found {
		scores[r.Id] = score(r, words, userID, uses[r.Id])
	}
	sort.SliceStable(found, func(i, j int) bool {
		if scores[found[i].Id] != scores[found[j].Id] {
			return scores[found[i].Id] > scores[found[j].Id]
		}
		return found[i].GetString("name") < found[j].GetString("name")
	})
	if len(found) > limit {
		found = found[:limit]
	}
	return found, nil
}

func matches(app core.App, userID string, words []string) ([]*core.Record, error) {
	query := app.RecordQuery("foods").
		AndWhere(dbx.Or(dbx.HashExp{"owner": ""}, dbx.HashExp{"owner": userID}))
	for _, w := range words {
		query.AndWhere(dbx.Like("search", w))
	}
	// a rough order, so the best matches are among the candidates: the
	// person's own foods, then names that start with the first word, then the
	// generic tables, then short names
	first := dbx.Params{"first": escapeLike(words[0]) + "%"}
	query.OrderBy(
		"(owner != '') DESC",
		"(search LIKE {:first} ESCAPE '\\') DESC",
		"(source IN ('usda', 'nevo', 'ciqual')) DESC",
		"length(name)",
	).Bind(first).Limit(candidates)
	var found []*core.Record
	err := query.All(&found)
	return found, err
}

func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

// usage counts how often the person logged each food in the last 90 days.
func usage(app core.App, userID string) map[string]int {
	since := types.NowDateTime().Add(-90 * 24 * time.Hour).String()[:10]
	var rows []struct {
		Food string `db:"food"`
		N    int    `db:"n"`
	}
	err := app.DB().
		Select("food", "count(*) AS n").
		From("entries").
		Where(dbx.HashExp{"user": userID}).
		AndWhere(dbx.NewExp("food != '' AND date >= {:since}", dbx.Params{"since": since})).
		GroupBy("food").
		All(&rows)
	uses := make(map[string]int, len(rows))
	if err != nil {
		return uses
	}
	for _, r := range rows {
		uses[r.Food] = r.N
	}
	return uses
}

// score rates how well a food matches the words.
func score(r *core.Record, words []string, userID string, uses int) float64 {
	best := 0.0
	for _, field := range []struct {
		name   string
		weight float64
	}{{"name", 1}, {"name_nl", 1}, {"name_fr", 1}, {"aliases", 0.9}} {
		for _, text := range strings.Split(r.GetString(field.name), ";") {
			s := food.FieldScore(Normalize(text), words)
			// the food itself, before the first comma, is what was asked for:
			// "Rice, white, cooked" rather than "Rice flour, white" for "white rice"
			if head, _, found := strings.Cut(text, ","); found && s > 0 && food.CoversAll(Normalize(head), words) {
				s += 2
			}
			if s *= field.weight; s > best {
				best = s
			}
		}
	}
	// a brand adds to a name match: "boni yoghurt"
	total := best + 0.5*food.FieldScore(Normalize(r.GetString("brand")), words)

	if uses > 0 {
		total += 4 + float64(min(uses, 10))
	}
	switch source := r.GetString("source"); {
	case r.GetString("owner") == userID:
		total += 3
	case source == "usda" || source == "nevo" || source == "ciqual":
		total += 2
	}
	// "Apple, raw" before "Apple pie, homemade, with puff pastry"
	name := Normalize(r.GetString("name"))
	total -= 0.15 * float64(len(strings.Fields(name)))
	// plain foods before processed ones: fresh apples before dried or canned
	// ones, unless that is what was asked for
	if food.HasOtherWord(name, processedWords, words) {
		total -= 0.8
	}
	// meat, fish, grains and pulses are logged cooked: cooked rice before raw
	if cookedGroups[r.GetString("food_group")] && food.HasOtherWord(name, rawWords, words) {
		total -= 0.8
	}
	return total
}

var (
	processedWords = []string{"dried", "dehydrated", "powder", "frozen", "canned", "condensed", "imitation",
		"unprepared", "prepackaged", "deli", "concentrate", "dry"}
	rawWords     = []string{"raw", "uncooked", "dry"}
	cookedGroups = map[string]bool{
		"Poultry Products": true, "Beef Products": true, "Pork Products": true, "Lamb, Veal, and Game Products": true,
		"Finfish and Shellfish Products": true, "Sausages and Luncheon Meats": true,
		"Cereal Grains and Pasta": true, "Legumes and Legume Products": true,
	}
)
