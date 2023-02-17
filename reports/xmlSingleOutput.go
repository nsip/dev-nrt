package reports

import (
	"strconv"

	"github.com/nsip/dev-nrt/records"
	"github.com/tidwall/gjson"
)

type XMLSingleOutput struct {
	baseReport // embed common setup capability
}

//
// XML redaction
//
func XmlSingleOutputReport() *XMLSingleOutput {

	r := XMLSingleOutput{}
	r.initialise("./config/XMLSingleOutput.toml")
	r.printStatus()

	return &r

}

//
// implement the ...Pipe interface, core work of the
// report engine.
//
func (r *XMLSingleOutput) ProcessObjectRecords(in chan *records.ObjectRecord) chan *records.ObjectRecord {

	out := make(chan *records.ObjectRecord)
	go func() {

		defer r.outF.Close()
		_, _ = r.outF.Write([]byte("<sif xmlns=\"http://www.sifassociation.org/datamodel/au/3.4\">\n"))

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
			/*
				raw = bytes.Replace(raw, []byte("\\u003c"), []byte("<"), -1)
				raw = bytes.Replace(raw, []byte("\\u003e"), []byte(">"), -1)
				raw = bytes.Replace(raw, []byte("\\n"), []byte("\n"), -1)
				raw = bytes.Replace(raw, []byte("\\r"), []byte("\r"), -1)
				raw = bytes.Replace(raw, []byte("\\\""), []byte("\""), -1)
			*/

			r.outF.Write([]byte(out1))

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
func (r *XMLSingleOutput) calculateFields(or *records.ObjectRecord) []byte {

	return or.CalculatedFields
}
