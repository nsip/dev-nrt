package reports

import (
	"encoding/csv"
	"fmt"

	"github.com/nsip/dev-nrt/records"
)

type SaHomeschooledTests struct {
	baseReport // embed common setup capability
}

//
// summary of each doamin score for each student as requested
// by ACT
//
func SaHomeschooledTestsReport() *SaHomeschooledTests {

	r := SaHomeschooledTests{}
	r.initialise("./config/SaHomeschooledTests.toml")
	r.printStatus()

	return &r

}

//
// implement the EventPipe interface, core work of the
// report engine.
//
func (r *SaHomeschooledTests) ProcessEventRecords(in chan *records.EventOrientedRecord) chan *records.EventOrientedRecord {

	out := make(chan *records.EventOrientedRecord)
	go func() {
		defer close(out)
		// open the csv file writer, and set the header
		w := csv.NewWriter(r.outF)
		w.Write(r.config.header)
		defer w.Flush()

		for eor := range in {
			if r.config.activated { // only process if active
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
