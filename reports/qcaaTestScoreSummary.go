package reports

import (
	"encoding/csv"
	"fmt"

	"github.com/nsip/dev-nrt/records"
)

type QcaaTestScoreSummary struct {
	baseReport // embed common setup capability
}

//
// Simple object report, translates object to csv table
//
func QcaaTestScoreSummaryReport() *QcaaTestScoreSummary {

	r := QcaaTestScoreSummary{}
	r.initialise("./config/QcaaTestScoreSummary.toml")
	r.printStatus()

	return &r

}

//
// implement the ...Pipe interface, core work of the
// report engine.
//
func (r *QcaaTestScoreSummary) ProcessObjectRecords(in chan *records.ObjectRecord) chan *records.ObjectRecord {

	out := make(chan *records.ObjectRecord)
	go func() {
		defer close(out)
		// open the csv file writer, and set the header
		w := csv.NewWriter(r.outF)
		defer r.outF.Close()
		w.Write(r.config.header)
		defer w.Flush()

		for or := range in {
			if !r.config.activated { // only process if activated
				out <- or
				continue
			}

			if or.RecordType != "NAPTestScoreSummary" { // only deal with score summaries
				out <- or
				continue
			}

			//
			// generate any calculated fields required
			//
			or.CalculatedFields = r.calculateFields(or)

			//
			// now loop through the ouput definitions to create a
			// row of results
			//
			var result string
			var row []string = make([]string, 0, len(r.config.queries))
			for _, query := range r.config.queries {
				result = or.GetValueString(query)
				row = append(row, result)
			}
			// write the row to the output file
			if err := w.Write(row); err != nil {
				fmt.Println("Warning: error writing record to csv:", r.config.name, err)
			}

			out <- or
		}
	}()
	return out
}

//
// generates a block of json that can be added to the
// record containing values that are not in the original data
//
//
func (r *QcaaTestScoreSummary) calculateFields(or *records.ObjectRecord) []byte {

	return or.CalculatedFields
}
