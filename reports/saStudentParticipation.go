package reports

import (
	"encoding/csv"
	"fmt"

	"github.com/nsip/dev-nrt/records"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

type SaStudentParticipation struct {
	baseReport // embed common setup capability
}

// summary of each doamin score for each student as requested
// by SA
func SaStudentParticipationReport() *SaStudentParticipation {

	r := SaStudentParticipation{}
	r.initialise("./config/SaStudentParticipation.toml")
	r.printStatus()

	return &r

}

// implement the EventPipe interface, core work of the
// report engine.
func (r *SaStudentParticipation) ProcessStudentRecords(in chan *records.StudentOrientedRecord) chan *records.StudentOrientedRecord {

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

// generates a block of json that can be added to the
// record containing values that are not in the original data
func (r *SaStudentParticipation) calculateFields(sor *records.StudentOrientedRecord) []byte {

	json := sor.CalculatedFields // maintain exsting calc fields

	/* Not currenty using these, they are bogus
	ir := sor.GetValueString("CalculatedFields.Writing.NAPEventStudentLink.ParticipationCode")
	if ir == "P" || ir == "S" {
		ir = "Marking"
	}
	json, _ = sjson.SetBytes(json, "CalculatedFields.WritingStatus", ir)

	ir = sor.GetValueString("CalculatedFields.Reading.NAPEventStudentLink.ParticipationCode")
	if ir == "P" || ir == "S" {
		ir = "Marking"
	}
	json, _ = sjson.SetBytes(json, "CalculatedFields.ReadingStatus", ir)

	ir = sor.GetValueString("CalculatedFields.Numeracy.NAPEventStudentLink.ParticipationCode")
	if ir == "P" || ir == "S" {
		ir = "Marking"
	}
	json, _ = sjson.SetBytes(json, "CalculatedFields.NumeracyStatus", ir)

	ir = sor.GetValueString("CalculatedFields.GrammarAndPunctuation.NAPEventStudentLink.ParticipationCode")
	if ir == "P" || ir == "S" {
		ir = "Marking"
	}
	json, _ = sjson.SetBytes(json, "CalculatedFields.CoLStatus", ir)
	*/

	for _, test := range sor.GetTestsByDomain() {
		testlvl := gjson.GetBytes(test, "NAPTest.TestContent.TestLevel.Code").String()
		path := "CalculatedFields.TestLevel"
		json, _ = sjson.SetBytes(json, path, testlvl)
		break
	}

	return json
}
