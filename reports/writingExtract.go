package reports

import (
	"encoding/csv"
	"fmt"
	"html"
	"strconv"
	"strings"

	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/nsip/dev-nrt/records"
	"github.com/tidwall/sjson"
)

type WritingExtract struct {
	baseReport // embed common setup capability
}

// creates file used to feed student writing responses into
// local marking systems
func WritingExtractReport() *WritingExtract {

	we := WritingExtract{}
	we.initialise("./config/WritingExtract.toml")
	we.printStatus()

	return &we

}

// implement the EventPipe interface, core work of the
// report engine.
func (we *WritingExtract) ProcessEventRecords(in chan *records.EventOrientedRecord) chan *records.EventOrientedRecord {

	out := make(chan *records.EventOrientedRecord)
	go func() {
		defer close(out)
		// open the csv file writer, and set the header
		w := csv.NewWriter(we.outF)
		defer we.outF.Close()
		w.Write(we.config.header)
		defer w.Flush()

		for eor := range in {
			if we.config.activated { // only process if active
				if eor.IsWritingResponse() { // only process writing responses
					// if there is no response recorded and no start time,
					// consider this to be an open event, and ignore it
					skip := false
					if len(eor.GetValueString("NAPStudentResponseSet.TestletList.Testlet.0.ItemResponseList.ItemResponse.0.Response")) == 0 {
						participationCode := eor.GetValueString("NAPEventStudentLink.ParticipationCode")
						startTime := eor.GetValueString("NAPEventStudentLink.StartTime")
						if participationCode == "P" && len(startTime) == 0 {
							skip = true
						}
					}

					if !skip {
						//
						// generate any calculated fields required
						//
						eor.CalculatedFields = we.calculateFields(eor)
						//
						// now loop through the ouput definitions to create a
						// row of results
						//
						var result string
						var row []string = make([]string, 0, len(we.config.queries))
						for _, query := range we.config.queries {
							result = eor.GetValueString(query)
							row = append(row, result)
						}
						// write the row to the output file
						if err := w.Write(row); err != nil {
							fmt.Println("Warning: error writing record to csv:", we.config.name, err)
						}
					}
				}
			}
			out <- eor
		}
	}()
	return out

}

// generates a block of json that can be added to the
// record containing values that are not in the original data
func (we *WritingExtract) calculateFields(eor *records.EventOrientedRecord) []byte {

	anonid := gonanoid.MustGenerate("ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789", 16)
	ir := eor.GetValueString("NAPStudentResponseSet.TestletList.Testlet.0.ItemResponseList.ItemResponse.0.Response")
	ir = html.UnescapeString(ir) // html of writing response needs unescaping
	wc := strconv.Itoa(countwords(strings.Replace(strings.Replace(ir, "\\n", "\n", -1), "\\r", "\r", -1)))

	json := eor.CalculatedFields
	json, _ = sjson.SetBytes(json, "CalculatedFields.AnonId", anonid)
	json, _ = sjson.SetBytes(json, "CalculatedFields.WordCount", wc)
	json, _ = sjson.SetBytes(json, "CalculatedFields.EscapedResponse", ir)

	return json
}
