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
	"log"
	"strconv"
	"strings"
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
	data            map[string]map[string][]byte
	reverseLookup   map[string]map[string]string
	itemSequence    map[string]map[string]string
	locationInStage map[string]string
	rubrics         []string
	substitutes     map[string]string
}

//
// Creates a new codeframe helper instance.
// r - a repository containing the rrd data
//
func NewHelper(r *repository.BadgerRepo) (Helper, error) {

	defer utils.TimeTrack(time.Now(), "codeframe NewHelper()")

	h := Helper{
		data:            make(map[string]map[string][]byte, 0),
		reverseLookup:   make(map[string]map[string]string, 0),
		itemSequence:    make(map[string]map[string]string, 0),
		locationInStage: make(map[string]string, 0),
		substitutes:     make(map[string]string, 0),
		rubrics:         []string{},
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
	// create reverse lookup items -> tests/testlets
	//
	h.buildReverseLookup()
	//
	// build lookup for sequencing info of testlets in tests
	//
	h.extractLocationInStage()
	//
	// extract substitute items
	//
	h.extractSubstitutes()
	//
	// extract item sequencing within testlets
	//
	h.extractItemSequence()

	return h, nil

}

//
// creates a lookup to return the sequence locaiton of an
// item within a testlet
// NOTE: index needs baselining from 1 to align with testlet
// definitions - codeframe baselines from 0
//
func (cfh *Helper) extractItemSequence() {

	var testletRefId, itemRefId, sequenceNumber string
	for _, cfBytes := range cfh.data["NAPCodeFrame"] {
		//
		// iterate the nested json strucure & extract containers
		//
		gjson.GetBytes(cfBytes, "NAPCodeFrame.TestletList.Testlet").
			ForEach(func(key, value gjson.Result) bool {
				testletRefId = value.Get("NAPTestletRefId").String() // get the testlet refid
				value.Get("TestItemList.TestItem").
					ForEach(func(key, value gjson.Result) bool {
						itemRefId = value.Get("TestItemRefId").String() // get the item refid
						sn := value.Get("SequenceNumber").Int() + 1     // re-baseline
						sequenceNumber = strconv.Itoa(int(sn))          // convert to string
						if _, ok := cfh.itemSequence[itemRefId]; !ok {  // avoid null nodes
							cfh.itemSequence[itemRefId] = make(map[string]string, 0)
						}
						cfh.itemSequence[itemRefId][testletRefId] = sequenceNumber // store the lookup
						return true                                                // keep iterating
					})
				return true // keep iterating
			})
	}

}

//
// creates a lookup to resolve substitute items against
// their alternates
//
func (cfh *Helper) extractSubstitutes() {
	var itemRefId, substituteRefId string
	for _, cfBytes := range cfh.data["NAPCodeFrame"] {
		//
		// iterate the nested json strucure & extract substitutes
		//
		gjson.GetBytes(cfBytes, "NAPCodeFrame.TestletList.Testlet").
			ForEach(func(key, value gjson.Result) bool {
				value.Get("TestItemList.TestItem").
					ForEach(func(key, value gjson.Result) bool {
						itemRefId = value.Get("TestItemRefId").String() // get the item refid
						// see if this item is a substitute
						value.Get("TestItemContent.ItemSubstitutedForList.SubstituteItem").
							ForEach(func(key, value gjson.Result) bool {
								// if so get the refid of the item it subs for
								substituteRefId = value.Get("SubstituteItemRefId").String()
								cfh.substitutes[itemRefId] = substituteRefId
								return true // keep iterating
							})
						return true // keep iterating
					})
				return true // keep iterating
			})
	}
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
// telstlet location in stage only available in
// codeframe, so create lookup - testletid -> location
//
func (cfh *Helper) extractLocationInStage() {

	var testletRefId, lis string
	for _, cfBytes := range cfh.data["NAPCodeFrame"] {
		//
		// iterate the nested json strucure & extract locations
		//
		gjson.GetBytes(cfBytes, "NAPCodeFrame.TestletList.Testlet").
			ForEach(func(key, value gjson.Result) bool {
				testletRefId = value.Get("NAPTestletRefId").String()       // get the testlet refid
				lis = value.Get("TestletContent.LocationInStage").String() // location in stage
				cfh.locationInStage[testletRefId] = lis                    // store in lookup
				return true                                                // keep iterating
			})
	}

}

//
// we need to be able to reverse lookup the codeframe structure
// e.g. Test from Item - find the test/s an item was assinged to
// via testlets
//
func (cfh *Helper) buildReverseLookup() {

	var testRefId, testletRefId, itemRefId string
	for _, cfBytes := range cfh.data["NAPCodeFrame"] {
		//
		// get the test id
		//
		testRefId = gjson.GetBytes(cfBytes, "NAPCodeFrame.NAPTestRefId").String()
		//
		// iterate the nested json strucure & extract containers
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
}

//
// internal function to create list of writing rubric types
// / subscores from actual test data
//
func (cfh *Helper) extractRubrics() {

	rubrics := []string{}

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
									rubrics = append(rubrics, value.String())
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

	cfh.rubrics = rubrics
}

//
// returns ordered list of writing rubrics
//
func (cfh Helper) WritingRubricTypes() []string {

	return cfh.WritingSubscoreTypes()

}

//
// alias for writing rubrics, known as subscores in results
//
func (cfh Helper) WritingSubscoreTypes() []string {

	return cfh.rubrics
}

//
// get the list of Disability Adjustment Codes supported for this naplan cycle
//
func (cfh Helper) GetDACs() []string {
	return []string{
		// school-level
		"AIA", //"Alternative items - audio",
		"AIV", //"Alternative items - visual",
		"AST", //"Assistive technology",
		"BNB", //"Colour contrast Black with Blue background",
		"BNG", //"Colour contrast Black with Green background",
		"BNL", //"Colour contrast Black with Lilac background",
		"BNW", //"Colour contrast Black with White background",
		"BNY", //"Colour contrast Black with Yellow background",
		"COL", //"Colour contrast modification",
		"ETA", //"Extra Time – one minute for every six minutes of test time",
		"ETB", //"Extra Time – one minute for every three minutes of test time",
		"ETC", //"Extra Time – one minute for every two minutes of test time",
		"ETD", //"Extra Time – double total test time",
		"OFF", //"Braille, large print, black and white, electronic test format",
		"OSS", //"Oral sign/support",
		"RBK", //"Rest break",
		"SCR", //"Scribe",
		"SUP", //"NAPLAN Support person",
		// system-admin level
		"CAL",   //  "Calculator Fit to Screen",
		"ENZ",   //  "Enable Zoom",
		"EST",   //  "Editor Sticky Toolbar",
		"LFS",   //  "Larger Font Sizes",
		"RZL",   //  "Remember Zoom Level",
		"ZOF",   //  "Zoomed Optimised Features",
		"ZTFAO", //"Zoom to Always On",
	}
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
// return the refids of the test/testlet combinations
// that use a particular item
// refid - the refid of a test item
//
// returns a map of pairs where key is the testlet refid and value is the test refid
//
func (cfh Helper) GetContainersForItem(refid string) map[string]string {

	c := make(map[string]string, 0)
	c = cfh.reverseLookup[refid]

	return c

}

//
// find location in stage for testlet
// comes from codeframe not testlet object
//
func (cfh Helper) GetTestletLocationInStage(testletrefid string) string {

	lis, ok := cfh.locationInStage[testletrefid]
	if !ok {
		return ""
	}

	return lis
}

//
// for a given item refid, returns the refid (string) of the item
// this item substitutes for
// boolean return can be used simply to determine if this item is
// a substitute
//
func (cfh Helper) IsSubstituteItem(refid string) (string, bool) {

	subForRefId, ok := cfh.substitutes[refid]
	if !ok {
		return "", ok
	}

	return subForRefId, ok
}

//
// pass a json path to retrieve the value at that location as a
// string
// refid - the identifier of the object to be queried
// queryPath the gjson query string to apply to the object
//
func (cfh Helper) GetCodeframeObjectValueString(refid, queryPath string) string {

	//
	// get the root object
	//
	objName := strings.Split(queryPath, ".")[0]
	record, ok := cfh.data[objName][refid]
	if !ok {
		log.Println("GetCodeframeObjectValueString() cannot find value for path:", queryPath)
		return ""
	}

	//
	// return the result of the json query
	//
	return gjson.GetBytes(record, queryPath).String()
}

//
// returns the sequnce number for a test item within a testlet
//
func (cfh Helper) GetItemTestletSequenceNumber(itemrefid, testletrefid string) string {

	sqnum, ok := cfh.itemSequence[itemrefid][testletrefid]
	if !ok {
		log.Println("No sequence number found for item:testlet pair:", itemrefid, testletrefid)
		return ""
	}

	return sqnum

}
