package foods

import (
	"strings"
	"testing"
)

// Reading the tables is tested in github.com/selfhostit/food; this checks
// which Open Food Facts products Troye keeps.

func TestReadOFFExport(t *testing.T) {
	export := strings.Join([]string{
		"code\tproduct_name\tgeneric_name\tbrands\tcountries_tags\tenergy-kcal_100g\tenergy_100g\tproteins_100g\tcarbohydrates_100g\tfat_100g\tsugars_100g\tsalt_100g\tsodium_100g\tserving_quantity\tserving_size",
		"5410000000001\tSpeculoos\t\tLotus,Lotus Bakeries\ten:belgium,en:france\t484\t\t4.9\t72.6\t19\t38\t0.9\t\t6.25\t1 biscuit",
		"8710000000001\t\tKarnemelk\tAH\ten:netherlands\t\t159\t3.5\t4\t0.5\t\t\t0.04\t\t",
		"3000000000001\tCroissant\t\t\ten:france\t400\t\t8\t45\t21\t\t1\t\t\t",
		"5410000000002\tNo calories\t\t\ten:belgium\t\t\t\t\t\t\t\t\t\t",
		"5410000000003\tImpossible\t\t\ten:belgium\t2000\t\t\t\t\t\t\t\t\t",
		"short\tline",
	}, "\n")
	var got []Food
	err := readOFFExport(strings.NewReader(export), func(f Food) error {
		got = append(got, f)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("kept %+v", got)
	}
	lotus, milk := got[0], got[1]
	if lotus.Barcode != "5410000000001" || lotus.Name != "Speculoos" || lotus.Brand != "Lotus" || lotus.Kcal != 484 || lotus.ServingG != 6.25 {
		t.Errorf("speculoos %+v", lotus)
	}
	// the generic name stands in, kJ become kcal and sodium becomes salt
	if milk.Name != "Karnemelk" || milk.Kcal < 37.9 || milk.Kcal > 38.1 || milk.Salt != 0.1 {
		t.Errorf("karnemelk %+v", milk)
	}
}
