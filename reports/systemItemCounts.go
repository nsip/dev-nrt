package reports

import (
	"encoding/csv"
	"fmt"
	"strings"

	"github.com/nsip/dev-nrt/codeframe"
	"github.com/nsip/dev-nrt/records"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

type SystemItemCounts struct {
	baseReport // embed common setup capability
	cfh        codeframe.Helper
	usage      itemUsage // storage for data from stream
}

//
// usage totals need context as same item can be seen different
// amount of times in different tests
//
// Implements CalcFields lookup for reporting
//
type itemContext struct {
	testName   string
	testDomain string
	testLevel  string
	itemRefId  string
}

//
// for writing out we no longer have an event record
// this struct just a convenience placehloder for
// output
// Implements CalcFields lookup for reporting
// using the regular csv rendering approach
//
type itemSummary struct {
	CalculatedFields []byte
}

//
// pass a gjson path to retrieve the value at that location as a
// string, makes the summary conform to the standard csv rendering loop
//
func (i *itemSummary) GetValueString(queryPath string) string {

	//
	// get the root object
	//
	objName := strings.Split(queryPath, ".")[0]
	var data []byte
	switch objName {
	case "CalculatedFields":
		data = i.CalculatedFields
	default:
		data = []byte{}
	}

	return gjson.GetBytes(data, queryPath).String()
}

//
// core data structure, items and counts
//
type itemUsage = map[itemContext]int

//
// Number of times a test item was encountered by students accross all tests
//
func SystemItemCountsReport(cfh codeframe.Helper) *SystemItemCounts {

	r := SystemItemCounts{
		cfh:   cfh,
		usage: make(itemUsage, 0),
	}
	r.initialise("./config/SystemItemCounts.toml")
	r.printStatus()

	return &r

}

//
// implement the EventPipe interface, core work of the
// report engine.
//
func (r *SystemItemCounts) ProcessEventRecords(in chan *records.EventOrientedRecord) chan *records.EventOrientedRecord {

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
			// record the item details
			//
			r.countItem(eor)

			//
			// move on, this is a collector report so actual data
			// gets written out once the input stream has cloed, see below...
			//

			out <- eor
		}

		//
		// Once strem has clsed we've seen all event data...
		//
		//
		// iterate the collected item usage scores and create the ouptut report
		//
		for context, count := range r.usage {
			//
			// generate any calculated fields required
			//
			is := &itemSummary{CalculatedFields: []byte{}}
			is.CalculatedFields = r.calculateFields(context, count)

			//
			// now loop through the ouput definitions to create a
			// row of results
			//
			var result string
			var row []string = make([]string, 0, len(r.config.queries))
			for _, query := range r.config.queries {
				result = is.GetValueString(query)
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
// records the item info and usage count
//
func (r *SystemItemCounts) countItem(eor *records.EventOrientedRecord) {

	// populate the context object
	ic := itemContext{
		itemRefId:  eor.GetValueString("CalculatedFields.ItemResponse.NAPTestItemRefId"),
		testDomain: eor.GetValueString("NAPTest.TestContent.Domain"),
		testLevel:  eor.GetValueString("NAPTest.TestContent.TestLevel.Code"),
		testName:   eor.GetValueString("NAPTest.TestContent.TestName"),
	}

	// capture the score
	r.usage[ic]++

}

//
// generates a block of json that can be added to the
// record containing values that are not in the original data
//
//
func (r *SystemItemCounts) calculateFields(ic itemContext, usageCount int) []byte {

	json := []byte{}

	//
	// get any details required about this item
	//
	var itemLocalId, subLocalId string
	inCodeFrame, _ := r.cfh.GetItem(ic.itemRefId)
	if inCodeFrame {
		itemLocalId = r.cfh.GetCodeframeObjectValueString(ic.itemRefId, "NAPTestItem.TestItemContent.NAPTestItemLocalId")
	}
	//
	// check substitute status
	//
	subRefIds, isSubstitute := r.cfh.IsSubstituteItem(ic.itemRefId)
	for subRefId := range subRefIds {
		validSubItem, _ := r.cfh.GetItem(subRefId)
		if validSubItem {
			subLocalId = r.cfh.GetCodeframeObjectValueString(subRefId, "NAPTestItem.TestItemContent.NAPTestItemLocalId")
		}
	}

	//
	// set the report properties, insert this reportname to namespace these outputs
	//
	json, _ = sjson.SetBytes(json, "CalculatedFields.SystemItemCounts.NAPTest.TestContent.Domain", ic.testDomain)
	json, _ = sjson.SetBytes(json, "CalculatedFields.SystemItemCounts.TestContent.TestLevel.Code", ic.testLevel)
	json, _ = sjson.SetBytes(json, "CalculatedFields.SystemItemCounts.TestContent.TestName", ic.testName)

	json, _ = sjson.SetBytes(json, "CalculatedFields.SystemItemCounts.NAPTestItem.ItemInCodeFrame", inCodeFrame)
	json, _ = sjson.SetBytes(json, "CalculatedFields.SystemItemCounts.NAPTestItem.ItemIsSubstitute", isSubstitute)
	json, _ = sjson.SetBytes(json, "CalculatedFields.SystemItemCounts.NAPTestItem.UsageCount", usageCount)

	json, _ = sjson.SetBytes(json, "CalculatedFields.SystemItemCounts.NAPTestItem.TestItemContent.NAPTestItemLocalId", itemLocalId)
	json, _ = sjson.SetBytes(json, "CalculatedFields.SystemItemCounts.NAPTestItem.TestItemContent.SubstituteItem.SubstituteItemLocalId", subLocalId)

	return json
}
