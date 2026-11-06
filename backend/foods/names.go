package foods

import "strings"

// TidyUSDA shortens a USDA name to what a person would call the food, dropping
// the parts that only matter to a lab: "Chicken, broilers or fryers, breast,
// skinless, boneless, meat only, cooked, grilled" becomes "Chicken breast,
// grilled", and "meat and skin" becomes "with skin". The full name stays
// searchable as an alias.
func TidyUSDA(name string) string {
	var parts []string
	raw := strings.Split(name, ",")
	for i, p := range raw {
		p = strings.TrimSpace(p)
		l := strings.ToLower(p)
		switch {
		case p == "" || usdaFiller[l]:
			continue
		case l == "meat and skin":
			p = "with skin"
		// "cooked, roasted" is roasted
		case l == "cooked" && i+1 < len(raw) && cookingMethods[strings.ToLower(strings.TrimSpace(raw[i+1]))]:
			continue
		}
		parts = append(parts, p)
	}
	if len(parts) == 0 {
		return name
	}
	// "Chicken, rotisserie, breast" is a chicken breast, rotisserie
	if meats[strings.ToLower(parts[0])] {
		for i := 1; i < len(parts); i++ {
			if cut := strings.ToLower(parts[i]); cuts[cut] {
				parts[0] += " " + cut
				parts = append(parts[:i], parts[i+1:]...)
				break
			}
		}
	}
	return strings.Join(parts, ", ")
}

var (
	usdaFiller = setOf("broilers or fryers", "broiler or fryers", "broilers", "broiler", "all classes",
		"meat only", "boneless", "skinless", "boneless and skinless", "variety meats and by-products")
	cookingMethods = setOf("roasted", "grilled", "braised", "stewed", "fried", "pan-fried", "simmered",
		"boiled", "baked", "broiled", "pan-browned", "microwaved", "steamed", "poached", "scrambled",
		"moist heat", "dry heat", "rotisserie")
	meats = setOf("chicken", "turkey", "duck", "goose", "beef", "pork", "lamb", "veal")
	cuts  = setOf("breast", "thigh", "drumstick", "wing", "leg", "back", "liver", "heart", "gizzard", "feet",
		"tongue", "kidneys")
)

func setOf(words ...string) map[string]bool {
	m := make(map[string]bool, len(words))
	for _, w := range words {
		m[w] = true
	}
	return m
}
