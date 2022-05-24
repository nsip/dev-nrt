package reports

import (
	"encoding/csv"
	"fmt"

	"github.com/nsip/dev-nrt/records"
	"github.com/tidwall/sjson"
)

type SystemTestTypeImpacts struct {
	baseReport // embed common setup capability
}

//
// Reports tests for which the response contents are unexpected based on the test domain
//
func SystemTestTypeImpactsReport() *SystemTestTypeImpacts {

	r := SystemTestTypeImpacts{}
	r.initialise("./config/SystemTestTypeImpacts.toml")
	r.printStatus()

	return &r

}

//
// implement the EventPipe interface, core work of the
// report engine.
//
func (r *SystemTestTypeImpacts) ProcessEventRecords(in chan *records.EventOrientedRecord) chan *records.EventOrientedRecord {

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

			tterr := validate(eor)
			if tterr == nil { // only process events which fail validation
				out <- eor
				continue
			}

			//
			// generate any calculated fields required
			//
			eor.CalculatedFields = r.calculateFields(eor, tterr)

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
func (r *SystemTestTypeImpacts) calculateFields(eor *records.EventOrientedRecord, err error) []byte {

	json := eor.CalculatedFields

	json, _ = sjson.SetBytes(json, "CalculatedFields.SystemTestTypeImpactError", err.Error())

	return json
}

func validate(eor *records.EventOrientedRecord) error {

	participationCode := eor.GetValueString("NAPEventStudentLink.ParticipationCode")
	if participationCode != "S" && participationCode != "P" { // only evaluate valid tests
		return nil
	}

	// writing response but adaptive pathway
	if eor.IsWritingResponse() {
		if eor.GetValueString("NAPStudentResponseSet.PathTakenForDomain") != "" ||
			eor.GetValueString("NAPStudentResponseSet.ParallelTest") != "" {
			return errWritingAdaptive
		}
		return nil
	}

	// not a writing response, but no adaptive pathway
	if eor.GetValueString("NAPStudentResponseSet.PathTakenForDomain") == "" ||
		eor.GetValueString("NAPStudentResponseSet.ParallelTest") == "" {
		if eor.GetValueString("NAPStudentResponseSet.DomainScore.RawScore") != "0" {
			return errNonAdaptive
		}
	}

	return nil
}
