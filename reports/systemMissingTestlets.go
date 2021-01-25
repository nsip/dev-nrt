package reports

import (
	"encoding/csv"
	"fmt"
	"strings"

	"github.com/nsip/dev-nrt/records"
)

type SystemMissingTestlets struct {
	baseReport // embed common setup capability
}

//
// Flags responses where student has not been presented with the expected number of testlets
//
func SystemMissingTestletsReport() *SystemMissingTestlets {

	r := SystemMissingTestlets{}
	r.initialise("./config/SystemMissingTestlets.toml")
	r.printStatus()

	return &r

}

//
// implement the EventPipe interface, core work of the
// report engine.
//
func (r *SystemMissingTestlets) ProcessEventRecords(in chan *records.EventOrientedRecord) chan *records.EventOrientedRecord {

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

			if !eor.ParticipatedInTest() { // no point checking events unless a 'P' participation code
				out <- eor
				continue
			}

			if !missingTestlets(eor) { // check if this event has an issue
				out <- eor
				continue
			}

			//
			// generate any calculated fields required
			//
			eor.CalculatedFields = r.calculateFields(eor)

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
// determines for this event if the student has been presented with
// the expected number of testlets
//
// testing logic taken from n2
//
func missingTestlets(eor *records.EventOrientedRecord) bool {

	testLevel := eor.GetValueString("NAPTest.TestContent.TestLevel.Code")
	testDomain := eor.GetValueString("NAPTest.TestContent.Domain")
	nodePath := eor.GetValueString("NAPStudentResponseSet.PathTakenForDomain")
	nodesSeen := len(strings.Split(nodePath, ":"))

	var expectednodes int
	switch testDomain {
	case "Writing":
		expectednodes = 1 // see note
	case "Numeracy":
		if testLevel == "7" || testLevel == "9" {
			expectednodes = 4
		} else {
			expectednodes = 3
		}
	case "Reading":
		expectednodes = 3
	case "Grammar and Punctuation":
		expectednodes = 1
	case "Spelling":
		expectednodes = 3
	default:
		expectednodes = -1
	}

	if nodesSeen != expectednodes {
		return true
	}

	return false


	// note: writing test responses have a null path taken member, there's no path
	// just a single testlet.
	// the strings.split function returns an array of length 1 however beacuse
	// if no spliting is available it returns the input - in this case an empty string
	// but which still counts as an array member for the len() call, giving a length
	// of one.

}

//
// generates a block of json that can be added to the
// record containing values that are not in the original data
//
//
func (r *SystemMissingTestlets) calculateFields(eor *records.EventOrientedRecord) []byte {

	return eor.CalculatedFields
}
