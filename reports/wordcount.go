package reports

import (
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/clipperhouse/jargon"
	"github.com/clipperhouse/jargon/filters/contractions"
)

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
