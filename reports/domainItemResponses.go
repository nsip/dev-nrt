package reports

import (
	"fmt"

	"github.com/nsip/dev-nrt/records"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

type DomainItemResponses struct {
	baseReport // embed common setup capability
	domainName string
}

//
// Provides a comprehensive detailing of
// each item-level response in each domain tested
// for each student
//
func DomainItemResponsesReport(domainName string) *DomainItemResponses {

	r := DomainItemResponses{domainName: domainName}
	r.initialise("./config/internal/DomainItemResponses.toml")
	r.printStatus()

	return &r

}

//
// implement the ...Pipe interface, core work of the
// report engine.
//
func (r *DomainItemResponses) ProcessStudentRecords(in chan *records.StudentOrientedRecord) chan *records.StudentOrientedRecord {

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
func (r *DomainItemResponses) calculateFields(sor *records.StudentOrientedRecord) []byte {

	// defer utils.TimeTrack(time.Now(), "ItemResponses calc fields()")

	json := sor.CalculatedFields // maintain exsting calc fields

	// iterate the responses of this student, are keyed by camel-case rendering of test domain
	domain := r.domainName
	response := sor.GetResponsesByDomain()[domain]
	//
	// iterate through testlets to the item-level responses
	//
	// for domain, response := range sor.GetResponsesByDomain() {
	testletCount := 0
	gjson.GetBytes(response, "NAPStudentResponseSet.TestletList.Testlet").
		ForEach(func(key, value gjson.Result) bool {
			itemResponseCount := 0
			value.Get("ItemResponseList.ItemResponse").
				ForEach(func(key, value gjson.Result) bool {
					//
					// get the item response refid
					//
					itemrefid := value.Get("NAPTestItemRefId").String()
					path := fmt.Sprintf("CalculatedFields.%s.NAPStudentResponseSet.TestletList.Testlet.%d.ItemResponseList.ItemResponse.%d.ItemRefId", domain, testletCount, itemResponseCount)
					json, _ = sjson.SetBytes(json, path, itemrefid)
					//
					// get the item response score
					//
					itemscore := value.Get("Score").String()
					path = fmt.Sprintf("CalculatedFields.%s.NAPStudentResponseSet.TestletList.Testlet.%d.ItemResponseList.ItemResponse.%d.Score", domain, testletCount, itemResponseCount)
					json, _ = sjson.SetBytes(json, path, itemscore)
					//
					//
					//
					itemcorrectness := value.Get("ResponseCorrectness").String()
					path = fmt.Sprintf("CalculatedFields.%s.NAPStudentResponseSet.TestletList.Testlet.%d.ItemResponseList.ItemResponse.%d.ResponseCorrectness", domain, testletCount, itemResponseCount)
					json, _ = sjson.SetBytes(json, path, itemcorrectness)

					// fmt.Printf("\n%s : %s : %s", itemrefid, itemscore, itemcorrectness)
					//
					itemResponseCount++
					return true // keep iterating
				})
			testletCount++
			return true // keep iterating
		})

		// runtime.Gosched()
	// }

	// fmt.Printf("\n%s\n\n", json)

	return json
}
