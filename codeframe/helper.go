//
// processing of data withitn the context of the codeframe, e.g.
// inserting user results into the overall structure of tests, testlets and items
// for a given domain is complex and repetitive.
//
// This package created to provide a single encapsulated helper that can be fed once
// with the codeframe data and then answer all codeframe related formatting and
// data extraction needs.
//
//
package codeframe

type Helper struct {
}

// temporary value, will be filled from real data
// so that lookups cannot fail due to local
// renaming of rubrics, also removes need to maintain such a list
// in an external config that may be overlooked for update.
//
// order is taken from the originating testItem writing rubrics list
//
var rubrics = []string{
	"Audience",
	"Text Structure",
	"Ideas",
	"Persuasive Devices",
	"Vocabulary",
	"Cohesion",
	"Paragraphing",
	"Sentence structure",
	"Punctuation",
	"Spelling",
}

//
// returns ordered list of writing rubrics
//
func (cfh *Helper) WritingRubricTypes() []string {

	return rubrics

}

//
// alias for writing rubrics, known as subscores in reslts
//
func (cfh *Helper) WritingSubscoreTypes() []string {

	return cfh.WritingRubricTypes()
}
