package foods

import (
	"github.com/pocketbase/pocketbase/core"
	"github.com/selfhostit/food"
)

// NEVOAttribution is the citation RIVM asks for wherever NEVO data is used.
const NEVOAttribution = food.NEVOAttribution

// FormatError means a file isn't the table it should be.
type FormatError = food.FormatError

// ImportNEVO reads a NEVO download (the CSV or Excel file, or the zip they
// came in) and stores its foods.
func ImportNEVO(app core.App, filename string, data []byte) error {
	t, err := food.ReadNEVO(filename, data)
	if err != nil {
		return err
	}
	return importTable(app, "nevo", t.Version, each(t.Foods))
}
