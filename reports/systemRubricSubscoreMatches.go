package reports

import (
	"encoding/csv"
	"fmt"

	"github.com/fatih/set"
	"github.com/iancoleman/strcase"
	"github.com/nsip/dev-nrt/codeframe"
	"github.com/nsip/dev-nrt/records"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

type SystemRubricSubscoreMatches struct {
	baseReport // embed common setup capability
	cfh        codeframe.Helper
}

type mismatches struct {
	ExpectedRubricsNotUsed string
	UsedRubricsNotExpected string
	SubscoresNotDefined    string
	RubricsNotScored       string
	itemName               string
	itemLocalId            string
}

//
// Checks writing responses to see if there are any problems with the
// subscores (i.e. missing), or if subscores do not match the expected list
// for the item.
//
func SystemRubricSubscoreMatchesReport(cfh codeframe.Helper) *SystemRubricSubscoreMatches {

	r := SystemRubricSubscoreMatches{cfh: cfh}
	r.initialise("./config/SystemRubricSubscoreMatches.toml")
	r.printStatus()

	return &r

}

//
// implement the EventPipe interface, core work of the
// report engine.
//
func (r *SystemRubricSubscoreMatches) ProcessEventRecords(in chan *records.EventOrientedRecord) chan *records.EventOrientedRecord {

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

			if !eor.IsWritingResponse() { // only check writing responses
				out <- eor
				continue
			}

			mismatch, found := r.validateSubscoreMatches(eor)
			if !found {
				out <- eor
				continue
			}

			//
			// generate any calculated fields required
			//
			eor.CalculatedFields = r.calculateFields(eor, mismatch)

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
// performs the multiple logic checks against the sources of scores/rubrics:
//
// - are any rubrics unused
// - rubric was used but was not expected
// - rubric was correct for item but not in codeframe
// - rubric was present bu not scored
//
// returns true with a populated mismatches structure if any errors found
// otherwise returns false and nil.
//
func (r *SystemRubricSubscoreMatches) validateSubscoreMatches(eor *records.EventOrientedRecord) (*mismatches, bool) {

	itemRubricTypes := set.New(set.NonThreadSafe)      // actual in item presented
	codeframeRubricTypes := set.New(set.NonThreadSafe) // expected in codeframe
	subscoreTypes := set.New(set.NonThreadSafe)        // scoretypes in response

	//
	// get rubrics / scores from the record
	//
	var path string
	path = "NAPStudentResponseSet.TestletList.Testlet.0.ItemResponseList.ItemResponse.0.SubscoreList.Subscore"
	gjson.GetBytes(eor.NAPStudentResponseSet, path). // get the subscores
								ForEach(func(key, value gjson.Result) bool {
			st := value.Get("SubscoreType").String()
			subscoreTypes.Add(strcase.ToCamel(st))
			return true // keep iterating
		})
	//
	// get the item rubrics
	//
	itemRefId := eor.GetValueString("NAPStudentResponseSet.TestletList.Testlet.0.ItemResponseList.ItemResponse.0.NAPTestItemRefId")
	ok, item := r.cfh.GetItem(itemRefId)
	if !ok {
		return nil, false // no item means response is only partial, so no analysis possible
	}
	path = "NAPTestItem.TestItemContent.NAPWritingRubricList.NAPWritingRubric"
	gjson.GetBytes(item, path). // get the item rubric types
					ForEach(func(key, value gjson.Result) bool {
			rt := value.Get("RubricType").String()
			itemRubricTypes.Add(strcase.ToCamel(rt))
			return true // keep iterating
		})
	//
	// finally the 'official' rubrics from the codeframe
	//
	for _, codeframeRubric := range r.cfh.WritingRubricTypes() {
		codeframeRubricTypes.Add(strcase.ToCamel(codeframeRubric))
	}

	//
	// now check for mismatches
	//
	rubricNotUsed := set.Difference(itemRubricTypes, subscoreTypes)
	rubricNotExpected := set.Difference(subscoreTypes, itemRubricTypes)
	subscoreNotDefined := set.Difference(codeframeRubricTypes, itemRubricTypes)
	rubricsNotScored := set.Difference(itemRubricTypes, codeframeRubricTypes)

	if (rubricNotUsed.Size() + rubricNotExpected.Size() +
		subscoreNotDefined.Size() + rubricsNotScored.Size()) > 0 {
		m := mismatches{
			itemName:    r.cfh.GetCodeframeObjectValueString(itemRefId, "NAPTestItem.TestItemContent.ItemName"),
			itemLocalId: r.cfh.GetCodeframeObjectValueString(itemRefId, "NAPTestItem.TestItemContent.NAPTestItemLocalId"),
		}
		//
		// only add data to results if present, avoid noise in report of empty arrays
		//
		if rubricNotUsed.Size() > 0 {
			m.ExpectedRubricsNotUsed = rubricNotUsed.String()
		}
		if rubricNotExpected.Size() > 0 {
			m.UsedRubricsNotExpected = rubricNotExpected.String()
		}
		if subscoreNotDefined.Size() > 0 {
			m.SubscoresNotDefined = subscoreNotDefined.String()
		}
		if rubricsNotScored.Size() > 0 {
			m.RubricsNotScored = rubricsNotScored.String()
		}

		return &m, true
	}

	return nil, false
}

//
// generates a block of json that can be added to the
// record containing values that are not in the original data
//
//
func (r *SystemRubricSubscoreMatches) calculateFields(eor *records.EventOrientedRecord, mm *mismatches) []byte {

	json := eor.CalculatedFields // maintain any existing calc fields

	var path string
	path = "CalculatedFields.RubricSubscoreMatches.TestItem.TestItemContent.NAPTestItemLocalId"
	json, _ = sjson.SetBytes(json, path, mm.itemLocalId)
	path = "CalculatedFields.RubricSubscoreMatches.TestItem.TestItemContent.ItemName"
	json, _ = sjson.SetBytes(json, path, mm.itemName)

	path = "CalculatedFields.RubricSubscoreMatches.ExpectedRubricsNotUsed"
	json, _ = sjson.SetBytes(json, path, mm.ExpectedRubricsNotUsed)
	path = "CalculatedFields.RubricSubscoreMatches.UsedRubricsNotExpected"
	json, _ = sjson.SetBytes(json, path, mm.UsedRubricsNotExpected)
	path = "CalculatedFields.RubricSubscoreMatches.SubscoresNotDefined"
	json, _ = sjson.SetBytes(json, path, mm.SubscoresNotDefined)
	path = "CalculatedFields.RubricSubscoreMatches.RubricsNotScored"
	json, _ = sjson.SetBytes(json, path, mm.RubricsNotScored)

	return json
}
