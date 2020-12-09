package reports

import (
	"bufio"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/nsip/dev-nrt/codeframe"
	"github.com/nsip/dev-nrt/records"
	"github.com/tidwall/sjson"
)

type NswWritingPearsonY3 struct {
	baseReport // embed common setup capability
	cfh        codeframe.Helper
}

//
// Creates fixed format report of writing rubric scores in pearson format
// of 205 char lines.
//
func NswWritingPearsonY3Report(cfh codeframe.Helper) *NswWritingPearsonY3 {

	r := NswWritingPearsonY3{cfh: cfh}
	r.initialise("./config/NswWritingPearsonY3.toml")
	r.printStatus()

	return &r

}

//
// implement the EventPipe interface, core work of the
// report engine.
//
func (r *NswWritingPearsonY3) ProcessEventRecords(in chan *records.EventOrientedRecord) chan *records.EventOrientedRecord {

	out := make(chan *records.EventOrientedRecord)
	go func() {
		defer close(out)
		// open the fixed format file writer
		w := bufio.NewWriter(r.outF)
		defer r.outF.Close()
		defer w.Flush()

		var result, paddedResult string
		var length int
		var convErr error
		var row strings.Builder // single string record for this format

		for eor := range in {

			if !r.config.activated { // only process if activated
				out <- eor
				continue
			}

			if !eor.IsWritingResponse() { // only process writing responses
				out <- eor
				continue
			}

			if eor.GetValueString("CalculatedFields.YrLevel") != "3" { // only for yr 3
				out <- eor
				continue
			}

			if !eor.HasNAPStudentResponseSet { // extra check for yr3 as responses may be added post-exam
				out <- eor
				continue
			}

			//
			// generate any calculated fields required
			//
			eor.CalculatedFields = r.calculateFields(eor)

			//
			// now loop through the ouput definitions to create a
			// 'row' of results
			//
			for i, query := range r.config.queries {
				result = eor.GetValueString(query)
				length, convErr = strconv.Atoi(r.config.header[i])
				if convErr != nil {
					log.Println("WARNING: unexpected value for field length in ", r.configFileName, r.config.header[i])
				}
				paddedResult = PadLeft(result, length, defaultPaddingToken)
				row.WriteString(paddedResult)
			}
			// write the row to the output file
			if _, err := fmt.Fprintln(w, row.String()); err != nil {
				fmt.Println("Warning: error writing record to output file:", r.config.name, err)
			}
			row.Reset()

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
func (r *NswWritingPearsonY3) calculateFields(eor *records.EventOrientedRecord) []byte {

	json := eor.CalculatedFields // keep any existing data

	// create truncated format birthdate
	fullDob := eor.GetValueString("StudentPersonal.PersonInfo.Demographics.BirthDate")
	truncDob := strings.ReplaceAll(fullDob, "-", "")
	json, _ = sjson.SetBytes(json, "CalculatedFields.TruncatedDOB", truncDob)

	// create blank placeholder
	json, _ = sjson.SetBytes(json, "CalculatedFields.BlankToken", " ")

	// get writing subscores according to the codeframe ordering
	for i, rt := range r.cfh.WritingRubricTypes() {
		// get response value for each rubric
		query := fmt.Sprintf(`NAPStudentResponseSet.TestletList.Testlet.0.ItemResponseList.ItemResponse.0.SubscoreList.Subscore.#[SubscoreType==%s].SubscoreValue`, rt)
		result := eor.GetValueString(query)
		// add each result to calulated fields
		rubricPath := fmt.Sprintf("CalculatedFields.WritingSubscore.%d.Rubric", i)
		resultPath := fmt.Sprintf("CalculatedFields.WritingSubscore.%d.Score", i)
		json, _ = sjson.SetBytes(json, rubricPath, rt)
		json, _ = sjson.SetBytes(json, resultPath, result)
	}

	return json
}
