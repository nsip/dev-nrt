package reports

import (
	"github.com/nsip/dev-nrt/codeframe"
	"github.com/nsip/dev-nrt/records"
	"github.com/tidwall/sjson"
)

type ItemTestLink struct {
	cfh        codeframe.Helper
	baseReport // embed common setup capability
}

//
// Establishes all links between a test item and the
// rest of the test hierachy; item->testlet->test
// noting that items can be re-used across different testlets 
// and possibly even different tests
// 
// 
func ItemTestLinkReport(cfh codeframe.Helper) *ItemTestLink {

	r := ItemTestLink{cfh: cfh}
	r.initialise("./config/internal/ItemTestLink.toml")
	r.printStatus()

	return &r

}

//
// implement the EventPipe interface, core work of the
// report engine.
//
func (r *ItemTestLink) ProcessCodeframeRecords(in chan *records.CodeframeRecord) chan *records.CodeframeRecord {

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

			//
			// get all test containers associated with this item
			//
			for testletRefId, testRefId := range r.cfh.GetContainersForItem(cfr.RefId()) {
				// fetch localids
				testLocalId := r.cfh.GetCodeframeObjectValueString(testRefId, "NAPTest.TestContent.NAPTestLocalId")
				testletLocalId := r.cfh.GetCodeframeObjectValueString(testletRefId, "NAPTestlet.TestletContent.NAPTestletLocalId")
				itemSeqNo := r.cfh.GetItemTestletSequenceNumber(cfr.RefId(), testletRefId)
				// create a copy for each test, and assign the container ids to calculated fields
				calcf, _ := sjson.SetBytes([]byte{}, "CalculatedFields.NAPTestRefId", testRefId)
				calcf, _ = sjson.SetBytes(calcf, "CalculatedFields.NAPTestletRefId", testletRefId)
				calcf, _ = sjson.SetBytes(calcf, "CalculatedFields.NAPTestLocalId", testLocalId)
				calcf, _ = sjson.SetBytes(calcf, "CalculatedFields.NAPTestletLocalId", testletLocalId)
				calcf, _ = sjson.SetBytes(calcf, "CalculatedFields.SequenceNumber", itemSeqNo)
				newcfr := records.CodeframeRecord{
					RecordType:       cfr.RecordType,
					Json:             cfr.Json,
					CalculatedFields: calcf,
				}
				out <- &newcfr
			}
		}
	}()
	return out
}
