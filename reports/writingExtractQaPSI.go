package reports

import (
	"encoding/csv"
	"fmt"

	"github.com/nsip/dev-nrt/records"
)

type WritingExtractQaPSI struct {
	baseReport // embed common setup capability
}

//
// Creates adjunct file to the writing extract to
// capture the PSIs of students who have created valid
// writing responses.
// Allows rrd extracts to be taken from platform multiple times
// but only process the responses from users not seen before
//
func WritingExtractQaPSIReport() *WritingExtractQaPSI {

	we := WritingExtractQaPSI{}
	we.initialise("./config/WritingExtractQaPSI.toml")
	we.printStatus()

	return &we

}

//
// implement the EventPipe interface, core work of the
// report engine.
//
func (we *WritingExtractQaPSI) ProcessEventRecords(in chan *records.EventOrientedRecord) chan *records.EventOrientedRecord {

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
			out <- eor
		}
	}()
	return out
}
