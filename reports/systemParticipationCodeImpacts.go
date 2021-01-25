package reports

import (
	"encoding/csv"
	"errors"
	"fmt"

	"github.com/nsip/dev-nrt/records"
	"github.com/tidwall/sjson"
)

type SystemParticipationCodeImpacts struct {
	baseReport // embed common setup capability
}

var (
	errUnexpectedAdaptivePathway = errors.New("Adaptive pathway without student undertaking test")
	errUnexpectedScore           = errors.New("Scored test with status other than P or R")
	errRefusedScore              = errors.New("Non-zero score with status of R")
	errMissingScore              = errors.New("Unscored test with status of P or R")
)

//
// Reports errors when responses contain unexpected information based on the
// participation code
//
func SystemParticipationCodeImpactsReport() *SystemParticipationCodeImpacts {

	r := SystemParticipationCodeImpacts{}
	r.initialise("./config/SystemParticipationCodeImpacts.toml")
	r.printStatus()

	return &r

}

//
// implement the EventPipe interface, core work of the
// report engine.
//
func (r *SystemParticipationCodeImpacts) ProcessEventRecords(in chan *records.EventOrientedRecord) chan *records.EventOrientedRecord {

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

			err := participationImpact(eor) // we report only if errors are found
			if err == nil {
				out <- eor
				continue
			}

			//
			// generate any calculated fields required
			//
			eor.CalculatedFields = r.calculateFields(eor, err)

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
// checks for anomalies in the record.
//
// same checks as n2.
//
func participationImpact(eor *records.EventOrientedRecord) error {

	//
	// fetch properties from record
	//
	participationCode := eor.GetValueString("NAPEventStudentLink.ParticipationCode")
	rawScore := eor.GetValueString("NAPStudentResponseSet.DomainScore.RawScore")
	scaledScore := eor.GetValueString("NAPStudentResponseSet.DomainScore.ScaledScoreValue")
	pathTaken := eor.GetValueString("NAPStudentResponseSet.PathTakenForDomain")
	parallelTest := eor.GetValueString("NAPStudentResponseSet.ParallelTest")

	//
	// test logic ported from n2
	//
	if pathTaken > "" || parallelTest > "" {
		if participationCode != "P" && participationCode != "S" {
			return errUnexpectedAdaptivePathway
		}
	}
	//
	if rawScore > "" || scaledScore > "" {
		if participationCode != "P" && participationCode != "R" {
			return errUnexpectedScore
		}
	}
	//
	if rawScore > "" && nonzero(rawScore) {
		if participationCode == "R" {
			return errRefusedScore
		}
	}
	//
	if rawScore == "" || scaledScore == "" {
		if participationCode == "P" || participationCode == "R" {
			return errMissingScore
		}
	}

	return nil
}

//
// converting string to float to zero is too risky
// so simply test the number against the known possible
// zero representations as a string
//
func nonzero(number string) bool {

	switch {
	case number == "0", number == "0.0", number == "0.00", number == "0.000", number == "00.00":
		return false
	default:
		return true
	}

}

//
// generates a block of json that can be added to the
// record containing values that are not in the original data
//
//
func (r *SystemParticipationCodeImpacts) calculateFields(eor *records.EventOrientedRecord, err error) []byte {

	json := eor.CalculatedFields

	json, _ = sjson.SetBytes(json, "CalculatedFields.ParticipationCodeImpactError", err.Error())

	return json
}
