package reports

import (
	"encoding/csv"
	"fmt"

	"github.com/nsip/dev-nrt/codeframe"
	"github.com/nsip/dev-nrt/records"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

type QldTestData struct {
	baseReport // embed common setup capability
	cfh        codeframe.Helper
}

//
// Simplified breakdown of items in test
//
func QldTestDataReport(cfh codeframe.Helper) *QldTestData {

	r := QldTestData{cfh: cfh}
	r.initialise("./config/QldTestData.toml")
	r.printStatus()

	return &r

}

//
// implement the EventPipe interface, core work of the
// report engine.
//
func (r *QldTestData) ProcessCodeframeRecords(in chan *records.CodeframeRecord) chan *records.CodeframeRecord {

	out := make(chan *records.CodeframeRecord)
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

			if cfr.RecordType != "NAPTestItem" { // only deal with test items
				out <- cfr
				continue
			}

			//
			// generate any calculated fields required
			//
			cfr.CalculatedFields = r.calculateFields(cfr)

			//
			// now loop through the ouput definitions to create a
			// row of results
			//
			var result string
			var row []string = make([]string, 0, len(r.config.queries))
			for _, query := range r.config.queries {
				result = cfr.GetValueString(query)
				row = append(row, result)
			}
			// write the row to the output file
			if err := w.Write(row); err != nil {
				fmt.Println("Warning: error writing record to csv:", r.config.name, err)
			}

			out <- cfr
		}
	}()
	return out
}

//
// generates a block of json that can be added to the
// record containing values that are not in the original data
//
//
func (r *QldTestData) calculateFields(cfr *records.CodeframeRecord) []byte {

	cf := cfr.CalculatedFields
	// get the refids of the containers of this item
	testRefId := gjson.GetBytes(cf, "CalculatedFields.NAPTestRefId").String()

	// get the test properties required
	testyear := r.cfh.GetCodeframeObjectValueString(testRefId, "NAPTest.TestContent.TestYear")
	testlevel := r.cfh.GetCodeframeObjectValueString(testRefId, "NAPTest.TestContent.TestLevel.Code")
	testdomain := r.cfh.GetCodeframeObjectValueString(testRefId, "NAPTest.TestContent.Domain")

	// add the data to calculated fields
	cf, _ = sjson.SetBytes(cf, "CalculatedFields.NAPTest.TestContent.TestYear", testyear)
	cf, _ = sjson.SetBytes(cf, "CalculatedFields.NAPTest.TestContent.TestLevel.Code", testlevel)
	cf, _ = sjson.SetBytes(cf, "CalculatedFields.NAPTest.TestContent.Domain", testdomain)

	return cf

}
