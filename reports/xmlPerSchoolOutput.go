package reports

import (
	"bytes"
	"log"
	"os"
	"path"
	"strconv"
	"sync"

	"github.com/nsip/dev-nrt/helper"
	"github.com/nsip/dev-nrt/records"
	"github.com/tidwall/gjson"
)

type XMLPerSchoolOutput struct {
	baseReport // embed common setup capability
	cfh        helper.ObjectHelper
	wg         *sync.WaitGroup
}

//
// XML redaction, emulating NAPLAN API output
//
func XmlPerSchoolOutputReport(cfh helper.ObjectHelper, wg *sync.WaitGroup) *XMLPerSchoolOutput {

	r := XMLPerSchoolOutput{cfh: cfh, wg: wg}
	r.initialise("./config/XMLPerSchoolOutput.toml")
	r.printStatus()

	return &r
}

//
// implement the ...Pipe interface, core work of the
// report engine.
//
func (r *XMLPerSchoolOutput) ProcessObjectRecords(in chan *records.ObjectRecord) chan *records.ObjectRecord {
	var err error
	var f *os.File
	out := make(chan *records.ObjectRecord)
	schools := r.cfh.GetSchoolRefIds()
	r.outF.Close()
	os.Remove(r.config.outputFileName)
	files := make(map[string]string, 0)
	r.wg.Add(1)

	for _, refid := range schools {
		filename := path.Dir(r.config.outputFileName) + "/schooldata_" + refid + ".xml"
		acaraid := r.cfh.GetSchoolFromGuid(refid)
		files[acaraid] = filename
		if f, err = os.Create(files[acaraid]); err != nil {
			log.Fatal(err)
		}
		if _, err = f.Write([]byte("<sif xmlns=\"http://www.sifassociation.org/datamodel/au/3.4\">\r\n")); err != nil {
			log.Fatal(err)
		}
		f.Close()
	}
	filename := path.Dir(r.config.outputFileName) + "/testdata.xml"
	testdata, err := os.Create(filename)
	if err != nil {
		log.Fatal(err)
	}
	//defer testdata.Close()
	if _, err = testdata.Write([]byte("<sif xmlns=\"http://www.sifassociation.org/datamodel/au/3.4\">\r\n")); err != nil {
		log.Fatal(err)
	}
	filename = path.Dir(r.config.outputFileName) + "/schoollist.xml"
	schoollist, err := os.Create(filename)
	if err != nil {
		log.Fatal(err)
	}
	//defer schoollist.Close()
	if _, err = schoollist.Write([]byte("<sif xmlns=\"http://www.sifassociation.org/datamodel/au/3.4\">\r\n")); err != nil {
		log.Fatal(err)
	}

	go func() {
		var err error
		defer close(out)
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
			out2, err := strconv.Unquote(string(raw))
			if err != nil {
				log.Fatal(err)
			}
			out1 := bytes.Replace([]byte(out2), []byte("\n"), []byte("\r\n"), -1)

			switch or.RecordType {
			case "SchoolInfo":
				if _, err = schoollist.Write(out1); err != nil {
					log.Fatal(err)
				}
			case "StudentPersonal", "NAPStudentResponseSet",
				"NAPTestScoreSummary", "NAPEventStudentLink":
				school := r.cfh.GetSchoolFromGuid(or.RefId())
				f, err := os.OpenFile(files[school], os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
				if err != nil {
					log.Fatal(err)
				}
				if _, err = f.Write(out1); err != nil {
					log.Fatal(err)
				}
				f.Close()
			case "NAPCodeFrame", "NAPTest",
				"NAPTestlet", "NAPTestItem":
				if _, err = testdata.Write(out1); err != nil {
					log.Fatal(err)
				}
			}

			out <- or
		}
		for _, n := range files {
			f, err := os.OpenFile(n, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				log.Fatal(err)
			}
			if _, err = f.Write([]byte("</sif>\r\n")); err != nil {
				log.Fatal(err)
			}
			f.Close()
		}
		if _, err = testdata.Write([]byte("</sif>\r\n")); err != nil {
			log.Fatal(err)
		}
		testdata.Close()
		if _, err = schoollist.Write([]byte("</sif>\r\n")); err != nil {
			log.Fatal(err)
		}
		schoollist.Close()
		r.wg.Done()

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
