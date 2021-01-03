package reports

import (
	
	"github.com/nsip/dev-nrt/codeframe"
	"github.com/nsip/dev-nrt/records"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

type ItemDetail struct {
	baseReport // embed common setup capability
	cfh        codeframe.Helper
}

//
// Specialist report processor inserted into pipeline for
// item-printing reports.
// Does not write any output itself but adds the full
// item definition to the calculated fields
//
func ItemDetailReport(cfh codeframe.Helper) *ItemDetail {

	r := ItemDetail{cfh: cfh}
	r.initialise("./config/internal/ItemDetail.toml")
	r.printStatus()

	return &r

}

//
// implement the EventPipe interface, core work of the
// report engine.
//
func (r *ItemDetail) ProcessEventRecords(in chan *records.EventOrientedRecord) chan *records.EventOrientedRecord {

	out := make(chan *records.EventOrientedRecord)
	go func() {
		defer close(out)
		// does not open csv file
		for eor := range in {

			if !r.config.activated { // only process if activated
				out <- eor
				continue
			}

			//
			// generate any calculated fields required
			//
			eor.CalculatedFields = r.calculateFields(eor)

			out <- eor

		}

	}()
	return out
}

//
// generates a block of json that can be added to the
// record containing values that are not in the original data
//
//
func (r *ItemDetail) calculateFields(eor *records.EventOrientedRecord) []byte {

	itemRefId := eor.GetValueString("CalculatedFields.ItemResponse.NAPTestItemRefId")
	itemBytes := r.cfh.GetItem(itemRefId)
	itemDetail := gjson.GetBytes(itemBytes, "NAPTestItem")
	json, _ := sjson.SetRawBytes(eor.CalculatedFields, "CalculatedFields.NAPTestItem", []byte(itemDetail.String()))

	return json
}
