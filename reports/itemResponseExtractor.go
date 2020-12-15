package reports

import (
	"fmt"

	"github.com/nsip/dev-nrt/records"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

type ItemResponseExtractor struct {
	baseReport // embed common setup capability
}

//
// Specialist report processor inserted into pipeline for
// item-printing reports.
// Does not write any output itself but splits each record
// into a duplicate containing just a single item response
//
func ItemResponseExtractorReport() *ItemResponseExtractor {

	r := ItemResponseExtractor{}
	r.initialise("./config/ItemResponseExtractor.toml")
	r.printStatus()

	return &r

}

//
// implement the EventPipe interface, core work of the
// report engine.
//
func (r *ItemResponseExtractor) ProcessEventRecords(in chan *records.EventOrientedRecord) chan *records.EventOrientedRecord {

	out := make(chan *records.EventOrientedRecord)
	go func() {
		defer close(out)
		// does not open csv file
		var itemResponses [][]byte
		var eorItem *records.EventOrientedRecord
		for eor := range in {
			if !r.config.activated { // only process if activated
				out <- eor
				continue
			}

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
	}()
	return out
}

//
// add item response to the calc fields block
//
func addItemResponse(calculatedFields []byte, itemResponse []byte) []byte {

	var path string
	// preserve any existing calculated values
	json := calculatedFields
	gjson.GetBytes(itemResponse, "ir").
		ForEach(func(key, value gjson.Result) bool {
			path = fmt.Sprintf("CalculatedFields.%s", key)
			if key.String() == "ItemResponse" {
				json, _ = sjson.SetRawBytes(json, path, []byte(value.String()))
			} else {
				json, _ = sjson.SetBytes(json, path, value.String())
			}
			return true // keep iterating
		})

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

	// // need to add testlet ids as top-level elements...
	// // iterate with sjson block to add to responses

	responseId := eor.GetValueString("NAPStudentResponseSet.RefId")
	json := []byte{}
	gjson.GetBytes(eor.NAPStudentResponseSet, "NAPStudentResponseSet.TestletList.Testlet").
		ForEach(func(key, value gjson.Result) bool {
			testletId := value.Get("NAPTestletRefId").String()
			testletLocalId := value.Get("NAPTestletLocalId").String()
			value.Get("ItemResponseList.ItemResponse").
				ForEach(func(key, value gjson.Result) bool {
					json, _ = sjson.SetBytes(json, "ir.NAPStudentResponseSetRefId", responseId)
					json, _ = sjson.SetBytes(json, "ir.NAPTestletRefId", testletId)
					json, _ = sjson.SetBytes(json, "ir.NAPTestletLocalId", testletLocalId)
					json, _ = sjson.SetRawBytes(json, "ir.ItemResponse", []byte(value.String()))
					responses = append(responses, json)
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
func (r *ItemResponseExtractor) calculateFields(eor *records.EventOrientedRecord) []byte {

	return eor.CalculatedFields
}
