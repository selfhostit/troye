package app_test

import (
	"strings"
	"testing"

	"github.com/selfhostit/troye/foods"
)

// The bundled USDA table answers everyday searches with the plain food.
func TestSearchBundledFoods(t *testing.T) {
	a := newApp(t)
	defer a.Cleanup()
	for q, want := range map[string]string{
		"apple":      "Apples, raw",
		"banana":     "Bananas, raw",
		"white rice": "Rice, white,",
		"broccoli":   "Broccoli, raw",
		"cheddar":    "Cheese, cheddar",
	} {
		found, err := foods.Search(a, "", q, 3)
		if err != nil || len(found) == 0 || !strings.HasPrefix(found[0].GetString("name"), want) {
			t.Errorf("%s: %v %v", q, found, err)
		}
		if q == "white rice" && len(found) > 0 && !strings.Contains(found[0].GetString("name"), "cooked") {
			t.Errorf("white rice: %q isn't cooked", found[0].GetString("name"))
		}
	}
}

// A chicken breast search lists plain, short-named chicken breasts, each once,
// before sliced deli chicken, skin on, or fast food.
func TestSearchChickenBreast(t *testing.T) {
	a := newApp(t)
	defer a.Cleanup()
	found, err := foods.Search(a, "", "chicken breast", 30)
	if err != nil || len(found) < 6 {
		t.Fatalf("%v %v", found, err)
	}
	seen := map[string]bool{}
	for i, r := range found {
		name := r.GetString("name")
		if seen[name] {
			t.Errorf("%q twice", name)
		}
		seen[name] = true
		if i < 6 && (!strings.HasPrefix(name, "Chicken breast, ") || strings.Contains(name, "skin") ||
			r.GetString("food_group") != "Poultry Products") {
			t.Errorf("%d: %q (%s)", i, name, r.GetString("food_group"))
		}
	}
	for _, want := range []string{"Chicken breast, grilled", "Chicken breast, roasted", "Chicken breast, raw"} {
		if !seen[want] {
			t.Errorf("no %q", want)
		}
	}
	// the full USDA name still finds it
	found, err = foods.Search(a, "", "broilers or fryers breast roasted", 3)
	if err != nil || len(found) == 0 || found[0].GetString("name") != "Chicken breast, roasted" {
		t.Errorf("full name: %v %v", found, err)
	}
}
