package reports

import (
	"encoding/csv"
	"fmt"
	"html"

	"github.com/iancoleman/strcase"
	"github.com/nsip/dev-nrt/codeframe"
	"github.com/nsip/dev-nrt/records"
	"github.com/tidwall/sjson"
)

type ItemWritingPrinting struct {
	baseReport // embed common setup capability
	cfh        codeframe.Helper
}

//
// Summary of response data for writing tests, one row per student
//
func ItemWritingPrintingReport(cfh codeframe.Helper) *ItemWritingPrinting {

	r := ItemWritingPrinting{cfh: cfh}
	r.initialise("./config/ItemWritingPrinting.toml")
	r.printStatus()

	return &r

}

//
// implement the EventPipe interface, core work of the
// report engine.
//
func (r *ItemWritingPrinting) ProcessEventRecords(in chan *records.EventOrientedRecord) chan *records.EventOrientedRecord {

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

			if !eor.IsWritingResponse() { // only deal with writing responses
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
// generates a block of json that can be added to the
// record containing values that are not in the original data
//
//
func (r *ItemWritingPrinting) calculateFields(eor *records.EventOrientedRecord) []byte {

	json := eor.CalculatedFields

	if !eor.ParticipatedInTest() { // only analyse if a full response available
		return json
	}

	var path string
	//
	// extract the test-item attributes
	//
	itemRefId := eor.GetValueString("NAPStudentResponseSet.TestletList.Testlet.0.ItemResponseList.ItemResponse.0.NAPTestItemRefId")
	path = "CalculatedFields.ItemWritingPrinting.NAPTestItem.RefId"
	json, _ = sjson.SetBytes(json, path, itemRefId)
	itemLocalId := r.cfh.GetCodeframeObjectValueString(itemRefId, "NAPTestItem.TestItemContent.NAPTestItemLocalId")
	path = "CalculatedFields.ItemWritingPrinting.NAPTestItem.NAPTestItemLocalId"
	json, _ = sjson.SetBytes(json, path, itemLocalId)
	itemName := r.cfh.GetCodeframeObjectValueString(itemRefId, "NAPTestItem.TestItemContent.ItemName")
	path = "CalculatedFields.ItemWritingPrinting.NAPTestItem.ItemName"
	json, _ = sjson.SetBytes(json, path, itemName)
	itemSubDomain := r.cfh.GetCodeframeObjectValueString(itemRefId, "NAPTestItem.TestItemContent.Subdomain")
	path = "CalculatedFields.ItemWritingPrinting.NAPTestItem.Subdomain"
	json, _ = sjson.SetBytes(json, path, itemSubDomain)
	itemGenre := r.cfh.GetCodeframeObjectValueString(itemRefId, "NAPTestItem.TestItemContent.WritingGenre")
	path = "CalculatedFields.ItemWritingPrinting.NAPTestItem.WritingGenre"
	json, _ = sjson.SetBytes(json, path, itemGenre)

	path = "CalculatedFields.ItemWritingPrinting.NAPTestItem.SubstituteFor"
	subRefIds, ok := r.cfh.IsSubstituteItem(itemRefId)
	if ok {
		json, _ = sjson.SetBytes(json, path, subRefIds)
	}

	//
	// then the writing rubric scores
	//
	scorePath := "CalculatedFields.ItemWritingPrinting.%s.Score"
	for _, rubric := range r.cfh.WritingRubricTypes() {
		query := fmt.Sprintf(`NAPStudentResponseSet.TestletList.Testlet.0.ItemResponseList.ItemResponse.0.SubscoreList.Subscore.#[SubscoreType==%s].SubscoreValue`, rubric)
		rubricScore := eor.GetValueString(query)
		path = fmt.Sprintf(scorePath, strcase.ToCamel(rubric))
		json, _ = sjson.SetBytes(json, path, rubricScore)
	}
	//
	// finally escape the writing response
	//
	ir := eor.GetValueString("NAPStudentResponseSet.TestletList.Testlet.0.ItemResponseList.ItemResponse.0.Response")
	ir = html.UnescapeString(ir) // html of writing response needs unescaping
	json, _ = sjson.SetBytes(json, "CalculatedFields.ItemWritingPrinting.EscapedResponse", ir)

	return json
}
