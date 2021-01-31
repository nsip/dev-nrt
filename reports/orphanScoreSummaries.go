package reports

import (
	"encoding/csv"
	"fmt"

	"github.com/nsip/dev-nrt/records"
)

type OrphanScoreSummaries struct {
	baseReport // embed common setup capability
	schools    map[string]struct{}
	summaries  []*records.ObjectRecord
}

//
// Reports any score summary objects where the schoolinfo they relate to is not a
// registered school in the rrd dataset
//
func OrphanScoreSummariesReport() *OrphanScoreSummaries {

	r := OrphanScoreSummaries{
		schools:   make(map[string]struct{}, 0),
		summaries: make([]*records.ObjectRecord, 0),
	}
	r.initialise("./config/OrphanScoreSummaries.toml")
	r.printStatus()

	return &r

}

//
// implement the ...Pipe interface, core work of the
// report engine.
//
func (r *OrphanScoreSummaries) ProcessObjectRecords(in chan *records.ObjectRecord) chan *records.ObjectRecord {

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

			// only process score summaries and school infos
			if or.RecordType != "NAPTestScoreSummary" && or.RecordType != "SchoolInfo" {
				out <- or
				continue
			}

			//
			// generate any calculated fields required
			//
			or.CalculatedFields = r.calculateFields(or)

			// pass on to next stage, writing is done at end of stream when input
			// channel closes, see below...
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
		for _, or := range r.getOrphans() {
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
		}

	}()
	return out
}

//
// intersects the collections of schoolinfos & summaries to determine
// orphans
// returns a new collection of objectRecords to then iterate and apply
// the usual csv writing cycle
//
func (r *OrphanScoreSummaries) getOrphans() []*records.ObjectRecord {

	orphans := make([]*records.ObjectRecord, 0)

	var schoolACARAId string
	for _, summary := range r.summaries {
		schoolACARAId = summary.GetValueString("NAPTestScoreSummary.SchoolACARAId")
		if _, ok := r.schools[schoolACARAId]; !ok { // see if the referenced school is in the stored set
			orphans = append(orphans, summary) // if not, this is an orphan
		}
	}

	return orphans
}

//
// generates a block of json that can be added to the
// record containing values that are not in the original data
//
//
func (r *OrphanScoreSummaries) calculateFields(or *records.ObjectRecord) []byte {

	// no calulations required, just store the data internally for use when
	// stream has completed
	//
	//
	// register a 'hit' for the school by adding it to the set of
	// known school ids
	//
	if or.RecordType == "SchoolInfo" {
		schoolACARAId := or.GetValueString("SchoolInfo.ACARAId")
		r.schools[schoolACARAId] = struct{}{}
	}
	//
	// keep a copy of the score summary object
	//
	if or.RecordType == "NAPTestScoreSummary" {
		r.summaries = append(r.summaries, or)
	}

	// return input as a no-op
	return or.CalculatedFields
}
