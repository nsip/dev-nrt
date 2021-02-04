package reports

import (
	"encoding/csv"
	"fmt"

	"github.com/nsip/dev-nrt/codeframe"
	"github.com/nsip/dev-nrt/records"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

//
// capture the connected graph of any
// detected codeframe issue
//
type codeframeIssue struct {
	eor            *records.EventOrientedRecord
	testRefId      string
	testLocalId    string
	testletRefId   string
	testletLocalId string
	itemRefId      string
	itemLocalId    string
	itemSubRefIds  map[string]struct{}
	itemSubLocalId string
	substitue      bool
}

type QaCodeframeCheck struct {
	baseReport // embed common setup capability
	cfh        codeframe.Helper
	issues     []*codeframeIssue
}

//
// Checks that items-testlets-tests seen in responses
// are all in the codeframe
//
// NOTE: This is a Collector report, it harvests data from the stream but
// only writes it out once all data has passed through.
//
//
func QaCodeframeCheckReport(cfh codeframe.Helper) *QaCodeframeCheck {

	r := QaCodeframeCheck{cfh: cfh}
	r.initialise("./config/QaCodeframeCheck.toml")
	r.printStatus()

	return &r

}

//
// implement the EventPipe interface, core work of the
// report engine.
//
func (r *QaCodeframeCheck) ProcessEventRecords(in chan *records.EventOrientedRecord) chan *records.EventOrientedRecord {

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
			// check for codeframe errors
			//
			r.validate(eor)

			out <- eor
		}

		//
		// Write out issues collected
		//
		for _, issue := range r.issues {
			//
			// generate any calculated fields required
			//
			issue.eor.CalculatedFields = r.calculateFields(issue)

			//
			// now loop through the ouput definitions to create a
			// row of results
			//
			var result string
			var row []string = make([]string, 0, len(r.config.queries))
			for _, query := range r.config.queries {
				result = issue.eor.GetValueString(query)
				row = append(row, result)
			}
			// write the row to the output file
			if err := w.Write(row); err != nil {
				fmt.Println("Warning: error writing record to csv:", r.config.name, err)
			}
		}

	}()
	return out
}

//
// generates a block of json that can be added to the
// record containing values that are not in the original data
//
//
func (r *QaCodeframeCheck) calculateFields(issue *codeframeIssue) []byte {

	json := issue.eor.CalculatedFields

	// add details of any issues found
	json, _ = sjson.SetBytes(json, "CalculatedFields.NAPTestlet.RefId", issue.testletRefId)
	json, _ = sjson.SetBytes(json, "CalculatedFields.NAPTestlet.TestletContent.NAPTestletLocalId", issue.testletLocalId)
	json, _ = sjson.SetBytes(json, "CalculatedFields.NAPTestItem.RefId", issue.itemRefId)
	json, _ = sjson.SetBytes(json, "CalculatedFields.NAPTestItem.TestItemContent.NAPTestItemLocalId", issue.itemLocalId)
	json, _ = sjson.SetBytes(json, "CalculatedFields.ErrorType", "item->testlet->test not found in codeframe")

	//
	// this will be true if a substitute item has been used as the primary reference in the
	// codeframe - in effect the item/substitute refernces have been created the wrong way round
	//
	if issue.substitue {
		json, _ = sjson.SetBytes(json, "CalculatedFields.SubstituteItemRefId", issue.itemSubRefIds)
		json, _ = sjson.SetBytes(json, "CalculatedFields.SubstituteItemLocalId", issue.itemSubLocalId)
		json, _ = sjson.SetBytes(json, "CalculatedFields.ErrorType", "codeframe acyclical use of substitute item")
	}

	return json
}

//
// checks integrity of codeframe assets in the response member of the record
// iterates each item-testlet-test triplet and tests against the data in the
// codeframe helper.
// Any discrepancies in the members or the linkage will be reported.
// Mutiple issues may be detected in the same response.
//
//
func (r *QaCodeframeCheck) validate(eor *records.EventOrientedRecord) {

	//
	// iterate response testlets
	//
	gjson.GetBytes(eor.NAPStudentResponseSet, "NAPStudentResponseSet.TestletList.Testlet").
		ForEach(func(key, value gjson.Result) bool {
			cfi := &codeframeIssue{}
			containers := make(map[string][]string, 0)
			//
			// get testlet identifiers
			//
			cfi.testletRefId = value.Get("NAPTestletRefId").String()
			cfi.testletLocalId = value.Get("NAPTestletLocalId").String()
			//
			// now iterate testlet item responses
			//
			value.Get("ItemResponseList.ItemResponse").
				ForEach(func(key, value gjson.Result) bool {
					//
					// get item identifiers
					//
					cfi.itemRefId = value.Get("NAPTestItemRefId").String()
					cfi.itemLocalId = value.Get("NAPTestItemLocalId").String()
					//
					// first we need ot see if this is a substitute item
					//
					if cfi.itemSubRefIds, cfi.substitue = r.cfh.IsSubstituteItem(cfi.itemRefId); cfi.substitue {
						//
						// if it is then use the substitute to look up the
						// codeframe validity, codeframe itself doesn;t know about
						// substitute items
						//
						for subRefId := range cfi.itemSubRefIds {
							for k, v := range r.cfh.GetContainersForItem(subRefId) {
								containers[k] = v // add testlet-test reference to lookup
							}
						}
						cfi.itemSubLocalId = r.cfh.GetCodeframeObjectValueString(cfi.itemRefId, "NAPTestItem.TestItemContent.NAPTestItemLocalId")
					} else {
						//
						// if not we check the validity of the item as given
						//
						containers = r.cfh.GetContainersForItem(cfi.itemRefId)
					}
					// get test identifiers
					//
					cfi.testRefId = eor.GetValueString("NAPTest.RefId")
					cfi.testLocalId = eor.GetValueString("NAPTest.TestContent.NAPTestLocalId")
					//
					// now we check that the available combinations of test and testlet
					// in the codeframe for this item have at least one match with the
					// combination captured in this response
					//
					for testlet, tests := range containers {
						for _, test := range tests {
							if testlet == cfi.testletRefId && test == cfi.testRefId {
								return true // keep iterating - gjson equiv. of continue - to next item response
							}
						}
					}
					//
					// if we got here then our item has an issue
					// add to the issues list
					//
					cfi.eor = eor
					r.issues = append(r.issues, cfi)
					//
					return true // keep iterating, move on to next item response
				})
			return true // keep iterating
		})

}
