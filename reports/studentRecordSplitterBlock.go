package reports

import (
	"github.com/nsip/dev-nrt/records"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

type StudentRecordSplitterBlock struct {
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
func StudentRecordSplitterBlockReport() *StudentRecordSplitterBlock {

	r := StudentRecordSplitterBlock{}
	r.initialise("./config/internal/StudentRecordSplitterBlock.toml")
	r.printStatus()

	return &r

}

//
// implement the ...Pipe interface, core work of the
// report engine.
//
func (r *StudentRecordSplitterBlock) ProcessStudentRecords(in chan *records.StudentOrientedRecord) chan *records.StudentOrientedRecord {

	out := make(chan *records.StudentOrientedRecord)
	go func() {
		defer close(out)
		for sor := range in {
			if r.config.activated { // only process if active

				sor.CalculatedFields = r.calculateFields(sor)

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
func (r *StudentRecordSplitterBlock) calculateFields(sor *records.StudentOrientedRecord) []byte {

	schoolid := gjson.GetBytes(sor.SchoolInfo, "SchoolInfo.ACARAId")
	yrlvl := gjson.GetBytes(sor.StudentPersonal, "StudentPersonal.MostRecent.TestLevel.Code")
	domain := "All"

	json := sor.CalculatedFields // keep any exisiting settings
	json, _ = sjson.SetBytes(json, "CalculatedFields.SchoolId", schoolid.String())
	json, _ = sjson.SetBytes(json, "CalculatedFields.YrLevel", yrlvl.String())
	json, _ = sjson.SetBytes(json, "CalculatedFields.Domain", domain)

	return json
}
