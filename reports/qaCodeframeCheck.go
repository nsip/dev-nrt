package reports

import (
	"encoding/csv"
	"fmt"

	"github.com/nsip/dev-nrt/helper"
	"github.com/nsip/dev-nrt/records"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

//
// capture the connected graph of any
// detected codeframe issue
//
type codeframeIssue struct {
	cfr           *records.ObjectRecord
	id            string
	localid       string
	objtype       string
	itemSubRefIds []string
}

type QaCodeframeCheck struct {
	baseReport // embed common setup capability
	cfh        helper.ObjectHelper
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
func QaCodeframeCheckReport(cfh helper.ObjectHelper) *QaCodeframeCheck {

	r := QaCodeframeCheck{cfh: cfh}
	r.initialise("./config/QaCodeframeCheck.toml")
	r.printStatus()

	return &r

}

func (r *QaCodeframeCheck) ProcessObjectRecords(in chan *records.ObjectRecord) chan *records.ObjectRecord {

	out := make(chan *records.ObjectRecord)
	go func() {
		defer close(out)
		// open the csv file writer, and set the header
		w := csv.NewWriter(r.outF)
		defer r.outF.Close()
		w.Write(r.config.header)
		defer w.Flush()

		for cfr := range in {
			if !r.config.activated { // only process if activated
				out <- cfr
				continue
			}

			if cfr.RecordType != "NAPCodeFrame" && cfr.RecordType != "NAPTest" && cfr.RecordType != "NAPTestlet" && cfr.RecordType != "NAPTestItem" {
				out <- cfr
				continue
			}

			//
			// check for codeframe errors
			//
			r.validate(cfr)
			out <- cfr
		}
		//
		// Write out issues collected
		//
		for _, issue := range r.issues {

			//
			// generate any calculated fields required
			//
			issue.cfr.CalculatedFields = r.calculateFields(issue)

			//
			// now loop through the ouput definitions to create a
			// row of results
			//
			var result string
			var row []string = make([]string, 0, len(r.config.queries))
			for _, query := range r.config.queries {
				result = issue.cfr.GetValueString(query)
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

	json := issue.cfr.CalculatedFields

	// add details of any issues found
	json, _ = sjson.SetBytes(json, "CalculatedFields.ObjectID", issue.id)
	json, _ = sjson.SetBytes(json, "CalculatedFields.LocalID", issue.localid)
	json, _ = sjson.SetBytes(json, "CalculatedFields.ObjectType", issue.objtype)

	if issue.itemSubRefIds != nil {
		json, _ = sjson.SetBytes(json, "CalculatedFields.SubstituteItemRefIds", issue.itemSubRefIds)
	}

	return json
}

func gjsonArray2StringSlice(res []gjson.Result) []string {
	ret := make([]string, 0)
	for _, name := range res {
		ret = append(ret, name.String())
	}
	return ret
}

//
// checks integrity of codeframe assets in the response member of the record
// iterates each item-testlet-test triplet and tests against the data in the
// codeframe helper.
// Any discrepancies in the members or the linkage will be reported.
// Mutiple issues may be detected in the same response.
//
//
func (r *QaCodeframeCheck) validate(cfr *records.ObjectRecord) {

	switch cfr.RecordType {
	case "NAPCodeFrame":
		testid := cfr.GetValueString("NAPCodeFrame.NAPTestRefId")
		testlocalid := cfr.GetValueString("NAPCodeFrame.TestContent.NAPTestLocalId")
		objtype := r.cfh.GetTypeFromGuid(testid)
		if objtype != "NAPTest" {
			r.issues = append(r.issues, &codeframeIssue{id: testid, localid: testlocalid, objtype: "test", cfr: cfr})
		}
		gjson.GetBytes(cfr.Json, "NAPCodeFrame.TestletList.Testlet").
			ForEach(func(key, value gjson.Result) bool {
				testletRefId := value.Get("NAPTestletRefId").String()
				testletlocalid := value.Get("TestletContent.NAPTestletLocalId").String()
				objtype := r.cfh.GetTypeFromGuid(testletRefId)
				if objtype != "NAPTestlet" {
					r.issues = append(r.issues, &codeframeIssue{id: testletRefId, localid: testletlocalid, objtype: "testlet", cfr: cfr})
				}
				value.Get("TestItemList.TestItem").
					ForEach(func(key, value gjson.Result) bool {
						//
						// get item identifiers
						//
						itemRefId := value.Get("TestItemRefId").String()
						itemlocalid := value.Get("TestItemContent.NAPTestItemLocalId").String()
						objtype := r.cfh.GetTypeFromGuid(testletRefId)
						if objtype != "NAPTestlet" {
							subs := value.Get("TestItemContent.ItemSubstitutedForList.SubstituteItem.#.SubstituteItemRefId")

							r.issues = append(r.issues, &codeframeIssue{id: itemRefId, localid: itemlocalid, objtype: "testitem", itemSubRefIds: gjsonArray2StringSlice(subs.Array()), cfr: cfr})

						}
						return true // keep iterating, move on to next item response
					})
				return true // keep iterating
			})
	case "NAPTest":
		testid := cfr.GetValueString("NAPTest.RefId")
		if !r.cfh.InCodeFrame(testid) {
			r.issues = append(r.issues, &codeframeIssue{id: testid, localid: "nil", objtype: "test", cfr: cfr})
		}
	case "NAPTestlet":
		testid := cfr.GetValueString("NAPTestlet.RefId")
		if !r.cfh.InCodeFrame(testid) {
			r.issues = append(r.issues, &codeframeIssue{id: testid, localid: "nil", objtype: "testlet", cfr: cfr})
		}
	case "NAPTestItem":
		testid := cfr.GetValueString("NAPTestItem.RefId")
		if !r.cfh.InCodeFrame(testid) {
			subs := gjson.GetBytes(cfr.Json, "NAPTestItem.TestItemContent.ItemSubstitutedForList.SubstituteItem.#.SubstituteItemRefId")
			r.issues = append(r.issues, &codeframeIssue{id: testid, localid: "nil", objtype: "testitem", itemSubRefIds: gjsonArray2StringSlice(subs.Array()), cfr: cfr})
		}

	}

}
