package reports

import (
	"encoding/json"
	"encoding/xml"
	//"log"
	"strings"

	"github.com/nsip/dev-nrt/records"
	"github.com/nsip/sifxml2go/sifxml"
	"github.com/subchen/go-xmldom"
)

type XMLReport struct {
	baseReport // embed common setup capability
}

//
// XML redaction
//
func XmlRedactionReport() *XMLReport {

	r := XMLReport{}
	r.initialise("./config/XMLRedaction.toml")
	r.printStatus()

	return &r

}

//
// implement the ...Pipe interface, core work of the
// report engine.
//
func (r *XMLReport) ProcessObjectRecords(in chan *records.ObjectRecord) chan *records.ObjectRecord {

	out := make(chan *records.ObjectRecord)
	go func() {

		defer r.outF.Close()
		_, _ = r.outF.Write([]byte("<sif xmlns=\"http://www.sifassociation.org/datamodel/au/3.4\">\n"))

		// In this report, the queries are fields to redact, not individual CSV columns
		filters, ok := r.config.tree.Get("options.filters").([]string)
		if !ok {
			filters = []string{}
		}

		for or := range in {
			if !r.config.activated { // only process if activated
				out <- or
				continue
			}
			var xml_out []byte
			var err error

			//
			// generate any calculated fields required
			//
			or.CalculatedFields = r.calculateFields(or)

			switch or.RecordType {
			case "SchoolInfo":
				a := sifxml.SchoolInfo{}
				err = json.Unmarshal([]byte(or.Json), &a)
				xml_out, err = xml.MarshalIndent(a, "", "  ")
			case "StudentPersonal":
				a := sifxml.StudentPersonal{}
				err = json.Unmarshal([]byte(or.Json), &a)
				xml_out, err = xml.MarshalIndent(a, "", "  ")
			case "NAPEventStudentLink":
				a := sifxml.NAPEventStudentLink{}
				err = json.Unmarshal([]byte(or.Json), &a)
				xml_out, err = xml.MarshalIndent(a, "", "  ")
			case "NAPTest":
				a := sifxml.NAPTest{}
				err = json.Unmarshal([]byte(or.Json), &a)
				xml_out, err = xml.MarshalIndent(a, "", "  ")
			case "NAPTestlet":
				a := sifxml.NAPTestlet{}
				err = json.Unmarshal([]byte(or.Json), &a)
				xml_out, err = xml.MarshalIndent(a, "", "  ")
			case "NAPTestItem":
				a := sifxml.NAPTestItem{}
				err = json.Unmarshal([]byte(or.Json), &a)
				xml_out, err = xml.MarshalIndent(a, "", "  ")
			case "NAPStudentResponseSet":
				a := sifxml.NAPStudentResponseSet{}
				err = json.Unmarshal([]byte(or.Json), &a)
				xml_out, err = xml.MarshalIndent(a, "", "  ")
			case "NAPCodeFrame":
				a := sifxml.NAPCodeFrame{}
				err = json.Unmarshal([]byte(or.Json), &a)
				xml_out, err = xml.MarshalIndent(a, "", "  ")
			case "NAPTestScoreSummary":
				a := sifxml.NAPTestScoreSummary{}
				err = json.Unmarshal([]byte(or.Json), &a)
				xml_out, err = xml.MarshalIndent(a, "", "  ")
			}

			if err != nil {
			}
			//log.Printf("%s", string(or.Json))
			//log.Printf("%s", string(xml_out))

			//
			// now loop through the output definitions to create XML output
			//
			doc := xmldom.Must(xmldom.ParseXML("<sif>" + string(xml_out) + "</sif>"))
			for _, path := range filters {
				nodelist := doc.Root.Query(path)
				for _, c := range nodelist {
					c.SetAttributeValue("xsi:nil", "true")
					c.Text = ""
				}
			}
			out0 := doc.Root.FirstChild().XMLPretty()
			out0 = strings.Replace(out0, "&#xA;", "\n", -1)
			out0 = strings.Replace(out0, "&#xD;", "\r", -1)
			out0 = strings.Replace(out0, "&#x9;", "\t", -1)
			out0 = strings.Replace(out0, "&#9;", "\t", -1)
			out0 = strings.Replace(out0, "&#34;", "\"", -1)
			out0 = strings.Replace(out0, "&#x27;", "'", -1)
			out0 = strings.Replace(out0, "&#39;", "'", -1)
			out1 := []byte(out0)
			_, err = r.outF.Write(out1)

			out <- or
		}
		_, _ = r.outF.Write([]byte("</sif>\n"))
	}()
	return out
}

//
// generates a block of json that can be added to the
// record containing values that are not in the original data
//
//
func (r *XMLReport) calculateFields(or *records.ObjectRecord) []byte {

	return or.CalculatedFields
}
