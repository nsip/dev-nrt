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
	"fmt"
	"time"

	"github.com/nsip/dev-nrt/pipelines"
	"github.com/nsip/dev-nrt/utils"
	"github.com/nsip/dev-nrt/records"
	"github.com/nsip/dev-nrt/repository"
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

func NewHelper(r *repository.BadgerRepo) (Helper, error) {

	defer utils.TimeTrack(time.Now(),"codeframe NewHelper()")

	h := Helper{data: make(map[string]map[string][]byte, 0)} // initialise the internal map

	// 
	// extract codeframe data from repository
	// 

	// wrap repo in suitable emitter
	opts := []records.Option{records.EmitterRepository(r)}
	em, err := records.NewEmitter(opts...)
	if err != nil {
		return h,err
	}
	
	// create a simple one-element pipeline
	cfpl := pipelines.NewCodeframePipeline(h)
	defer cfpl.Close()
	// spawn a no-op reader to consume pipeline output
	go cfpl.Dequeue(func(cfr *records.CodeframeRecord){cfr = nil})
	// iterate the codeframe dataset, will be handled by Process... method 
	for cfr := range em.CodeframeStream() {
		cfpl.Enqueue(cfr)
	}

	// 
	// do any further pre-processing after all data received
	// 

	// 
	// access writing rubrics directly
	// 
	h.extractRubrics()

	// 
	// extract item sequence ordering from codeframe
	// 

	return h, nil

}

//
// implement the codeframe pipe interface, so this can be attached to a
// codeframe emitter.
//
func (cfh Helper) ProcessCodeframeRecords(in chan *records.CodeframeRecord) chan *records.CodeframeRecord {
	out := make(chan *records.CodeframeRecord)
	go func() {
		defer close(out)

		// collect all codeframe data
		for cfr := range in {
			// watch out for null nodes in map
			if _, ok := cfh.data[cfr.RecordType]; !ok {
				cfh.data[cfr.RecordType] = make(map[string][]byte, 0)
			}
			cfh.data[cfr.RecordType][cfr.RefID()] = cfr.Json

			out <- cfr
		}

	}()
	return out

}

//
// internal function to create list of writing rubric types
// / subscores from actual test data
//
func (cfh Helper) extractRubrics() {

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
func (cfh Helper) WritingRubricTypes() []string {

	return cfh.rubrics

}

//
// alias for writing rubrics, known as subscores in reslts
//
func (cfh Helper) WritingSubscoreTypes() []string {

	return cfh.WritingRubricTypes()
}

//
// return the json block for a given testitem
//
func (cfh Helper) GetItem(refid string) []byte {

	item, ok := cfh.data["NAPTestItem"][refid]
	if !ok {
		fmt.Println("cfh unable to find TestItem", refid)
		return []byte{}
	}
	return item
}
