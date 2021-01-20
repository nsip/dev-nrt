package reports

import (
	"fmt"

	"github.com/nsip/dev-nrt/records"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

type DomainResponsesScores struct {
	baseReport // embed common setup capability
}

//
// Extracts scoring info on a per-domain basis
// from the student-oriented record, and inserts into
// calc fields for use by reports
//
func DomainResponsesScoresReport() *DomainResponsesScores {

	r := DomainResponsesScores{}
	r.initialise("./config/internal/DomainResponsesScores.toml")
	r.printStatus()

	return &r

}

//
// implement the ...Pipe interface, core work of the
// report engine.
//
func (r *DomainResponsesScores) ProcessStudentRecords(in chan *records.StudentOrientedRecord) chan *records.StudentOrientedRecord {

	out := make(chan *records.StudentOrientedRecord)
	go func() {
		defer close(out)
		for sor := range in {
			if !r.config.activated { // only process if active
				out <- sor
				continue
			}

			sor.CalculatedFields = r.calculateFields(sor)

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
func (r *DomainResponsesScores) calculateFields(sor *records.StudentOrientedRecord) []byte {

	json := sor.CalculatedFields // maintain exsting calc fields

	// iterate the responses of this student, are keyed by camel-case rendering of test domain
	for domain, event := range sor.GetResponsesByDomain() {
		//
		// get the scaled score
		//
		score := gjson.GetBytes(event, "NAPStudentResponseSet.DomainScore.ScaledScoreValue").String()
		path := fmt.Sprintf("CalculatedFields.%s.NAPStudentResponseSet.DomainScore.ScaledScoreValue", domain)
		json, _ = sjson.SetBytes(json, path, score)
		//
		// get the raw score
		//
		rawscore := gjson.GetBytes(event, "NAPStudentResponseSet.DomainScore.RawScore").String()
		path = fmt.Sprintf("CalculatedFields.%s.NAPStudentResponseSet.DomainScore.RawScore", domain)
		json, _ = sjson.SetBytes(json, path, rawscore)
		//
		// get the test path taken
		//
		ptfd := gjson.GetBytes(event, "NAPStudentResponseSet.PathTakenForDomain").String()
		path = fmt.Sprintf("CalculatedFields.%s.NAPStudentResponseSet.PathTakenForDomain", domain)
		json, _ = sjson.SetBytes(json, path, ptfd)
		//
		// get the parallel test path
		//
		prrl := gjson.GetBytes(event, "NAPStudentResponseSet.ParallelTest").String()
		path = fmt.Sprintf("CalculatedFields.%s.NAPStudentResponseSet.ParallelTest", domain)
		json, _ = sjson.SetBytes(json, path, prrl)

		//
		// get the stnd-deviation for the domain score
		//
		ssse := gjson.GetBytes(event, "NAPStudentResponseSet.DomainScore.ScaledScoreStandardError").String()
		path = fmt.Sprintf("CalculatedFields.%s.NAPStudentResponseSet.DomainScore.ScaledScoreStandardError", domain)
		json, _ = sjson.SetBytes(json, path, ssse)
		//
		// get the domain band
		//
		domb := gjson.GetBytes(event, "NAPStudentResponseSet.DomainScore.StudentDomainBand").String()
		path = fmt.Sprintf("CalculatedFields.%s.NAPStudentResponseSet.DomainScore.StudentDomainBand", domain)
		json, _ = sjson.SetBytes(json, path, domb)
		//
		// get the refid
		//
		refid := gjson.GetBytes(event, "NAPStudentResponseSet.RefId").String()
		path = fmt.Sprintf("CalculatedFields.%s.NAPStudentResponseSet.RefId", domain)
		json, _ = sjson.SetBytes(json, path, refid)
	}

	// iterate the school score summaries of this student, are keyed by camel-case rendering of test domain
	for domain, summ := range sor.GetScoreSummariesByDomain() {
		//
		// get the domain average
		//
		avg := gjson.GetBytes(summ, "NAPTestScoreSummary.DomainNationalAverage").String()
		path := fmt.Sprintf("CalculatedFields.%s.NAPTestScoreSummary.DomainNationalAverage", domain)
		json, _ = sjson.SetBytes(json, path, avg)
	}

	return json

}
