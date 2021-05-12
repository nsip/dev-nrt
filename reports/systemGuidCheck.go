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
// detected GUID issue
//
type guidIssue struct {
	or            *records.ObjectRecord
	objectname    string
	objecttype    string
	shouldpointto string
	pointsto      string
	guid          string
}

type QaGuidCheck struct {
	baseReport // embed common setup capability
	cfh        helper.ObjectHelper
	issues     []*guidIssue
}

//
// Checks that all guids point somewhere valid
//
// NOTE: This is a Collector report, it harvests data from the stream but
// only writes it out once all data has passed through.
//
//
func QaGuidCheckReport(cfh helper.ObjectHelper) *QaGuidCheck {

	r := QaGuidCheck{cfh: cfh}
	r.initialise("./config/SystemGuidCheck.toml")
	r.printStatus()

	return &r

}

//
// implement the EventPipe interface, core work of the
// report engine.
//
func (r *QaGuidCheck) ProcessObjectRecords(in chan *records.ObjectRecord) chan *records.ObjectRecord {

	out := make(chan *records.ObjectRecord)
	go func() {
		defer close(out)
		// open the csv file writer, and set the header
		w := csv.NewWriter(r.outF)
		defer r.outF.Close()
		w.Write(r.config.header)
		defer w.Flush()

		for or := range in {
			if !r.config.activated { // only process if activated
				out <- or
				continue
			}

			//
			// check for GUID errors
			//
			r.validate(or)

			out <- or
		}

		//
		// Write out issues collected
		//
		for _, issue := range r.issues {
			//
			// generate any calculated fields required
			//
			issue.or.CalculatedFields = r.calculateFields(issue)

			//
			// now loop through the ouput definitions to create a
			// row of results
			//
			var result string
			var row []string = make([]string, 0, len(r.config.queries))
			for _, query := range r.config.queries {
				result = issue.or.GetValueString(query)
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
func (r *QaGuidCheck) calculateFields(issue *guidIssue) []byte {

	json := issue.or.CalculatedFields

	// add details of any issues found
	json, _ = sjson.SetBytes(json, "CalculatedFields.ObjectName", issue.objectname)
	json, _ = sjson.SetBytes(json, "CalculatedFields.ObjectType", issue.objecttype)
	json, _ = sjson.SetBytes(json, "CalculatedFields.ShouldPointTo", issue.shouldpointto)
	json, _ = sjson.SetBytes(json, "CalculatedFields.PointsTo", issue.pointsto)
	json, _ = sjson.SetBytes(json, "CalculatedFields.GUID", issue.guid)

	return json
}

func guidcheck(issues []*guidIssue, or *records.ObjectRecord, objecttype string, guid string, expected_type string, pointsto string) []*guidIssue {
	if guid == "" {
		return issues
	}
	if pointsto == "" {
		pointsto = "nil"
	}
	if pointsto != expected_type {
		issues = append(issues, &guidIssue{
			or:            or,
			objectname:    or.RefId(),
			objecttype:    objecttype,
			shouldpointto: expected_type,
			pointsto:      pointsto,
			guid:          guid,
		})
	}
	return issues
}

//
// checks integrity of assets in the response member of the record
// iterates all linkages and tests against the data in the
// object helper.
// Any discrepancies in the members or the linkage will be reported.
// Mutiple issues may be detected in the same response.
//
//
func (r *QaGuidCheck) validate(or *records.ObjectRecord) {

	switch or.RecordType {
	case "NAPTestlet":
		test := or.GetValueString("NAPTestlet.NAPTestRefId")
		r.issues = guidcheck(r.issues, or, "NAPTestlet", test, "NAPTest", r.cfh.GetTypeFromGuid(test))
		gjson.GetBytes(or.Json, "NAPTestlet.TestItemList.TestItem").
			ForEach(func(key, value gjson.Result) bool {
				item := value.Get("TestItemRefId").String()
				r.issues = guidcheck(r.issues, or, "NAPTestlet", item, "NAPTestItem", r.cfh.GetTypeFromGuid(item))
				return true
			})
	case "NAPTestItem":
		gjson.GetBytes(or.Json, "NAPTestItem.TestItemContent.ItemSubstitutedForList.SubstituteItem").
			ForEach(func(key, value gjson.Result) bool {
				item := value.Get("SubstituteItemRefId").String()
				r.issues = guidcheck(r.issues, or, "NAPTestItem", item, "NAPTestItem", r.cfh.GetTypeFromGuid(item))
				return true
			})
	case "NAPTestScoreSummary":
		school := or.GetValueString("NAPTestScoreSummary.SchoolInfoRefId")
		test := or.GetValueString("NAPTestScoreSummary.NAPTestRefId")
		r.issues = guidcheck(r.issues, or, "NAPTestScoreSummary", school, "SchoolInfo", r.cfh.GetTypeFromGuid(school))
		r.issues = guidcheck(r.issues, or, "NAPTestScoreSummary", test, "NAPTest", r.cfh.GetTypeFromGuid(test))
	case "NAPEventStudentLink":
		school := or.GetValueString("NAPEventStudentLink.SchoolInfoRefId")
		student := or.GetValueString("NAPEventStudentLink.StudentPersonalRefId")
		test := or.GetValueString("NAPEventStudentLink.NAPTestRefId")
		r.issues = guidcheck(r.issues, or, "NAPEventStudentLink", school, "SchoolInfo", r.cfh.GetTypeFromGuid(school))
		r.issues = guidcheck(r.issues, or, "NAPEventStudentLink", student, "StudentPersonal", r.cfh.GetTypeFromGuid(student))
		r.issues = guidcheck(r.issues, or, "NAPEventStudentLink", test, "NAPTest", r.cfh.GetTypeFromGuid(test))
	case "NAPStudentResponseSet":
		student := or.GetValueString("NAPStudentResponseSet.StudentPersonalRefId")
		test := or.GetValueString("NAPStudentResponseSet.NAPTestRefId")
		r.issues = guidcheck(r.issues, or, "NAPStudentResponseSet", student, "StudentPersonal", r.cfh.GetTypeFromGuid(student))
		r.issues = guidcheck(r.issues, or, "NAPStudentResponseSet", test, "NAPTest", r.cfh.GetTypeFromGuid(test))
		gjson.GetBytes(or.Json, "NAPStudentResponseSet.TestletList.Testlet").
			ForEach(func(key, value gjson.Result) bool {
				testlet := value.Get("NAPTestletRefId").String()
				r.issues = guidcheck(r.issues, or, "NAPStudentResponseSet", testlet, "NAPTestlet", r.cfh.GetTypeFromGuid(testlet))
				gjson.GetBytes([]byte(value.Raw), "ItemResponseList.ItemResponse").
					ForEach(func(key, value gjson.Result) bool {
						item := value.Get("NAPTestItemRefId").String()
						r.issues = guidcheck(r.issues, or, "NAPStudentResponseSet", item, "NAPTestItem", r.cfh.GetTypeFromGuid(item))
						return true
					})
				return true
			})
	case "NAPCodeFrame":
		test := or.GetValueString("NAPCodeFrame.NAPTestRefId")
		r.issues = guidcheck(r.issues, or, "NAPCodeFrame", test, "NAPTest", r.cfh.GetTypeFromGuid(test))
		gjson.GetBytes(or.Json, "NAPCodeFrame.TestletList.Testlet").
			ForEach(func(key, value gjson.Result) bool {
				testlet := value.Get("NAPTestletRefId").String()
				r.issues = guidcheck(r.issues, or, "NAPCodeFrame", testlet, "NAPTestlet", r.cfh.GetTypeFromGuid(testlet))
				gjson.GetBytes([]byte(value.Raw), "TestItemList.TestItem").
					ForEach(func(key, value gjson.Result) bool {
						item := value.Get("NAPTestItemRefId").String()
						r.issues = guidcheck(r.issues, or, "NAPCodeFrame", item, "NAPTestItem", r.cfh.GetTypeFromGuid(item))
						return true
					})
				return true
			})
	}
}
