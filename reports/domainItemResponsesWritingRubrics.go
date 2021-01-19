package reports

import (
	"fmt"

	"github.com/iancoleman/strcase"
	"github.com/nsip/dev-nrt/records"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

type DomainItemResponsesWritingRubrics struct {
	baseReport // embed common setup capability
	domainName string
}

//
// Provides a comprehensive detailing of
// each item-level response in each domain tested
// for each student; this is a specialised variant for
// writing that also breaks down the subscores by
// rubric type
//
func DomainItemResponsesWritingRubricsReport() *DomainItemResponsesWritingRubrics {

	r := DomainItemResponsesWritingRubrics{domainName: "Writing"}
	r.initialise("./config/internal/DomainItemResponsesWritingRubrics.toml")
	r.printStatus()

	return &r

}

//
// implement the ...Pipe interface, core work of the
// report engine.
//
func (r *DomainItemResponsesWritingRubrics) ProcessStudentRecords(in chan *records.StudentOrientedRecord) chan *records.StudentOrientedRecord {

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
func (r *DomainItemResponsesWritingRubrics) calculateFields(sor *records.StudentOrientedRecord) []byte {

	// defer utils.TimeTrack(time.Now(), "ItemResponses calc fields()")

	json := sor.CalculatedFields // maintain exsting calc fields

	// iterate the responses of this student, are keyed by camel-case rendering of test domain
	domain := r.domainName
	response := sor.GetResponsesByDomain()[domain]
	//
	// iterate through testlets to the item-level responses
	//
	testletCount := 0
	gjson.GetBytes(response, "NAPStudentResponseSet.TestletList.Testlet").
		ForEach(func(key, value gjson.Result) bool {
			itemResponseCount := 0
			value.Get("ItemResponseList.ItemResponse").
				ForEach(func(key, value gjson.Result) bool {
					//
					//
					// iterate the subscore rubric list for this response
					//
					value.Get("SubscoreList.Subscore").
						ForEach(func(key, value gjson.Result) bool {
							ssType := strcase.ToCamel(value.Get("SubscoreType").String())
							ssValue := value.Get("SubscoreValue").String()
							path := fmt.Sprintf("CalculatedFields.%s.NAPStudentResponseSet.TestletList.Testlet.%d.ItemResponseList.ItemResponse.%d.SubscoreList.Subscore.%s.SubscoreValue", domain, testletCount, itemResponseCount, ssType)
							json, _ = sjson.SetBytes(json, path, ssValue)
							return true // keep iterating
						})

					itemResponseCount++
					return true // keep iterating
				})
			testletCount++
			return true // keep iterating
		})

	return json
}
