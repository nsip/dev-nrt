package reports

import (
	"encoding/csv"
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/clipperhouse/jargon"
	"github.com/clipperhouse/jargon/filters/contractions"
	"github.com/nats-io/nuid"
	"github.com/nsip/dev-nrt/records"
	"github.com/tidwall/sjson"
)

type WritingExtract struct {
	baseReport // embed common setup capability
}

//
// creates file used to feed student writing responses into
// local marking systems
//
func WritingExtractReport() *WritingExtract {

	we := WritingExtract{}
	we.initialise("./config/WritingExtract.toml")
	we.printStatus()

	return &we

}

//
// implement the EventPipe interface, core work of the
// report engine.
//
func (we *WritingExtract) ProcessEventRecords(in chan *records.EventOrientedRecord) chan *records.EventOrientedRecord {

	out := make(chan *records.EventOrientedRecord)
	go func() {
		defer close(out)
		// open the csv file writer, and set the header
		w := csv.NewWriter(we.outF)
		w.Write(we.config.header)
		defer w.Flush()

		for eor := range in {
			if we.config.activated { // only process if active
				if eor.IsWritingResponse() { // only process writing responses
					//
					// generate any calculated fields required
					//
					eor.CalculatedFields = we.calculateFields(eor)
					//
					// now loop through the ouput definitions to create a
					// row of results
					//
					var result string
					var row []string = make([]string, 0, len(we.config.queries))
					for _, query := range we.config.queries {
						result = eor.GetValueString(query)
						row = append(row, result)
					}
					// write the row to the output file
					if err := w.Write(row); err != nil {
						fmt.Println("Warning: error writing record to csv:", we.config.name, err)
					}
					//
					// in this case no need to preserve the calculated
					// fields for any other purpose
					//
					eor.CalculatedFields = []byte{}
				}
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
func (we *WritingExtract) calculateFields(eor *records.EventOrientedRecord) []byte {

	anonid := nuid.Next()
	ir := eor.GetValueString("NAPStudentResponseSet.TestletList.Testlet.0.ItemResponseList.ItemResponse.0.Response")
	ir = html.UnescapeString(ir) // html of writing response needs unescaping
	wc := strconv.Itoa(countwords(ir))

	json := []byte{}
	json, _ = sjson.SetBytes(json, "CalculatedFields.AnonId", anonid)
	json, _ = sjson.SetBytes(json, "CalculatedFields.WordCount", wc)
	json, _ = sjson.SetBytes(json, "CalculatedFields.EscapedResponse", ir)

	// fmt.Println("CF:json:", string(json))
	return json
}

//
// Straight copy-in from n2 for compatability of results
//
//var lem = jargon.NewLemmatizer(contractions.Dictionary, 3)
var tokenRe = regexp.MustCompile("[a-zA-Z0-9]")
var hyphens = regexp.MustCompile("-+")

func countwords(html string) int {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return 0
	}
	doc.Find("script").Each(func(i int, el *goquery.Selection) {
		el.Remove()
	})
	// Jargon lemmatiser tokenises text, and resolves contractions
	// We will resolve hyphenated compounds ourselves
	tokens := jargon.Tokenize(strings.NewReader(doc.Text()))
	//lemmas := lem.Lemmatize(tokens)
	lemmas := contractions.Expand(tokens)
	wc := 0
	for {
		lemma, err := lemmas.Next()
		if lemma == nil || err != nil {
			break
		}
		wordpart := hyphens.Split(lemma.String(), -1)
		for _, w := range wordpart {
			if len(tokenRe.FindString(w)) > 0 {
				wc = wc + 1
			}
		}
	}
	return wc
}
