package reports

import (
	"github.com/nsip/dev-nrt/records"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

type ItemExtractor struct {
	baseReport // embed common setup capability
}

//
// Specialist report processor inserted into pipeline for
// item-printing reports.
// Does not write any output itself but splits each record
// into a duplicate containing just a sngle item response
//
func ItemExtractorReport() *ItemExtractor {

	r := ItemExtractor{}
	r.initialise("./config/ItemExtractor.toml")
	r.printStatus()

	return &r

}

//
// implement the EventPipe interface, core work of the
// report engine.
//
func (r *ItemExtractor) ProcessEventRecords(in chan *records.EventOrientedRecord) chan *records.EventOrientedRecord {

	out := make(chan *records.EventOrientedRecord)
	go func() {
		defer close(out)
		// does not open csv file
		var itemResponses [][]byte
		var eorItem *records.EventOrientedRecord
		for eor := range in {
			if r.config.activated { // only process if active

				itemResponses = getItemResponses(eor)
				for _, ir := range itemResponses {
					eorItem = &records.EventOrientedRecord{
						SchoolInfo:          eor.SchoolInfo,
						StudentPersonal:     eor.StudentPersonal,
						NAPTest:             eor.NAPTest,
						NAPEventStudentLink: eor.NAPEventStudentLink,
						CalculatedFields:    addItemResponse(eor.CalculatedFields, ir),
					}

					out <- eorItem
				}
			}
		}
	}()
	return out
}

//
// add item response to the calc fields block
//
func addItemResponse(calculatedFields []byte, itemResponse []byte) []byte {

	json, _ := sjson.SetRawBytes(calculatedFields, "CalculatedFields.ItemResponse", itemResponse)
	return json

}

//
// extract the item responses from the response-set
//
func getItemResponses(eor *records.EventOrientedRecord) [][]byte {

	responses := make([][]byte, 0)

	if !eor.HasNAPStudentResponseSet {
		return responses
	}

	// need to add testlet ids as top-level elements...
	// iterate with sjson block to add to responses

	tl := gjson.GetBytes(eor.NAPStudentResponseSet, "NAPStudentResponseSet.TestletList.Testlet.#.ItemResponseList.ItemResponse")
	tl.ForEach(func(key, value gjson.Result) bool {
		value.ForEach(func(key, value gjson.Result) bool {
			responses = append(responses, []byte(value.String()))
			return true // keep iterating
		})
		return true // keep iterating
	})

	return responses
}

//
// generates a block of json that can be added to the
// record containing values that are not in the original data
//
//
func (r *ItemExtractor) calculateFields(eor *records.EventOrientedRecord) []byte {

	return eor.CalculatedFields
}
