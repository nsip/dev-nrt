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
	"github.com/nsip/dev-nrt/records"
	"github.com/nsip/dev-nrt/repository"
	"github.com/nsip/dev-nrt/utils"
	"github.com/tidwall/gjson"
)

//
// Encapsulates data and helper methods to make
// working with codeframe objects easier
//
type Helper struct {
	data          map[string]map[string][]byte
	reverseLookup map[string]map[string]string
	rubrics       []string
}

//
// Creates a new codeframe helper instance.
// r - a repository containing the rrd data
//
func NewHelper(r *repository.BadgerRepo) (Helper, error) {

	defer utils.TimeTrack(time.Now(), "codeframe NewHelper()")

	h := Helper{
		data:          make(map[string]map[string][]byte, 0),
		reverseLookup: make(map[string]map[string]string, 0),
	} // initialise the internal maps

	// wrap repo in emitter
	opts := []records.Option{records.EmitterRepository(r)}
	em, err := records.NewEmitter(opts...)
	if err != nil {
		return h, err
	}

	// create a simple one-element pipeline, with this helper as the only processor
	cfpl := pipelines.NewCodeframePipeline(h)
	defer cfpl.Close()
	// spawn a no-op reader to consume pipeline output
	go cfpl.Dequeue(func(cfr *records.CodeframeRecord) { cfr = nil })
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
	h.buildLookup()

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
			cfh.data[cfr.RecordType][cfr.RefId()] = cfr.Json

			out <- cfr
		}

	}()
	return out

}

//
// we need to be able to reverse lookup the codeframe structure
// e.g. Test from Item - find the test/s an item was assinged to
// via testlets
//
func (cfh Helper) buildLookup() {

	var testRefId, testletRefId, itemRefId string
	for _, cfBytes := range cfh.data["NAPCodeFrame"] {
		//
		// get the test id
		//
		testRefId = gjson.GetBytes(cfBytes, "NAPCodeFrame.NAPTestRefId").String()
		//
		// iterate the nested json strucure & extract rubric types
		//
		gjson.GetBytes(cfBytes, "NAPCodeFrame.TestletList.Testlet").
			ForEach(func(key, value gjson.Result) bool {
				testletRefId = value.Get("NAPTestletRefId").String() // get the testlet refid
				value.Get("TestItemList.TestItem").
					ForEach(func(key, value gjson.Result) bool {
						itemRefId = value.Get("TestItemRefId").String() // get the item refid
						if _, ok := cfh.reverseLookup[itemRefId]; !ok { // avoid null nodes
							cfh.reverseLookup[itemRefId] = make(map[string]string, 0)
						}
						cfh.reverseLookup[itemRefId][testletRefId] = testRefId // store the lookup
						return true                                            // keep iterating
					})
				return true // keep iterating
			})
	}

	// //
	// // check by printing
	// //
	// for key, val := range cfh.reverseLookup {
	// 	fmt.Println("Item:", key)
	// 	for tl, tst := range val {
	// 		fmt.Println("\tTestlet:", tl)
	// 		fmt.Println("\t\tTest:", tst)
	// 	}
	// }

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
// return the json block for a given testitem refid
// boolean return value indicates if a value was found
//
func (cfh Helper) GetItem(refid string) (bool, []byte) {

	item, ok := cfh.data["NAPTestItem"][refid]
	if !ok {
		fmt.Println("cfh unable to find TestItem", refid)
		return false, []byte{}
	}
	return true, item
}

//
// return the refids of any tests that use a particular item
// refid - the refid of a test item
//
func (cfh Helper) GetTestsForItem(refid string) []string {

	testids := []string{}
	for _, v := range cfh.reverseLookup[refid] {
		testids = append(testids, v)
	}

	return testids

}
