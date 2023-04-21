package reports

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log"

	"strings"

	"github.com/nsip/dev-nrt/records"
	"github.com/nsip/sifxml2go/sifxml"
	"github.com/subchen/go-xmldom"
	"github.com/tidwall/sjson"
)

type XMLReport struct {
	baseReport // embed common setup capability
	filters    []string
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

	// In this report, the queries are fields to redact, not individual CSV columns
	r.filters = []string{}
	filters1, ok := r.config.tree.Get("options.filters").([]interface{})
	if filters1 != nil && ok {
		r.filters = make([]string, 0)
		for _, v := range filters1 {
			x := strings.TrimSpace(fmt.Sprint(v))
			if !strings.HasPrefix(x, "#") && x != "" {
				r.filters = append(r.filters, x)
			}
		}
	}

	out := make(chan *records.ObjectRecord)

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

			out <- or
		}
		close(out)
	}()
	return out
}

//
// generates a block of json that can be added to the
// record containing values that are not in the original data
//
//
func (r *XMLReport) calculateFields(or *records.ObjectRecord) []byte {
	var err error
	var xml_out []byte

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
	var out0 string
	var b strings.Builder
	if len(r.filters) == 0 {
		out0 = string(xml_out)
	} else {
		b.Reset()
		b.WriteString("<sif>")
		b.WriteString(string(xml_out))
		b.WriteString("</sif>")
		doc, err := xmldom.ParseXML(b.String())
		if err != nil {
			log.Println(string(or.Json))
			log.Println(string(xml_out))
			log.Println(b.String())
			log.Fatal(err)
		}
		// XPath implementation in golang not coping with roots, we will do // instead
		for _, path := range r.filters {
			nodelist := doc.Root.Query(path)
			if len(nodelist) > 0 {
				//log.Printf("%d nodes match '%s' for redaction\n", len(nodelist), path, i, len(r.filters))
			}
			for _, c := range nodelist {
				c.SetAttributeValue("xsi:nil", "true")
				c.Text = ""
			}
		}
		out0 = doc.Root.FirstChild().XMLPretty()
	}
	out0 = strings.Replace(out0, "&#xA;", "\n", -1)
	out0 = strings.Replace(out0, "&#xD;", "\r", -1)
	out0 = strings.Replace(out0, "&#x9;", "\t", -1)
	out0 = strings.Replace(out0, "&#9;", "\t", -1)
	out0 = strings.Replace(out0, "&#34;", "\"", -1)
	out0 = strings.Replace(out0, "&#x27;", "'", -1)
	out0 = strings.Replace(out0, "&#39;", "'", -1)
	out1 := []byte(out0)

	json := or.CalculatedFields
	json, _ = sjson.SetBytes(json, "CalculatedFields.Redacted", out1)

	return json
}
