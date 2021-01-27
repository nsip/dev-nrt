package reports

import (
	"encoding/csv"
	"fmt"

	"github.com/nsip/dev-nrt/records"
)

type SystemStudentEventAcaraIdDiscrepancies struct {
	baseReport // embed common setup capability
}

//
// Reports a row for every test administered at a different school from where the student was enrolled
//
func SystemStudentEventAcaraIdDiscrepanciesReport() *SystemStudentEventAcaraIdDiscrepancies {

	r := SystemStudentEventAcaraIdDiscrepancies{}
	r.initialise("./config/SystemStudentEventAcaraIdDiscrepancies.toml")
	r.printStatus()

	return &r

}

//
// implement the EventPipe interface, core work of the
// report engine.
//
func (r *SystemStudentEventAcaraIdDiscrepancies) ProcessEventRecords(in chan *records.EventOrientedRecord) chan *records.EventOrientedRecord {

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

			if schoolsMatch(eor) { // only process if mismatch detected
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
// checks the acaraid of the schoolin the student record and the
// school in the event record. Returns true if they match
// false otherwise.
//
func schoolsMatch(eor *records.EventOrientedRecord) bool {

	eventSchoolId := eor.GetValueString("NAPEventStudentLink.SchoolACARAId")
	studentSchoolId := eor.GetValueString("StudentPersonal.MostRecent.SchoolACARAId")

	return (eventSchoolId == studentSchoolId)

}

//
// generates a block of json that can be added to the
// record containing values that are not in the original data
//
//
func (r *SystemStudentEventAcaraIdDiscrepancies) calculateFields(eor *records.EventOrientedRecord) []byte {

	return eor.CalculatedFields
}
