package foods

import "testing"

func TestTidyUSDA(t *testing.T) {
	for name, want := range map[string]string{
		"Chicken, broiler or fryers, breast, skinless, boneless, meat only, cooked, grilled": "Chicken breast, grilled",
		"Chicken, broilers or fryers, breast, meat and skin, cooked, roasted":                "Chicken breast, with skin, roasted",
		"Chicken, broiler, rotisserie, BBQ, breast, meat only":                               "Chicken breast, rotisserie, BBQ",
		"Chicken, breast, boneless, skinless, raw":                                           "Chicken breast, raw",
		"Beef, variety meats and by-products, liver, cooked, braised":                        "Beef liver, braised",
		"Rice, white, long-grain, regular, enriched, cooked":                                 "Rice, white, long-grain, regular, enriched, cooked",
		"Apples, raw, with skin":                                                             "Apples, raw, with skin",
		"Espresso":                                                                           "Espresso",
	} {
		if got := TidyUSDA(name); got != want {
			t.Errorf("%q: %q, want %q", name, got, want)
		}
	}
}
