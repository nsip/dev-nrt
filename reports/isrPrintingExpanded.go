package reports

import (
	"encoding/csv"
	"fmt"

	"github.com/nsip/dev-nrt/records"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

type IsrPrintingExpanded struct {
	baseReport // embed common setup capability
}

//
// ISR Score summaries with extened personal and response info
//
func IsrPrintingExpandedReport() *IsrPrintingExpanded {

	r := IsrPrintingExpanded{}
	r.initialise("./config/IsrPrintingExpanded.toml")
	r.printStatus()

	return &r

}

//
// implement the ...Pipe interface, core work of the
// report engine.
//
func (r *IsrPrintingExpanded) ProcessStudentRecords(in chan *records.StudentOrientedRecord) chan *records.StudentOrientedRecord {

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
func (r *IsrPrintingExpanded) calculateFields(sor *records.StudentOrientedRecord) []byte {

	json := sor.CalculatedFields // maintain exsting calc fields

	// iterate the responses of this student, are keyed by camel-case rendering of test domain
	for domain, event := range sor.GetResponsesByDomain() {
		//
		// get the test path taken
		//
		ptfd := gjson.GetBytes(event, "NAPStudentResponseSet.PathTakenForDomain").String()
		// we need to separate the results by domain so create domain-based lookup path
		path := fmt.Sprintf("CalculatedFields.%s.NAPStudentResponseSet.PathTakenForDomain", domain)
		// finally assign the test-path code back into the domain-specific lookup in calc fields
		json, _ = sjson.SetBytes(json, path, ptfd)
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
	}

	//
	// the expanded ISR has slots for student address info, but these are not supplied by the RRD
	// so we add a generic placeholder, the columns can't be removed as they are input to
	// downstream printing systems
	//
	json, _ = sjson.SetBytes(json, "CalculatedFields.AddressPlaceholder", "")

	return json

}
