package nrt

import (
	"encoding/csv"
	"fmt"
	"html"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/clipperhouse/jargon"
	"github.com/clipperhouse/jargon/filters/contractions"
	"github.com/nats-io/nuid"
	"github.com/nsip/dev-nrt/records"
	repo "github.com/nsip/dev-nrt/repository"
)

func StreamResults() error {

	timeTrack(time.Now(), "StreamResults")

	// get the results data
	r, err := repo.OpenExistingBadgerRepo("./kv/")
	if err != nil {
		return err
	}
	defer r.Close()

	// create the emitter
	opts := []records.Option{records.EmitterRepository(r)}
	em, err := records.NewEmitter(opts...)

	// open a csv file
	file, err := os.Create("./out/writing_extract.csv")
	if err != nil {
		return err
	}
	w := csv.NewWriter(file)
	// write the header
	header := []string{"Test Year",
		"Test level",
		"Jurisdiction Id",
		"ACARA ID",
		"PSI",
		"Local school student ID",
		"TAA student ID",
		"Participation Code",
		"Item Response",
		"Anonymised Id",
		"Test Id",
		"Word Count",
		"Date",
		"StartTime"}

	w.Write(header)

	count := 0

	var tyr, tlvl, jid, aid, psi, lid, taaid, pc, ir, anonid, tid, wc, dt, st string
	for eor := range em.EventBasedStream() {
		//
		// only process writing tests
		//
		if !eor.IsWritingResponse() {
			continue
		}

		//
		// if no response, but marked as particpated is likely a
		// failed session so ignore for reporting
		//
		if eor.Err == records.ErrMissingResponse && eor.ParticipatedInTest() {
			continue
		}

		var csvrow = make([]string, 0)

		// extract report data fields
		tyr = records.GetValueString(eor.NAPTest, "NAPTest.TestContent.TestYear")
		tlvl = records.GetValueString(eor.NAPTest, "NAPTest.TestContent.TestLevel.Code")
		jid = records.GetValueString(eor.StudentPersonal, "StudentPersonal.OtherIdList.OtherId.#[Type==JurisdictionId].value")
		aid = records.GetValueString(eor.SchoolInfo, "SchoolInfo.ACARAId")
		psi = records.GetValueString(eor.StudentPersonal, "StudentPersonal.OtherIdList.OtherId.#[Type==NAPPlatformStudentId].value")
		lid = records.GetValueString(eor.StudentPersonal, "StudentPersonal.LocalId")
		taaid = records.GetValueString(eor.StudentPersonal, "StudentPersonal.OtherIdList.OtherId.#[Type==TAAStudentId].value")
		pc = records.GetValueString(eor.NAPEventStudentLink, "NAPEventStudentLink.ParticipationCode")
		ir = records.GetValueString(eor.NAPStudentResponseSet, "NAPStudentResponseSet.TestletList.Testlet.0.ItemResponseList.ItemResponse.0.Response")
		ir = html.UnescapeString(ir) // html of writing response needs unescaping
		tid = records.GetValueString(eor.NAPTest, "NAPTest.TestContent.NAPTestLocalId")
		dt = records.GetValueString(eor.NAPEventStudentLink, "NAPEventStudentLink.Date")
		st = records.GetValueString(eor.NAPEventStudentLink, "NAPEventStudentLink.StartTime")
		// calculated fields
		anonid = nuid.Next()
		wc = strconv.Itoa(countwords(ir))

		csvrow = append(csvrow, tyr, tlvl, jid, aid, psi, lid, taaid, pc, ir, anonid, tid, wc, dt, st)
		if err := w.Write(csvrow); err != nil {
			log.Fatalln("error writing record to csv:", err)
		}
		count++
	}

	w.Flush()

	fmt.Println("Events retrieved:", count)

	return nil

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
