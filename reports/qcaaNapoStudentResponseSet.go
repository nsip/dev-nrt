package reports

import (
	"encoding/csv"
	"fmt"

	"github.com/nsip/dev-nrt/records"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

type QcaaNapoStudentResponseSet struct {
	baseReport // embed common setup capability
}

//
// Detailed breakdown of student response object
//
func QcaaNapoStudentResponseSetReport() *QcaaNapoStudentResponseSet {

	r := QcaaNapoStudentResponseSet{}
	r.initialise("./config/QcaaNapoStudentResponseSet.toml")
	r.printStatus()

	return &r

}

//
// implement the EventPipe interface, core work of the
// report engine.
//
func (r *QcaaNapoStudentResponseSet) ProcessEventRecords(in chan *records.EventOrientedRecord) chan *records.EventOrientedRecord {

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

			if !eor.HasNAPStudentResponseSet { // not all events have responses
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
func (r *QcaaNapoStudentResponseSet) calculateFields(eor *records.EventOrientedRecord) []byte {

	ir := gjson.GetBytes(eor.NAPStudentResponseSet, "NAPStudentResponseSet.DomainScore.PlausibleScaledValueList.PlausibleScaledValue").String()
	if len(ir) == 0 {
		ir = "[]"
	}

	json := eor.CalculatedFields
	json, _ = sjson.SetBytes(json, "CalculatedFields.NAPStudentResponseSet.DomainScore.PlausibleScaledValueList.PlausibleScaledValue", ir)

	return json
}
