package reports

import (
	"encoding/csv"
	"fmt"

	"github.com/nsip/dev-nrt/records"
)

type SystemTestYearLevelDiscrepancies struct {
	baseReport // embed common setup capability
}

//
// Reports a row for every test administered at a different test level from the year level that the student was enrolled in
//
func SystemTestYearLevelDiscrepanciesReport() *SystemTestYearLevelDiscrepancies {

	r := SystemTestYearLevelDiscrepancies{}
	r.initialise("./config/SystemTestYearLevelDiscrepancies.toml")
	r.printStatus()

	return &r

}

//
// implement the EventPipe interface, core work of the
// report engine.
//
func (r *SystemTestYearLevelDiscrepancies) ProcessEventRecords(in chan *records.EventOrientedRecord) chan *records.EventOrientedRecord {

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

			if yearLevelsMatch(eor) { // only process if mismatch detected
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
// checks the year level in the student record and the
// yr level in the test record. Returns true if they match
// false otherwise.
//
func yearLevelsMatch(eor *records.EventOrientedRecord) bool {

	testYrLevel := eor.GetValueString("NAPTest.TestContent.TestLevel.Code")
	studentYrLevel := eor.GetValueString("StudentPersonal.MostRecent.YearLevel.Code")

	return (testYrLevel == studentYrLevel)

}

//
// generates a block of json that can be added to the
// record containing values that are not in the original data
//
//
func (r *SystemTestYearLevelDiscrepancies) calculateFields(eor *records.EventOrientedRecord) []byte {

	return eor.CalculatedFields
}
