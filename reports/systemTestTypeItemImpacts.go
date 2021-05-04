package reports

import (
	"encoding/csv"
	"fmt"

	"github.com/nsip/dev-nrt/helper"
	"github.com/nsip/dev-nrt/records"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

type SystemTestTypeItemImpacts struct {
	baseReport // embed common setup capability
	cfh        helper.CodeframeHelper
}

//
// Reports items for which the item responses are unexpected based on the test domain
//
func SystemTestTypeItemImpactsReport(cfh helper.CodeframeHelper) *SystemTestTypeItemImpacts {

	r := SystemTestTypeItemImpacts{cfh: cfh}
	r.initialise("./config/SystemTestTypeItemImpacts.toml")
	r.printStatus()

	return &r

}

//
// implement the EventPipe interface, core work of the
// report engine.
//
func (r *SystemTestTypeItemImpacts) ProcessEventRecords(in chan *records.EventOrientedRecord) chan *records.EventOrientedRecord {

	out := make(chan *records.EventOrientedRecord)
	go func() {
		defer close(out)
		// open the csv file writer, and set the header
		w := csv.NewWriter(r.outF)
		defer r.outF.Close()
		w.Write(r.config.header)
		defer w.Flush()

		for eor := range in {
			if !r.config.activated { // only process if activated
				out <- eor
				continue
			}

			//
			// single event record can produce multiple item errors
			//
			for _, itemError := range testItemImpact(eor) {

				//
				// generate any calculated fields required
				//
				eor.CalculatedFields = r.calculateFields(eor, itemError)

				//
				// now loop through the ouput definitions to create a
				// row of results
				//
				var result string
				var row []string = make([]string, 0, len(r.config.queries))
				for _, query := range r.config.queries {
					result = eor.GetValueString(query)
					row = append(row, result)
				}
				// write the row to the output file
				if err := w.Write(row); err != nil {
					fmt.Println("Warning: error writing record to csv:", r.config.name, err)
				}
			}

			out <- eor
		}
	}()
	return out
}

//
// check each items against the qa criteria, return a list of items that have errors
//
func testItemImpact(eor *records.EventOrientedRecord) []*itemError {

	//
	// iterate testlets -> items in response
	//
	itemErrors := make([]*itemError, 0)
	gjson.GetBytes(eor.NAPStudentResponseSet, "NAPStudentResponseSet.TestletList.Testlet").
		ForEach(func(key, value gjson.Result) bool {
			ierr := &itemError{}
			//
			// now iterate testlet item responses
			//
			value.Get("ItemResponseList.ItemResponse").
				ForEach(func(key, value gjson.Result) bool {
					// capture the item refid
					ierr.itemRefId = value.Get("NAPTestItemRefId").String()
					// and the json of the item response
					ierr.itemResponseJson = value.Raw
					// properties used for logic tests
					itemScore := value.Get("Score").String()
					itemSubScores := value.Get("SubscoreList.Subscore").Array()

					//
					// validation logic copied from n2 implementation
					//
					//
					if eor.IsWritingResponse() {
						if nonzero(itemScore) && len(itemSubScores) == 0 { // score assigned but no subscores to justify it
							ierr.err = errNoWritingSubscores
							itemErrors = append(itemErrors, ierr)
						}
					}
					//
					if !eor.IsWritingResponse() {
						if len(itemSubScores) > 0 { // not a writing item, but has subscores
							ierr.err = errUnexpectedItemSubscores
							itemErrors = append(itemErrors, ierr)
						}
					}
					//
					return true // keep iterating, move on to next item response
				})
			return true // keep iterating
		})

	return itemErrors

}

//
// generates a block of json that can be added to the
// record containing values that are not in the original data
//
//
func (r *SystemTestTypeItemImpacts) calculateFields(eor *records.EventOrientedRecord, ierr *itemError) []byte {

	json := eor.CalculatedFields // maintain existing calculated fields

	itemLocalId := r.cfh.GetCodeframeObjectValueString(ierr.itemRefId, "NAPTestItem.TestItemContent.NAPTestItemLocalId")
	itemName := r.cfh.GetCodeframeObjectValueString(ierr.itemRefId, "NAPTestItem.TestItemContent.ItemName")
	itemSubDomain := r.cfh.GetCodeframeObjectValueString(ierr.itemRefId, "NAPTestItem.TestItemContent.Subdomain")
	itemGenre := r.cfh.GetCodeframeObjectValueString(ierr.itemRefId, "NAPTestItem.TestItemContent.WritingGenre")
	itemScore := gjson.Get(ierr.itemResponseJson, "Score").String()
	itemSubScores := gjson.Get(ierr.itemResponseJson, "SubscoreList.Subscore").String()

	json, _ = sjson.SetBytes(json, "CalculatedFields.TestItem.TestItemContent.NAPTestItemLocalId", itemLocalId)
	json, _ = sjson.SetBytes(json, "CalculatedFields.TestItem.TestItemContent.ItemName", itemName)
	json, _ = sjson.SetBytes(json, "CalculatedFields.TestItem.TestItemContent.Subdomain", itemSubDomain)
	json, _ = sjson.SetBytes(json, "CalculatedFields.TestItem.TestItemContent.WritingGenre", itemGenre)
	json, _ = sjson.SetBytes(json, "CalculatedFields.ItemResponse.Score", itemScore)
	json, _ = sjson.SetBytes(json, "CalculatedFields.ItemResponse.Subscores", itemSubScores)

	json, _ = sjson.SetBytes(json, "CalculatedFields.TestTypeItemImpactError", ierr.err.Error())

	return json

}
