package reports

import (
	"encoding/csv"
	"fmt"
	"html"
	"strconv"
	"strings"

	"github.com/nsip/dev-nrt/records"
	"github.com/tidwall/sjson"
)

type CompareItemWriting struct {
	baseReport // embed common setup capability
}

//
// Simplified version of writing extract, shows writing item and response
// with some student info.
//
func CompareItemWritingReport() *CompareItemWriting {

	r := CompareItemWriting{}
	r.initialise("./config/CompareItemWriting.toml")
	r.printStatus()

	return &r

}

//
// implement the EventPipe interface, core work of the
// report engine.
//
func (r *CompareItemWriting) ProcessEventRecords(in chan *records.EventOrientedRecord) chan *records.EventOrientedRecord {

	out := make(chan *records.EventOrientedRecord)
	go func() {
		defer close(out)
		// open the csv file writer, and set the header
		w := csv.NewWriter(r.outF)
		defer r.outF.Close()
		w.Write(r.config.header)
		defer w.Flush()

		for eor := range in {
			if r.config.activated { // only process if active
				if eor.IsWritingResponse() { // only process writing responses
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
				}
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
func (we *CompareItemWriting) calculateFields(eor *records.EventOrientedRecord) []byte {

	ir := eor.GetValueString("NAPStudentResponseSet.TestletList.Testlet.0.ItemResponseList.ItemResponse.0.Response")
	ir = html.UnescapeString(ir) // html of writing response needs unescaping
	wc := strconv.Itoa(countwords(strings.Replace(ir, "\\n", "\n", -1)))

	json := eor.CalculatedFields
	json, _ = sjson.SetBytes(json, "CalculatedFields.WordCount", wc)
	json, _ = sjson.SetBytes(json, "CalculatedFields.EscapedResponse", ir)

	return json
}
