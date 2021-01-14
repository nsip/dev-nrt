package reports

import (
	"encoding/csv"
	"fmt"

	"github.com/nsip/dev-nrt/records"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

type SystemParticipation struct {
	baseReport // embed common setup capability
}

//
// Displays Participation Code per student for each test domain
//
func SystemParticipationReport() *SystemParticipation {

	r := SystemParticipation{}
	r.initialise("./config/SystemParticipation.toml")
	r.printStatus()

	return &r

}

//
// implement the ...Pipe interface, core work of the
// report engine.
//
func (r *SystemParticipation) ProcessStudentRecords(in chan *records.StudentOrientedRecord) chan *records.StudentOrientedRecord {

	out := make(chan *records.StudentOrientedRecord)
	go func() {
		defer close(out)
		// open the csv file writer, and set the header
		w := csv.NewWriter(r.outF)
		defer r.outF.Close()
		w.Write(r.config.header)
		defer w.Flush()

		for sor := range in {
			if !r.config.activated { // only process if activated
				out <- sor
				continue
			}

			//
			// generate any calculated fields required
			//
			sor.CalculatedFields = r.calculateFields(sor)

			//
			// now loop through the ouput definitions to create a
			// row of results
			//
			var result string
			var row []string = make([]string, 0, len(r.config.queries))
			for _, query := range r.config.queries {
				result = sor.GetValueString(query)
				row = append(row, result)
			}
			// write the row to the output file
			if err := w.Write(row); err != nil {
				fmt.Println("Warning: error writing record to csv:", r.config.name, err)
			}

			out <- sor
		}
	}()
	return out
}

//
// generates a block of json that can be added to the
// record containing values that are not in the original data
//
//
func (r *SystemParticipation) calculateFields(sor *records.StudentOrientedRecord) []byte {

	json := sor.CalculatedFields // maintain exsting calc fields

	// iterate the events of this student, are keyed by camel-case rendering of test domain
	for domain, event := range sor.GetEventsByDomain() {
		// get the particpation code
		code := gjson.GetBytes(event, "NAPEventStudentLink.ParticipationCode").String()
		// we need to separate the results by domain so create domain-based lookup path
		path := fmt.Sprintf("CalculatedFields.%s.NAPEventStudentLink.ParticipationCode", domain)
		// finally assign the event code back into the domain-specific lookup in calc fields
		json, _ = sjson.SetBytes(json, path, code)
	}

	return json
}
