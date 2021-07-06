package reports

import (
	"encoding/csv"
	"fmt"

	"github.com/nsip/dev-nrt/records"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	set "gopkg.in/fatih/set.v0"
)

type SystemObjectFrequency struct {
	baseReport // embed common setup capability
}

//
// Reports comparison of events and responses for each student
//
func SystemObjectFrequencyReport() *SystemObjectFrequency {

	r := SystemObjectFrequency{}
	r.initialise("./config/SystemObjectFrequency.toml")
	r.printStatus()

	return &r

}

//
// implement the ...Pipe interface, core work of the
// report engine.
//
func (r *SystemObjectFrequency) ProcessStudentRecords(in chan *records.StudentOrientedRecord) chan *records.StudentOrientedRecord {

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
func (r *SystemObjectFrequency) calculateFields(sor *records.StudentOrientedRecord) []byte {

	json := sor.CalculatedFields
	numEvents := 0
	numPRSEvents := 0
	numResponses := 0

	var path, flag string
	prs_events_set := set.New(set.ThreadSafe).(*set.Set)
	response_set := set.New(set.ThreadSafe).(*set.Set)

	// analyse the events
	for domain, event := range sor.GetEventsByDomain() {
		numEvents++
		path = fmt.Sprintf("CalculatedFields.ObjectFrequency.%s.NAPEventStudentLink", domain)
		flag = "" // default, makes gaps easier to see in report than 'no'
		response_set.Add(eventcode(event))
		if prs(event) {
			numPRSEvents++
			prs_events_set.Add(eventcode(event))
			flag = "yes"
		}
		json, _ = sjson.SetBytes(json, path, flag)
	}
	// analyse the responses
	for domain := range sor.GetResponsesByDomain() {
		numResponses++
		path = fmt.Sprintf("CalculatedFields.ObjectFrequency.%s.NAPStudentResponseSet", domain)
		json, _ = sjson.SetBytes(json, path, "yes")
	}

	// output totals
	json, _ = sjson.SetBytes(json, "CalculatedFields.ObjectFrequency.EventsCount", numEvents)
	json, _ = sjson.SetBytes(json, "CalculatedFields.ObjectFrequency.PRSEventsCount", numPRSEvents)
	json, _ = sjson.SetBytes(json, "CalculatedFields.ObjectFrequency.ResponsesCount", numResponses)

	json, _ = sjson.SetBytes(json, "CalculatedFields.ObjectFrequency.PRS_Events_Without_Responses", set.Difference(prs_events_set, response_set).String())
	json, _ = sjson.SetBytes(json, "CalculatedFields.ObjectFrequency.Responses_Without_PRS_Events", set.Difference(response_set, prs_events_set).String())

	return json
}

func eventcode(event []byte) string {
	return gjson.GetBytes(event, "SchoolDetails.ACARAId").String() + ":" + gjson.GetBytes(event, "NAPTest.TestContent.TestLevel").String() + ":" + gjson.GetBytes(event, "NAPTest.TestContent.TestDomain").String()
}

//
// helper function to check participation code is one of
// Participated/Refused/Sanctioned Abandonment
//
func prs(event []byte) bool {

	particpationCode := gjson.GetBytes(event, "NAPEventStudentLink.ParticipationCode").String()
	switch particpationCode {
	case "P", "R", "S", "F":
		return true
	}

	return false
}
