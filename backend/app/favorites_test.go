package app_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/pocketbase/dbx"
)

func TestFavorites(t *testing.T) {
	a := newApp(t)
	defer a.Cleanup()
	ada, adaToken := addUser(t, a, "ada@example.com", "owner")
	bob, bobToken := addUser(t, a, "bob@example.com", "member")
	bread := addFood(t, a, "Bread", "", 250, 9)
	gouda := addFood(t, a, "Gouda", "", 356, 25)
	emmental := addFood(t, a, "Emmental", "", 380, 28)
	boiled := addFood(t, a, "Egg, boiled", "", 155, 13)
	bobsJam := addFood(t, a, "Bob's jam", bob.Id, 250, 0)

	items := func(cheesePick int, eggOff bool) string {
		return fmt.Sprintf(`[
			{"choices":[{"food":%q,"grams":35,"amount":"1 slice"}],"pick":0},
			{"choices":[{"food":%q,"grams":20},{"food":%q,"grams":25}],"pick":%d},
			{"choices":[{"food":%q,"grams":50,"amount":"1 egg"}],"pick":0,"off":%t}
		]`, bread.Id, gouda.Id, emmental.Id, cheesePick, boiled.Id, eggOff)
	}
	create := func(token, user, items string) (int, map[string]any) {
		return call(t, a, "POST", "/api/collections/favorites/records", token,
			fmt.Sprintf(`{"user":%q,"name":"Breakfast toast","items":%s}`, user, items))
	}

	code, fav := create(adaToken, ada.Id, items(0, false))
	if code != 200 {
		t.Fatalf("create: %d %v", code, fav)
	}
	id := fav["id"].(string)

	// foods that aren't there or aren't yours, a pick that isn't a choice, and no amount are refused
	for name, bad := range map[string]string{
		"someone else's food": fmt.Sprintf(`[{"choices":[{"food":%q,"grams":20}],"pick":0}]`, bobsJam.Id),
		"a missing food":      `[{"choices":[{"food":"nope","grams":20}],"pick":0}]`,
		"a pick out of range": fmt.Sprintf(`[{"choices":[{"food":%q,"grams":20}],"pick":1}]`, bread.Id),
		"no amount":           fmt.Sprintf(`[{"choices":[{"food":%q,"grams":0}],"pick":0}]`, bread.Id),
		"no foods":            `[]`,
	} {
		if code, _ := create(adaToken, ada.Id, bad); code != 400 {
			t.Errorf("%s: got %d", name, code)
		}
	}
	// and nobody makes one for someone else, or sees or logs someone else's
	if code, _ := create(bobToken, ada.Id, items(0, false)); code != 400 {
		t.Errorf("made a favorite for someone else: %d", code)
	}
	if code, _ := call(t, a, "GET", "/api/collections/favorites/records/"+id, bobToken, ""); code != 404 {
		t.Errorf("saw someone else's favorite: %d", code)
	}
	if code, _ := call(t, a, "POST", "/api/app/favorites/"+id+"/log", bobToken,
		`{"date":"2026-11-10","meal":"breakfast","items":`+items(0, false)+`}`); code != 404 {
		t.Errorf("logged someone else's favorite: %d", code)
	}

	// logging it with Emmental and no egg: the bread with the cheese on it
	code, out := call(t, a, "POST", "/api/app/favorites/"+id+"/log", adaToken,
		`{"date":"2026-11-10","meal":"breakfast","items":`+items(1, true)+`}`)
	if code != 200 {
		t.Fatalf("log: %d %v", code, out)
	}
	logged, err := a.FindAllRecords("entries", dbx.HashExp{"user": ada.Id})
	if err != nil || len(logged) != 2 {
		t.Fatalf("logged %d entries (%v)", len(logged), err)
	}
	base, cheese := logged[0], logged[1]
	if base.GetString("parent") != "" {
		base, cheese = cheese, base
	}
	if base.GetString("food") != bread.Id || base.GetString("amount") != "1 slice" || base.GetFloat("kcal") != 88 ||
		base.GetString("meal") != "breakfast" || base.GetString("date") != "2026-11-10" {
		t.Errorf("base = %v", base.FieldsData())
	}
	if cheese.GetString("food") != emmental.Id || cheese.GetString("parent") != base.Id || cheese.GetFloat("kcal") != 95 {
		t.Errorf("add-on = %v", cheese.FieldsData())
	}

	// the picks are kept for next time
	saved, _ := a.FindRecordById("favorites", id)
	var kept []struct {
		Pick int  `json:"pick"`
		Off  bool `json:"off"`
	}
	if err := json.Unmarshal([]byte(saved.GetString("items")), &kept); err != nil {
		t.Fatal(err)
	}
	if kept[1].Pick != 1 || !kept[2].Off || saved.GetInt("uses") != 1 || saved.GetDateTime("last_used").IsZero() {
		t.Errorf("after logging: items %s, uses %d", saved.GetString("items"), saved.GetInt("uses"))
	}

	// leaving everything out logs nothing
	all := fmt.Sprintf(`[{"choices":[{"food":%q,"grams":35}],"pick":0,"off":true}]`, bread.Id)
	if code, _ := call(t, a, "POST", "/api/app/favorites/"+id+"/log", adaToken,
		`{"date":"2026-11-10","meal":"breakfast","items":`+all+`}`); code != 400 {
		t.Errorf("logged a favorite with everything left out: %d", code)
	}
}
