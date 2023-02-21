package reports

import (
	"os"
	"path"
	"strconv"

	"github.com/nsip/dev-nrt/helper"
	"github.com/nsip/dev-nrt/records"
	"github.com/tidwall/gjson"
)

type XMLPerSchoolOutput struct {
	baseReport // embed common setup capability
	cfh        helper.ObjectHelper
}

//
// XML redaction, emulating NAPLAN API output
//
func XmlPerSchoolOutputReport(cfh helper.ObjectHelper) *XMLPerSchoolOutput {

	r := XMLPerSchoolOutput{cfh: cfh}
	r.initialise("./config/XMLPerSchoolOutput.toml")
	r.printStatus()

	return &r
}

//
// implement the ...Pipe interface, core work of the
// report engine.
//
func (r *XMLPerSchoolOutput) ProcessObjectRecords(in chan *records.ObjectRecord) chan *records.ObjectRecord {

	out := make(chan *records.ObjectRecord)
	schools := r.cfh.GetSchoolRefIds()
	r.outF.Close()
	os.Remove(r.config.outputFileName)
	files := make(map[string]*os.File, 0)

	for _, refid := range schools {
		filename := path.Dir(r.config.outputFileName) + "/schooldata_" + refid + ".xml"
		acaraid := r.cfh.GetSchoolFromGuid(refid)
		files[acaraid], _ = os.Create(filename)
		_, _ = files[acaraid].Write([]byte("<sif xmlns=\"http://www.sifassociation.org/datamodel/au/3.4\">\n"))
	}
	filename := path.Dir(r.config.outputFileName) + "/testdata.xml"
	testdata, _ := os.Create(filename)
	_, _ = testdata.Write([]byte("<sif xmlns=\"http://www.sifassociation.org/datamodel/au/3.4\">\n"))
	filename = path.Dir(r.config.outputFileName) + "/schoollist.xml"
	schoollist, _ := os.Create(filename)
	_, _ = schoollist.Write([]byte("<sif xmlns=\"http://www.sifassociation.org/datamodel/au/3.4\">\n"))

	go func() {

		for or := range in {
			if !r.config.activated { // only process if activated
				out <- or
				continue
			}
			//
			// generate any calculated fields required
			//
			or.CalculatedFields = r.calculateFields(or)

			result := gjson.GetBytes(or.CalculatedFields, "CalculatedFields.Redacted")
			var raw []byte
			if result.Index > 0 {
				raw = or.CalculatedFields[result.Index : result.Index+len(result.Raw)]
			} else {
				raw = []byte(result.Raw)
			}
			out1, _ := strconv.Unquote(string(raw))

			switch or.RecordType {
			case "SchoolInfo":
				_, _ = schoollist.Write([]byte(out1))
			case "StudentPersonal", "NAPStudentResponseSet",
				"NAPTestScoreSummary", "NAPEventStudentLink":
				school := r.cfh.GetSchoolFromGuid(or.RefId())
				_, _ = files[school].Write([]byte(out1))
			case "NAPCodeFrame", "NAPTest",
				"NAPTestlet", "NAPTestItem":
				_, _ = testdata.Write([]byte(out1))
			}

			out <- or
		}
		for _, f := range files {
			_, _ = f.Write([]byte("</sif>\n"))
			f.Close()
		}
		_, _ = testdata.Write([]byte("</sif>\n"))
		testdata.Close()
		_, _ = schoollist.Write([]byte("</sif>\n"))
		schoollist.Close()

	}()
	return out
}

//
// generates a block of json that can be added to the
// record containing values that are not in the original data
//
//
func (r *XMLPerSchoolOutput) calculateFields(or *records.ObjectRecord) []byte {

	return or.CalculatedFields
}
