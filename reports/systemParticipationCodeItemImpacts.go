package reports

import (
	"encoding/csv"
	"errors"
	"fmt"

	"github.com/nsip/dev-nrt/codeframe"
	"github.com/nsip/dev-nrt/records"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

type SystemParticipationCodeItemImpacts struct {
	baseReport // embed common setup capability
	cfh        codeframe.Helper
}

var (
	errUnexpectedItemResponse  = errors.New("Item response captured without student completing test")
	errUnexpectedItemScore     = errors.New("Scored test item with p-code other than P or R")
	errNonZeroItemScore        = errors.New("Non-zero scored test item with p-code of R")
	errMissingItemScore        = errors.New("Unscored test with p-code of P or R")
	errMissingItemWritingScore = errors.New("Unscored writing item with p-code of P")
)

type itemError struct {
	err              error
	itemRefId        string
	itemResponseJson string
}

//
// Reports errors when response items contain unexpected information based on the
// participation code
//
func SystemParticipationCodeItemImpactsReport(cfh codeframe.Helper) *SystemParticipationCodeItemImpacts {

	r := SystemParticipationCodeItemImpacts{cfh: cfh}
	r.initialise("./config/SystemParticipationCodeItemImpacts.toml")
	r.printStatus()

	return &r

}

//
// implement the EventPipe interface, core work of the
// report engine.
//
func (r *SystemParticipationCodeItemImpacts) ProcessEventRecords(in chan *records.EventOrientedRecord) chan *records.EventOrientedRecord {

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
			for _, itemError := range participationItemImpact(eor) {
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
// checks for anomalies in the record.
//
// same checks as n2.
//
func participationItemImpact(eor *records.EventOrientedRecord) []itemError {

	//
	// fetch properties from record
	//
	participationCode := eor.GetValueString("NAPEventStudentLink.ParticipationCode")
	testDomain := eor.GetValueString("NAPTest.TestContent.Domain")

	//
	// now iterate testlets -> items in response
	//
	itemErrors := make([]itemError, 0)
	gjson.GetBytes(eor.NAPStudentResponseSet, "NAPStudentResponseSet.TestletList.Testlet").
		ForEach(func(key, value gjson.Result) bool {
			ierr := itemError{}
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
					itemLapsedTime := value.Get("LapsedTimeItem").String()
					itemScore := value.Get("Score").String()
					itemResponse := value.Get("Response").String()
					itemSubScores := value.Get("SubscoreList.Subscore").Array()

					//
					// validation logic copied from n2 implementation
					//
					// note: logic using testletscore has not been ported
					// as this score n longer exists in dataset
					//
					if itemLapsedTime == "" || itemResponse == "" {
						if participationCode != "P" && participationCode != "S" {
							ierr.err = errUnexpectedItemResponse
							itemErrors = append(itemErrors, ierr)
						}
					}
					//
					if itemScore == "" {
						if participationCode == "R" || participationCode == "P" {
							ierr.err = errMissingItemScore
							itemErrors = append(itemErrors, ierr)
						}
					}
					//
					if testDomain == "Writing" && participationCode == "P" {
						if len(itemSubScores) == 0 {
							ierr.err = errMissingItemWritingScore
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
func (r *SystemParticipationCodeItemImpacts) calculateFields(eor *records.EventOrientedRecord, ierr itemError) []byte {

	json := eor.CalculatedFields // maintain existing calculated fields

	itemLocalId := r.cfh.GetCodeframeObjectValueString(ierr.itemRefId, "NAPTestItem.TestItemContent.NAPTestItemLocalId")
	itemName := r.cfh.GetCodeframeObjectValueString(ierr.itemRefId, "NAPTestItem.TestItemContent.ItemName")
	itemScore := gjson.Get(ierr.itemResponseJson, "Score").String()
	itemSubScores := gjson.Get(ierr.itemResponseJson, "SubscoreList.Subscore").String()

	json, _ = sjson.SetBytes(json, "CalculatedFields.TestItem.TestItemContent.NAPTestItemLocalId", itemLocalId)
	json, _ = sjson.SetBytes(json, "CalculatedFields.TestItem.TestItemContent.ItemName", itemName)
	json, _ = sjson.SetBytes(json, "CalculatedFields.ItemResponse.Score", itemScore)
	json, _ = sjson.SetBytes(json, "CalculatedFields.ItemResponse.Subscores", itemSubScores)

	json, _ = sjson.SetBytes(json, "CalculatedFields.ParticipationCodeItemImpactError", ierr.err.Error())

	return json
}
