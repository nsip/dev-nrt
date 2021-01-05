package reports

import (
	"fmt"

	"github.com/nsip/dev-nrt/codeframe"
	"github.com/nsip/dev-nrt/records"
	"github.com/tidwall/sjson"
)

type ItemTestLink struct {
	cfh        codeframe.Helper
	baseReport // embed common setup capability
}

//
// Looks up the tests that use this item,
// in the case of items being used by multiple tests
// emits multiple copies of the record, one for each test allocation
//
func ItemTestLinkReport(cfh codeframe.Helper) *ItemTestLink {

	r := ItemTestLink{cfh: cfh}
	r.initialise("./config/internal/ItemTestLink.toml")
	r.printStatus()

	return &r

}

//
// implement the EventPipe interface, core work of the
// report engine.
//
func (r *ItemTestLink) ProcessCodeframeRecords(in chan *records.CodeframeRecord) chan *records.CodeframeRecord {

	out := make(chan *records.CodeframeRecord)
	go func() {
		defer close(out)

		for cfr := range in {
			if !r.config.activated { // only process if activated
				out <- cfr
				continue
			}

			if cfr.RecordType != "NAPTestItem" { // only deal with test items
				out <- cfr
				continue
			}

			//
			// check if substitute
			//

			//
			// get all tests associated with this item
			//
			for _, testid := range r.cfh.GetTestsForItem(cfr.RefId()) {
				fmt.Println("\tTest-id:", testid)
				calcf, _ := sjson.SetBytes([]byte{}, "CalculatedFields.NAPTestRefId", testid)
				newcfr := records.CodeframeRecord{
					RecordType:       cfr.RecordType,
					Json:             cfr.Json,
					CalculatedFields: calcf,
				}
				fmt.Printf("\n%s\n", string(newcfr.CalculatedFields))
				out <- &newcfr
			}

			// out <- cfr
		}
	}()
	return out
}
