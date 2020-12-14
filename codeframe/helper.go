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

import (
	"github.com/nsip/dev-nrt/records"
	repo "github.com/nsip/dev-nrt/repository"
	"github.com/tidwall/gjson"
)

//
// Encapsulates data and helper methods to make
// working with codeframe objects easier
//
type Helper struct {
	data    map[string]map[string][]byte
	rubrics []string
}

//
// Builds a new codeframe helper using the data
// provided from the repository
//
func NewHelper(r *repo.BadgerRepo) (Helper, error) {

	h := Helper{data: make(map[string]map[string][]byte, 0)}
	opts := []records.Option{records.EmitterRepository(r)}
	em, err := records.NewEmitter(opts...)
	if err != nil {
		return h, err
	}

	for cfr := range em.CodeframeStream() {
		// watch out for null nodes in map
		if _, ok := h.data[cfr.RecordType]; !ok {
			h.data[cfr.RecordType] = make(map[string][]byte, 0)
		}
		h.data[cfr.RecordType][cfr.RefID()] = cfr.Json
	}

	h.extractRubrics()

	return h, nil

}

//
// internal function to create list of writing rubric types
// / subscores from actual test data
//
func (cfh *Helper) extractRubrics() {

	cfh.rubrics = []string{}

	for _, cfBytes := range cfh.data["NAPCodeFrame"] {

		// pick a stable writing test, using yr 7
		if gjson.GetBytes(cfBytes, "NAPCodeFrame.TestContent.TestLevel.Code").String() != "7" {
			continue
		}

		if gjson.GetBytes(cfBytes, "NAPCodeFrame.TestContent.Domain").String() != "Writing" {
			continue
		}

		//
		// iterate the nested json strucure & extract rubric types
		//
		gjson.GetBytes(cfBytes, "NAPCodeFrame.TestletList.Testlet").
			ForEach(func(key, value gjson.Result) bool {
				value.Get("TestItemList.TestItem").
					ForEach(func(key, value gjson.Result) bool {
						value.Get("TestItemContent.NAPWritingRubricList.NAPWritingRubric").
							ForEach(func(key, value gjson.Result) bool {
								value.Get("RubricType").ForEach(func(key, value gjson.Result) bool {
									// add to the internal lookup array
									cfh.rubrics = append(cfh.rubrics, value.String())
									return true // keep iterating
								})
								return true // keep iterating
							})
						return true // keep iterating
					})
				return true // keep iterating
			})
		break // only need one full set
	}
}

//
// returns ordered list of writing rubrics
//
func (cfh *Helper) WritingRubricTypes() []string {

	return cfh.rubrics

}

//
// alias for writing rubrics, known as subscores in reslts
//
func (cfh *Helper) WritingSubscoreTypes() []string {

	return cfh.WritingRubricTypes()
}
