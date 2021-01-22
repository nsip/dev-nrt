package reports

import (
	"encoding/csv"
	"fmt"
	"strconv"

	"github.com/nsip/dev-nrt/records"
)

type SystemObjectsCount struct {
	baseReport // embed common setup capability
	total      map[string]int
}

//
// simple summary count of all the data objects in the rrd dataset by type
//
// NOTE: This is a Collector report, it harvests data from the stream but
// only writes it out once all data has passed through.
//
func SystemObjectsCountReport() *SystemObjectsCount {

	r := SystemObjectsCount{total: make(map[string]int, 0)}
	r.initialise("./config/SystemObjectsCount.toml")
	r.printStatus()

	return &r

}

//
// implement the ...Pipe interface, core work of the
// report engine.
//
func (r *SystemObjectsCount) ProcessObjectRecords(in chan *records.ObjectRecord) chan *records.ObjectRecord {

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

			//
			// generate any calculated fields required
			//
			or.CalculatedFields = r.calculateFields(or)

			out <- or
		}

		//
		// Collector report so report writing happens after the processing loop
		// when all data has been seen.
		//
		//
		// now loop through the ouput definitions to create a
		// row of results
		//
		var result string
		var row []string = make([]string, 0, len(r.config.queries))
		for _, query := range r.config.queries {
			count, ok := r.total[query]
			if !ok {
				result = "none found"
			} else {
				result = strconv.Itoa(count)
			}
			row = append(row, result)
		}
		// write the row to the output file
		if err := w.Write(row); err != nil {
			fmt.Println("Warning: error writing record to csv:", r.config.name, err)
		}

	}()
	return out
}

//
// generates a block of json that can be added to the
// record containing values that are not in the original data
//
//
func (r *SystemObjectsCount) calculateFields(or *records.ObjectRecord) []byte {

	// update the object totals
	r.total[or.RecordType]++

	return or.CalculatedFields
}
