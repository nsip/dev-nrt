package reports

import (
	"fmt"

	"github.com/nsip/dev-nrt/helper"
	"github.com/nsip/dev-nrt/records"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

type DomainDAC struct {
	baseReport // embed common setup capability
	cfh        helper.CodeframeHelper
	dacs       []string
}

//
// Provides a matrix of DAC values in the form
// DAC Code : true/false
// codes are flagged as true if they were in force
// for the given domain for the student
//
func DomainDACReport(cfh helper.CodeframeHelper) *DomainDAC {

	r := DomainDAC{cfh: cfh, dacs: cfh.GetDACs()}
	r.initialise("./config/internal/DomainDAC.toml")
	r.printStatus()

	return &r

}

//
// implement the ...Pipe interface, core work of the
// report engine.
//
func (r *DomainDAC) ProcessStudentRecords(in chan *records.StudentOrientedRecord) chan *records.StudentOrientedRecord {

	out := make(chan *records.StudentOrientedRecord)
	go func() {
		defer close(out)
		for sor := range in {
			if r.config.activated { // only process if active

				sor.CalculatedFields = r.calculateFields(sor)

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
func (r *DomainDAC) calculateFields(sor *records.StudentOrientedRecord) []byte {

	json := sor.CalculatedFields // maintain exsting calc fields

	// iterate the events of this student, are keyed by camel-case rendering of test domain
	for domain, event := range sor.GetEventsByDomain() {
		//
		// get any active DACs for this student
		//
		activeCodes := map[string]struct{}{}
		gjson.GetBytes(event, "NAPEventStudentLink.Adjustment.PNPCodeList.PNPCode").
			ForEach(func(key, value gjson.Result) bool {
				//pnpcode := value.Get("PNPCode").String() // get pnp code
				pnpcode := value.String()         // get pnp code
				activeCodes[pnpcode] = struct{}{} // log as active
				return true                       // keep iterating
			})
		//
		// then create the overall DAC matrix
		//
		for _, dac := range r.dacs {
			_, active := activeCodes[dac]
			path := fmt.Sprintf("CalculatedFields.%s.NAPEventStudentLink.Adjustment.PNPCode.%s", domain, dac)
			json, _ = sjson.SetBytes(json, path, active)
		}
	}

	return json
}
