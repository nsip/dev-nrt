package reports

import (
	"encoding/csv"
	"fmt"

	"github.com/nsip/dev-nrt/records"
	"github.com/tidwall/gjson"
)

type SystemPNPEvents struct {
	baseReport // embed common setup capability
}

//
// DESCRIPTION HERE
//
func SystemPNPEventsReport() *SystemPNPEvents {

	r := SystemPNPEvents{}
	r.initialise("./config/SystemPNPEvents.toml")
	r.printStatus()

	return &r

}

//
// implement the EventPipe interface, core work of the
// report engine.
//
func (r *SystemPNPEvents) ProcessEventRecords(in chan *records.EventOrientedRecord) chan *records.EventOrientedRecord {

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

			if !hasAdjustments(eor.NAPEventStudentLink) {
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
func (r *SystemPNPEvents) calculateFields(eor *records.EventOrientedRecord) []byte {

	// extract adjustments
	// assign array

	return eor.CalculatedFields
}

//
// checks the event to see if there are adjustments
// for the student
//
func hasAdjustments(napEvent []byte) bool {

	return gjson.GetBytes(napEvent, "NAPEventStudentLink.Adjustment.PNPCodeList").Exists()

}
