package reports

import (
	"github.com/nsip/dev-nrt/records"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

type ItemRubricExtractor struct {
	baseReport // embed common setup capability
}

//
// Mutiplexes a wriitng item into an item per
// writing rubric
//
func ItemRubricExtractorReport() *ItemRubricExtractor {

	r := ItemRubricExtractor{}
	r.initialise("./config/internal/ItemRubricExtractor.toml")
	r.printStatus()

	return &r

}

//
// implement the EventPipe interface, core work of the
// report engine.
//
func (r *ItemRubricExtractor) ProcessCodeframeRecords(in chan *records.CodeframeRecord) chan *records.CodeframeRecord {

	out := make(chan *records.CodeframeRecord)
	go func() {
		defer close(out)

		for cfr := range in {
			if !r.config.activated { // only process if activated
				out <- cfr
				continue
			}

			if cfr.RecordType != "NAPTestItem" { // only deal with test items
				out <- cfr
				continue
			}

			// only items with writing rubrics
			if !gjson.GetBytes(cfr.Json, "NAPTestItem.TestItemContent.NAPWritingRubricList").Exists() {
				out <- cfr
				continue
			}

			//
			// iterate rubric list if one exists
			//
			gjson.GetBytes(cfr.Json, "NAPTestItem.TestItemContent.NAPWritingRubricList.NAPWritingRubric").
				ForEach(func(key, value gjson.Result) bool {
					// get rubric values
					rubricType := value.Get("RubricType").String()
					rubricDescriptor := value.Get("Descriptor").String()
					rubricMaxScore := value.Get("ScoreList.Score.0.MaxScoreValue").String()
					// add to calc fields
					calcf := cfr.CalculatedFields // get current calc fields
					calcf, _ = sjson.SetBytes(calcf, "CalculatedFields.NAPWritingRubric.RubricType", rubricType)
					calcf, _ = sjson.SetBytes(calcf, "CalculatedFields.NAPWritingRubric.Descriptor", rubricDescriptor)
					calcf, _ = sjson.SetBytes(calcf, "CalculatedFields.NAPWritingRubric.MaxScoreValue", rubricMaxScore)
					// create new record
					newcfr := records.CodeframeRecord{
						RecordType:       cfr.RecordType,
						Json:             cfr.Json,
						CalculatedFields: calcf,
					}
					out <- &newcfr

					return true // keep iterating
				})

		}
	}()
	return out
}
