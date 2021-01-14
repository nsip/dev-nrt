package reports

import (
	"github.com/iancoleman/strcase"
	"github.com/nsip/dev-nrt/records"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

type EventRecordSplitterBlock struct {
	baseReport // embed common setup capability
}

//
// insert at the start of pipelines to create a common
// calculated fields block of common elements:
// school (ACARAId), Year Level, test domain
// so that post-report splitting process can
// create hiearchies of sub-reports split by
// these attributes.
//
func EventRecordSplitterBlockReport() *EventRecordSplitterBlock {

	r := EventRecordSplitterBlock{}
	r.initialise("./config/internal/EventRecordSplitterBlock.toml")
	r.printStatus()

	return &r

}

//
// implement the EventPipe interface, core work of the
// report engine.
//
func (r *EventRecordSplitterBlock) ProcessEventRecords(in chan *records.EventOrientedRecord) chan *records.EventOrientedRecord {

	out := make(chan *records.EventOrientedRecord)
	go func() {
		defer close(out)
		for eor := range in {
			if r.config.activated { // only process if active

				eor.CalculatedFields = r.calculateFields(eor)

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
func (r *EventRecordSplitterBlock) calculateFields(eor *records.EventOrientedRecord) []byte {

	schoolid := gjson.GetBytes(eor.SchoolInfo, "SchoolInfo.ACARAId")
	yrlvl := gjson.GetBytes(eor.NAPTest, "NAPTest.TestContent.TestLevel.Code")
	domain := gjson.GetBytes(eor.NAPTest, "NAPTest.TestContent.Domain")
	ccdomain := strcase.ToCamel(domain.String()) // as may be used for directory, convert to camelcase

	json := eor.CalculatedFields // keep any exisiting settings
	json, _ = sjson.SetBytes(json, "CalculatedFields.SchoolId", schoolid.String())
	json, _ = sjson.SetBytes(json, "CalculatedFields.YrLevel", yrlvl.String())
	json, _ = sjson.SetBytes(json, "CalculatedFields.Domain", ccdomain)

	return json
}
