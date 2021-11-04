package reports

import (
	"encoding/csv"
	"fmt"

	"github.com/nsip/dev-nrt/helper"
	"github.com/nsip/dev-nrt/records"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

type ItemExpectedResponses struct {
	baseReport // embed common setup capability
	cfh        helper.CodeframeHelper
}

//
// Highlights any differences between the items expected to be presented and those seen by the student
//
func ItemExpectedResponsesReport(cfh helper.CodeframeHelper) *ItemExpectedResponses {

	r := ItemExpectedResponses{cfh: cfh}
	r.initialise("./config/ItemExpectedResponses.toml")
	r.printStatus()

	return &r

}

//
// implement the EventPipe interface, core work of the
// report engine.
//
func (r *ItemExpectedResponses) ProcessEventRecords(in chan *records.EventOrientedRecord) chan *records.EventOrientedRecord {

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

			if !eor.HasNAPStudentResponseSet { // don't process if no response
				out <- eor
				continue
			}

			responseTestlets := gjson.GetBytes(eor.NAPStudentResponseSet, "NAPStudentResponseSet.TestletList.Testlet").Array()
			if len(responseTestlets) == 0 { // response with no content so don't process
				out <- eor
				continue
			}

			//
			// generate any calculated fields required
			//
			eor.CalculatedFields = r.calculateFields(eor)

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
func (r *ItemExpectedResponses) calculateFields(eor *records.EventOrientedRecord) []byte {

	json := eor.CalculatedFields // preserve any existing cal fields

	testRefId := eor.GetValueString("NAPTest.RefId") // get test so we can interrogate codeframe

	// iterate the list of testlets in the response
	var testletRefId, itemRefId, itemCorrectness string
	var testletName, path string
	var testletCount int // testlet index for output in report eg. Testlet.1.xxx, Testlet.2.xxx
	gjson.GetBytes(eor.NAPStudentResponseSet, "NAPStudentResponseSet.TestletList.Testlet").
		ForEach(func(key, value gjson.Result) bool { // handle testlets

			seenItems := make(map[string]struct{}, 0) // set of seen items
			expectedNotSeen := make([]string, 0)      // expected but not seen items
			seenNotExpected := make([]string, 0)      // seen but not expected items
			correctnessCounter := make(map[string]int, 0)
			testletCount++
			testletRefId = value.Get("NAPTestletRefId").String() // get the testlet refid

			value.Get("ItemResponseList.ItemResponse").
				ForEach(func(key, value gjson.Result) bool { // handle items
					itemCorrectness = value.Get("ResponseCorrectness").String()
					// update the status counter
					correctnessCounter[itemCorrectness]++
					// add to list of seen items
					itemRefId = value.Get("NAPTestItemRefId").String() // get the item refid
					seenItems[itemRefId] = struct{}{}
					//
					return true // keep iterating
				})
			//
			// ========================================================
			//
			// get the expected items for this testlet
			expectedItems := r.cfh.GetExpectedTestletItems(testRefId, testletRefId)
			//
			// check seen not expected
			//
			for item := range seenItems {

				// check in case this item has a substitute
				if subItems, validSub := r.cfh.IsSubstituteItem(item); validSub {
					// fmt.Println(validSub, item, subItems)
					var ok bool
					for subItem := range subItems {
						if _, ok = expectedItems[subItem]; ok {
							//	break
							expectedItems[item] = struct{}{}
						}
					}
				}
			}
			for item := range expectedItems {
				// check in case this item has a reverse substitute - shouldn't be the case but in practice has happened
				if subItems, validSub := r.cfh.IsSubstituteItem(item); validSub {
					var ok bool
					for subItem := range subItems {
						if _, ok = seenItems[subItem]; ok {
							//      break
							seenItems[item] = struct{}{}
						}
					}
				}
			}
			for item := range seenItems {
				// check if expected
				if _, ok := expectedItems[item]; !ok {
					localId := r.cfh.GetCodeframeObjectValueString(item, "NAPTestItem.TestItemContent.NAPTestItemLocalId")
					seenNotExpected = append(seenNotExpected, localId)
				}

			}

			//
			// check for expected not seen
			//
			for item := range expectedItems {
				// check if seen
				if _, ok := seenItems[item]; !ok {
					localId := r.cfh.GetCodeframeObjectValueString(item, "NAPTestItem.TestItemContent.NAPTestItemLocalId")
					expectedNotSeen = append(expectedNotSeen, localId)
				}
			}

			//
			// now write out the report for each testlet
			//
			//
			// testlet name
			//
			testletName = r.cfh.GetCodeframeObjectValueString(testletRefId, "NAPTestlet.TestletContent.TestletName")
			path = fmt.Sprintf("CalculatedFields.ItemExpectedResponses.Testlet.%d.TestletName", testletCount)
			json, _ = sjson.SetBytes(json, path, testletName)
			//
			// expected items count
			//
			path = fmt.Sprintf("CalculatedFields.ItemExpectedResponses.Testlet.%d.ExpectedItemsCount", testletCount)
			json, _ = sjson.SetBytes(json, path, len(expectedItems))
			//
			// item count by correctness status
			//
			for tc, count := range correctnessCounter {
				path = fmt.Sprintf("CalculatedFields.ItemExpectedResponses.Testlet.%d.FoundItems.%s", testletCount, tc)
				json, _ = sjson.SetBytes(json, path, count)
			}
			//
			// expected not seen
			//
			path = fmt.Sprintf("CalculatedFields.ItemExpectedResponses.Testlet.%d.ExpectedItemsNotFound", testletCount)
			if len(expectedNotSeen) > 0 {
				json, _ = sjson.SetBytes(json, path, expectedNotSeen)
			}
			// seen not expected
			path = fmt.Sprintf("CalculatedFields.ItemExpectedResponses.Testlet.%d.FoundItemsNotExpected", testletCount)
			if len(seenNotExpected) > 0 {
				json, _ = sjson.SetBytes(json, path, seenNotExpected)
			}
			//
			// ========================================================

			return true // keep iterating testlets
		})

	return json
}
