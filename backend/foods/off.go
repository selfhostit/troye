package foods

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/selfhostit/food"
)

// OFFAttribution credits Open Food Facts, whose data is under the Open
// Database Licence.
const OFFAttribution = food.OFFAttribution

// Where products are looked up and the monthly export comes from; tests point
// them elsewhere.
var (
	OpenFoodFactsURL = food.OpenFoodFacts
	OFFExportURL     = food.OFFExportURL
)

// UserAgent says who is asking, as Open Food Facts asks apps to.
var UserAgent = "Troye/dev (self-hosted calorie tracker)"

// ErrUnknownProduct means Open Food Facts doesn't have the barcode, or has it
// without a name or calories.
var ErrUnknownProduct = food.ErrUnknownProduct

var offClient = &http.Client{Timeout: 10 * time.Second}

// FindBarcode returns the food with this barcode: one of the person's own
// foods, a product already kept, or else one looked up in Open Food Facts and
// kept for next time.
func FindBarcode(app core.App, code, userID string) (*core.Record, error) {
	codes := []any{code}
	if len(code) == 12 {
		// a UPC-A code is the EAN-13 code with a leading zero
		codes = append(codes, "0"+code)
	} else if len(code) == 13 && code[0] == '0' {
		codes = append(codes, code[1:])
	}
	var found []*core.Record
	err := app.RecordQuery("foods").
		AndWhere(dbx.In("barcode", codes...)).
		AndWhere(dbx.Or(dbx.HashExp{"owner": ""}, dbx.HashExp{"owner": userID})).
		// your own version of a product first
		OrderBy("owner DESC").
		Limit(1).
		All(&found)
	if err != nil {
		return nil, err
	}
	if len(found) > 0 {
		return found[0], nil
	}

	f, err := food.Client{UserAgent: UserAgent, HTTP: offClient}.Product(context.Background(), OpenFoodFactsURL, code)
	if err != nil {
		return nil, err
	}
	if !f.Nutrition {
		return nil, ErrUnknownProduct
	}
	if _, err := Store(app, "off", []Food{f}); err != nil {
		return nil, err
	}
	return app.FindFirstRecordByFilter("foods", "source = 'off' && source_id = {:code}", dbx.Params{"code": f.SourceID})
}

// SyncOFF downloads the Open Food Facts export (about a gigabyte, read as it
// streams in) and keeps the products sold in Belgium and the Netherlands that
// have a name and calories.
func SyncOFF(app core.App) error {
	client := food.Client{UserAgent: UserAgent, HTTP: &http.Client{Timeout: 3 * time.Hour}}
	body, err := client.Export(context.Background(), OFFExportURL)
	if err != nil {
		return err
	}
	defer body.Close()
	version := time.Now().Format("2006-01-02")
	return importTable(app, "off", version, func(add func(Food) error) error {
		return readOFFExport(body, add)
	})
}

// readOFFExport hands add the products in the export that Troye keeps: sold in
// Belgium or the Netherlands, with calories, and values that can be right.
func readOFFExport(r io.Reader, add func(Food) error) error {
	return food.ReadExport(r, food.BelgiumNetherlands, func(f Food) error {
		if !f.Nutrition || !food.Plausible(f) {
			return nil
		}
		return add(f)
	})
}
