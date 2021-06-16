package reports

import (
	"encoding/csv"
	"fmt"
	"strings"

	"github.com/nsip/dev-nrt/records"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

//
// data is stored against this set of
// attributes in combination
//
type stcrKey struct {
	schoolACARAId string
	testDomain    string
	testLevel     string
}

//
// capture details of any record with issues
//
type stcrHit struct {
	PSI        string
	EventId    string // refid of napeventstudentlink
	ResponseId string // refid of napstudentresponseset
}

//
// summary object to contain json report strucures
//
type stcrSummary struct {
	CalculatedFields []byte
}

//
// pass a gjson path to retrieve the value at that location as a
// string, makes the summary conform to the standard csv rendering loop
//
func (s *stcrSummary) GetValueString(queryPath string) string {

	//
	// get the root object
	//
	objName := strings.Split(queryPath, ".")[0]
	var data []byte
	switch objName {
	case "CalculatedFields":
		data = s.CalculatedFields
	default:
		data = []byte{}
	}

	return gjson.GetBytes(data, queryPath).String()
}

type participationCodeCounter = map[string]int             // summary of attempts by p-code
type attemptCounter = map[stcrKey]participationCodeCounter // summary of test attempts
type responseCounter = map[stcrKey]int                     // summary of test responses
type attemptsWithNoResponse = map[stcrKey][]stcrHit
type responsesWithNoAttempts = map[stcrKey][]stcrHit

//
// core report data structures
//
type SystemTestCompleteness struct {
	baseReport // embed common setup capability
	ac         attemptCounter
	rc         responseCounter
	anr        attemptsWithNoResponse
	rna        responsesWithNoAttempts
}

//
// Summarises test attempts, and highlights any issues between attemtpts and responses
//
// NOTE: This is a Collector report, it harvests data from the stream but
// only writes it out once all data has passed through.
//
func SystemTestCompletenessReport() *SystemTestCompleteness {

	r := SystemTestCompleteness{
		ac:  make(attemptCounter, 0),
		rc:  make(responseCounter, 0),
		anr: make(attemptsWithNoResponse, 0),
		rna: make(responsesWithNoAttempts, 0),
	}
	r.initialise("./config/SystemTestCompleteness.toml")
	r.printStatus()

	return &r

}

//
// implement the EventPipe interface, core work of the
// report engine.
//
func (r *SystemTestCompleteness) ProcessEventRecords(in chan *records.EventOrientedRecord) chan *records.EventOrientedRecord {

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

			//
			// gather data from the record
			//
			r.assessCompleteness(eor)

			out <- eor // pass on to next stage

			// this is a collector so aggregated data is written out when the
			// input channel closes, see below...
		}

		//
		// write out collected data
		//
		for key := range r.ac {
			// create a new reporting object
			summ := &stcrSummary{CalculatedFields: []byte{}}
			//
			// generate any calculated fields required
			//
			summ.CalculatedFields = r.calculateFields(key)
			//
			// now loop through the ouput definitions to create a
			// row of results
			//
			var result string
			var row []string = make([]string, 0, len(r.config.queries))
			for _, query := range r.config.queries {
				result = summ.GetValueString(query)
				row = append(row, result)
			}
			// write the row to the output file
			if err := w.Write(row); err != nil {
				fmt.Println("Warning: error writing record to csv:", r.config.name, err)
			}
		}

	}()
	return out
}

//
// generates a block of json that can be added to the
// record containing values that are not in the original data
//
//
func (r *SystemTestCompleteness) calculateFields(key stcrKey) []byte {

	json := []byte{}

	// set basic information
	json, _ = sjson.SetBytes(json, "CalculatedFields.SystemTestCompleteness.SchoolInfo.ACARAId", key.schoolACARAId)
	json, _ = sjson.SetBytes(json, "CalculatedFields.SystemTestCompleteness.NAPTest.TestContent.Domain", key.testDomain)
	json, _ = sjson.SetBytes(json, "CalculatedFields.SystemTestCompleteness.NAPTest.TestContent.TestLevel.Code", key.testLevel)
	// maintain the splitter block
	json, _ = sjson.SetBytes(json, "CalculatedFields.SchoolId", key.schoolACARAId)
	json, _ = sjson.SetBytes(json, "CalculatedFields.Domain", key.testDomain)
	json, _ = sjson.SetBytes(json, "CalculatedFields.YrLevel", key.testLevel)

	// set the totals for attempts & responses for this row
	var path string
	var totalAttempts int
	for pCode, count := range r.ac[key] {
		//for _, pCode := range []string{"P", "S", "R"} {
		//	count := r.ac[key][pCode]
		path = fmt.Sprintf("CalculatedFields.SystemTestCompleteness.%s_AttemptsCount", pCode)
		json, _ = sjson.SetBytes(json, path, count)
		switch pCode {
		case "P", "S", "R":
			totalAttempts = totalAttempts + count
		}
	}
	json, _ = sjson.SetBytes(json, "CalculatedFields.SystemTestCompleteness.TotalAttemptsCount", totalAttempts)
	json, _ = sjson.SetBytes(json, "CalculatedFields.SystemTestCompleteness.TotalResponsesCount", r.rc[key])

	// set details of any errors detected
	//
	// attempts-no-response
	//
	for _, hit := range r.anr[key] {
		path = "CalculatedFields.SystemTestCompleteness.ANRList.-1.ANR"
		json, _ = sjson.SetBytes(json, path, hit)
	}
	//
	// response-no-attempt
	//
	for _, hit := range r.rna[key] {
		path = "CalculatedFields.SystemTestCompleteness.RNAList.-1.RNA"
		json, _ = sjson.SetBytes(json, path, hit)
	}

	return json
}

//
// compiles number of attempts by participation code, and captures any records with
// discrepancies between attempts and responses
//
func (r *SystemTestCompleteness) assessCompleteness(eor *records.EventOrientedRecord) {

	participationCode := eor.GetValueString("NAPEventStudentLink.ParticipationCode")

	key := stcrKey{
		schoolACARAId: eor.GetValueString("SchoolInfo.ACARAId"),
		testDomain:    eor.GetValueString("NAPTest.TestContent.Domain"),
		testLevel:     eor.GetValueString("NAPTest.TestContent.TestLevel.Code"),
	}

	hit := stcrHit{
		PSI:     eor.GetValueString("NAPEventStudentLink.PlatformStudentIdentifier"),
		EventId: eor.GetValueString("NAPEventStudentLink.RefId"),
	}

	// update the attempts counter
	if _, ok := r.ac[key]; !ok {
		r.ac[key] = make(participationCodeCounter, 0)
	}
	r.ac[key][participationCode]++

	if eor.HasNAPStudentResponseSet {
		// capture the refid
		hit.ResponseId = eor.GetValueString("NAPStudentResponseSet.RefId")
		r.rc[key]++ // and update the response counter

		switch participationCode {
		case "P", "S", "R":
			break
		default: // if p-code does not normally have a response, then capture it as a response-no-attempt
			if _, ok := r.rna[key]; !ok {
				r.rna[key] = make([]stcrHit, 0)
			}
			r.rna[key] = append(r.rna[key], hit)
		}
	} else {
		switch participationCode {
		case "P", "S", "R": // only keep valid attempts (as per n2 logic)
			// if no response then add details to the attempt-no-response list
			if _, ok := r.anr[key]; !ok {
				r.anr[key] = make([]stcrHit, 0)
			}
			r.anr[key] = append(r.anr[key], hit)
		}
	}

}
