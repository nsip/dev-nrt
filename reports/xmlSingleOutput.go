package reports

import (
	"bytes"
	"strconv"
	"sync"

	"github.com/nsip/dev-nrt/records"
	"github.com/tidwall/gjson"
)

type XMLSingleOutput struct {
	baseReport // embed common setup capability
	wg         *sync.WaitGroup
}

//
// XML redaction
//
func XmlSingleOutputReport(wg *sync.WaitGroup) *XMLSingleOutput {

	r := XMLSingleOutput{wg: wg}
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
	r.wg.Add(1)

	go func() {
		defer close(out)

		//defer r.outF.Close()
		_, _ = r.outF.Write([]byte("<sif xmlns=\"http://www.sifassociation.org/datamodel/au/3.4\">\r\n"))

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
			out2, _ := strconv.Unquote(string(raw))
			out1 := bytes.Replace([]byte(out2), []byte("\n"), []byte("\r\n"), -1)
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
		_, _ = r.outF.Write([]byte("</sif>\r\n"))
		r.wg.Done()
		r.outF.Close()
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
