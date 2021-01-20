package reports

import (
	"fmt"

	"github.com/nsip/dev-nrt/records"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

type DomainParticipation struct {
	baseReport // embed common setup capability
}

//
// Extracts participation info on a per-domain basis
// from the student-oriented record, and inserts into
// calc fields for use by reports
//
func DomainParticipationReport() *DomainParticipation {

	r := DomainParticipation{}
	r.initialise("./config/internal/DomainParticipation.toml")
	r.printStatus()

	return &r

}

//
// implement the ...Pipe interface, core work of the
// report engine.
//
func (r *DomainParticipation) ProcessStudentRecords(in chan *records.StudentOrientedRecord) chan *records.StudentOrientedRecord {

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
func (r *DomainParticipation) calculateFields(sor *records.StudentOrientedRecord) []byte {

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
