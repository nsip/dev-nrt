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
package helper

import (
	"strconv"
	"strings"
	"sync"

	"github.com/nsip/dev-nrt/pipelines"
	"github.com/nsip/dev-nrt/records"
	"github.com/nsip/dev-nrt/repository"
	"github.com/tidwall/gjson"
)

//
// Encapsulates data and helper methods to make
// working with codeframe objects easier
//
type CodeframeHelper struct {
	data               map[string]map[string][]byte
	reverseLookup      map[string]map[string][]string
	itemSequence       map[string]map[string]string
	locationInStage    map[string]string
	rubrics            []string
	substitutes        map[string]map[string]struct{}
	reverseSubstitutes map[string]map[string]struct{}
	expectedItems      map[string]map[string]map[string]struct{}
	item2test          map[string]map[string]struct{}
	testchars          map[string]map[string]string
}

//
// Creates a new codeframe helper instance.
// r - a repository containing the rrd data
//
func NewCodeframeHelper(r *repository.BadgerRepo, wg *sync.WaitGroup) (CodeframeHelper, error) {

	// defer utils.TimeTrack(time.Now(), "codeframe NewCodeframeHelper()")

	h := CodeframeHelper{
		data:               make(map[string]map[string][]byte, 0),
		reverseLookup:      make(map[string]map[string][]string, 0),
		itemSequence:       make(map[string]map[string]string, 0),
		locationInStage:    make(map[string]string, 0),
		substitutes:        make(map[string]map[string]struct{}, 0),
		reverseSubstitutes: make(map[string]map[string]struct{}, 0),
		rubrics:            []string{},
		expectedItems:      make(map[string]map[string]map[string]struct{}, 0),
		item2test:          make(map[string]map[string]struct{}, 0),
		testchars:          make(map[string]map[string]string, 0),
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
	//
	// extract expected items per testlet
	//
	h.extractExpectedTestletItems()

	wg.Done()
	return h, nil

}

//
// creates a lookup to return the sequence locaiton of an
// item within a testlet
// NOTE: index needs baselining from 1 to align with testlet
// definitions - codeframe baselines from 0
//
func (cfh *CodeframeHelper) extractItemSequence() {

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
func (cfh *CodeframeHelper) extractSubstitutes() {
	var itemRefId, substituteRefId string
	//
	// codeframe can contain items with substitutes, making them reverse substitutions
	// probably not deliberate, but capture anyway
	//
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
								if _, ok := cfh.substitutes[itemRefId]; !ok { // watch for null values
									cfh.substitutes[itemRefId] = make(map[string]struct{}, 0)
								}
								// if so get the refid of the items it can sub for
								substituteRefId = value.Get("SubstituteItemRefId").String()
								cfh.substitutes[itemRefId][substituteRefId] = struct{}{}
								if _, ok := cfh.reverseSubstitutes[substituteRefId]; !ok { // watch for null values
									cfh.reverseSubstitutes[substituteRefId] = make(map[string]struct{}, 0)
								}
								cfh.reverseSubstitutes[substituteRefId][itemRefId] = struct{}{}
								return true // keep iterating
							})
						return true // keep iterating
					})
				return true // keep iterating
			})
	}

	//
	// main capture of substitutes is from the atomic items themselves
	//
	for _, itemBytes := range cfh.data["NAPTestItem"] {
		//
		// iterate the nested json strucure & extract substitutes
		//
		itemRefId = gjson.GetBytes(itemBytes, "NAPTestItem.RefId").String()
		gjson.GetBytes(itemBytes, "NAPTestItem.TestItemContent.ItemSubstitutedForList.SubstituteItem").
			ForEach(func(key, value gjson.Result) bool {
				if _, ok := cfh.substitutes[itemRefId]; !ok { // watch for null values
					cfh.substitutes[itemRefId] = make(map[string]struct{}, 0)
				}
				// if so get the refid of the items it can sub for
				substituteRefId = value.Get("SubstituteItemRefId").String()
				cfh.substitutes[itemRefId][substituteRefId] = struct{}{}
				if _, ok := cfh.reverseSubstitutes[substituteRefId]; !ok { // watch for null values
					cfh.reverseSubstitutes[substituteRefId] = make(map[string]struct{}, 0)
				}
				cfh.reverseSubstitutes[substituteRefId][itemRefId] = struct{}{}
				return true // keep iterating
			})
	}

}

//
// implement the codeframe pipe interface, so this can be attached to a
// codeframe emitter.
//
func (cfh CodeframeHelper) ProcessCodeframeRecords(in chan *records.CodeframeRecord) chan *records.CodeframeRecord {
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
func (cfh *CodeframeHelper) extractLocationInStage() {

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

func removeDuplicateValues(strSlice []string) []string {
	keys := make(map[string]bool)
	list := []string{}

	// If the key(values of the slice) is not equal
	// to the already present value in new slice (list)
	// then we append it. else we jump on another element.
	for _, entry := range strSlice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

//
// we need to be able to reverse lookup the codeframe structure
// e.g. Test from Item - find the test/s an item was assinged to
// via testlets
//
// note: testlets are not normally re-used across tests, but the common
// exception is for them to be re-used in writing/alt_writing tests
//
func (cfh *CodeframeHelper) buildReverseLookup() {

	lookup := make(map[string]map[string][]string, 0)

	var testRefId, testletRefId, itemRefId string
	//
	// extract core lookup from codeframe
	//
	for _, cfBytes := range cfh.data["NAPCodeFrame"] {
		//
		// get the test id
		//
		testRefId = gjson.GetBytes(cfBytes, "NAPCodeFrame.NAPTestRefId").String()

		testDomain := gjson.GetBytes(cfBytes, "NAPTest.TestContent.Domain").String()
		testLevel := gjson.GetBytes(cfBytes, "NAPTest.TestContent.TestLevel.Code").String()
		testName := gjson.GetBytes(cfBytes, "NAPTest.TestContent.TestName").String()
		cfh.testchars[testRefId] = make(map[string]string, 0)
		cfh.testchars[testRefId]["Domain"] = testDomain
		cfh.testchars[testRefId]["Level"] = testLevel
		cfh.testchars[testRefId]["Name"] = testName

		//
		// iterate the nested json strucure & extract containers
		//
		gjson.GetBytes(cfBytes, "NAPCodeFrame.TestletList.Testlet").
			ForEach(func(key, value gjson.Result) bool {
				testletRefId = value.Get("NAPTestletRefId").String() // get the testlet refid
				value.Get("TestItemList.TestItem").
					ForEach(func(key, value gjson.Result) bool {
						itemRefId = value.Get("TestItemRefId").String() // get the item refid
						if _, ok := lookup[itemRefId]; !ok {            // avoid null nodes
							lookup[itemRefId] = make(map[string][]string, 0)
						}
						if _, ok := lookup[itemRefId][testletRefId]; !ok { // avoid null nodes
							lookup[itemRefId][testletRefId] = make([]string, 0)
						}
						lookup[itemRefId][testletRefId] = append(lookup[itemRefId][testletRefId], testRefId) // store the lookup
						if _, ok := cfh.item2test[itemRefId]; !ok {
							cfh.item2test[itemRefId] = make(map[string]struct{}, 0)
						}
						cfh.item2test[itemRefId][testRefId] = struct{}{}
						return true // keep iterating
					})
				return true // keep iterating
			})
	}

	//
	// also iterate just testlests as those used for alternate writing (for example)
	// are not reflected in the main codeframe
	//
	for _, testletBytes := range cfh.data["NAPTestlet"] {
		//
		// get the test/testlet id
		//
		testRefId = gjson.GetBytes(testletBytes, "NAPTestlet.NAPTestRefId").String()
		testletRefId = gjson.GetBytes(testletBytes, "NAPTestlet.RefId").String() // get the testlet refid
		//
		// iterate the nested json strucure & extract items
		//
		gjson.GetBytes(testletBytes, "NAPTestlet.TestItemList.TestItem").
			ForEach(func(key, value gjson.Result) bool {
				itemRefId = value.Get("TestItemRefId").String() // get the item refid
				if _, ok := lookup[itemRefId]; !ok {            // avoid null nodes
					lookup[itemRefId] = make(map[string][]string, 0)
				}
				if _, ok := lookup[itemRefId][testletRefId]; !ok { // avoid null nodes
					lookup[itemRefId][testletRefId] = make([]string, 0)
				}
				lookup[itemRefId][testletRefId] = append(lookup[itemRefId][testletRefId], testRefId) // store the lookup
				return true                                                                          // keep iterating
			})
	}

	for k, v := range lookup {
		for k1, _ := range v {
			lookup[k][k1] = removeDuplicateValues(lookup[k][k1])
		}
	}

	cfh.reverseLookup = lookup

}

//
// internal function to create list of writing rubric types
// / subscores from actual test data
//
func (cfh *CodeframeHelper) extractRubrics() {

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
func (cfh CodeframeHelper) WritingRubricTypes() []string {

	return cfh.WritingSubscoreTypes()

}

//
// alias for writing rubrics, known as subscores in results
//
func (cfh CodeframeHelper) WritingSubscoreTypes() []string {

	return cfh.rubrics
}

//
// get the list of Disability Adjustment Codes supported for this naplan cycle
//
func (cfh CodeframeHelper) GetDACs() []string {
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
// creates lookup sets of the items (refids) associated with
// each test/teslet combination
//
//
func (cfh *CodeframeHelper) extractExpectedTestletItems() {

	expectedItems := make(map[string]map[string]map[string]struct{}, 0)

	var cfTestRefId, testletRefId, itemRefId string
	for _, cfBytes := range cfh.data["NAPCodeFrame"] {
		//
		// get the test id
		//
		cfTestRefId = gjson.GetBytes(cfBytes, "NAPCodeFrame.NAPTestRefId").String()
		if _, ok := expectedItems[cfTestRefId]; !ok { // avoid null nodes
			expectedItems[cfTestRefId] = make(map[string]map[string]struct{}, 0)
		}
		//
		// iterate the nested json strucure & extract containers
		//
		gjson.GetBytes(cfBytes, "NAPCodeFrame.TestletList.Testlet").
			ForEach(func(key, value gjson.Result) bool {
				testletRefId = value.Get("NAPTestletRefId").String() // get the testlet refid
				value.Get("TestItemList.TestItem").
					ForEach(func(key, value gjson.Result) bool {
						itemRefId = value.Get("TestItemRefId").String()             // get the item refid
						if _, ok := expectedItems[cfTestRefId][testletRefId]; !ok { // avoid null nodes
							expectedItems[cfTestRefId][testletRefId] = make(map[string]struct{}, 0)
						}
						expectedItems[cfTestRefId][testletRefId][itemRefId] = struct{}{} // store the lookup
						return true                                                      // keep iterating
					})
				return true // keep iterating
			})
	}

	cfh.expectedItems = expectedItems
}

//
// For given test/testlet combination returns list of expected items
// - returned is a map[string]struct{} so acts as a set that can
// have lookups performed against it to test for presence of members
// the contents of the set are the refids of the expected items
//
func (cfh CodeframeHelper) GetExpectedTestletItems(testRefId, testletRefId string) map[string]struct{} {

	items, ok := cfh.expectedItems[testRefId][testletRefId]
	if !ok {
		return map[string]struct{}{}
	}

	return items

}

// return the test characteristics for an item
func (cfh CodeframeHelper) GetTest4Item(itemRefId string) []map[string]string {
	ret := make([]map[string]string, 0)
	tests, ok := cfh.item2test[itemRefId]
	if !ok {
		return ret
	}
	for k, _ := range tests {
		ret1 := make(map[string]string)
		ret1["Domain"] = cfh.testchars[k]["Domain"]
		ret1["Level"] = cfh.testchars[k]["Level"]
		ret1["Name"] = cfh.testchars[k]["Name"]
	}
	return ret
}

// return all codeframe items and their test characteristics
func (cfh CodeframeHelper) GetItemTests() map[string][]map[string]string {
	ret := make(map[string][]map[string]string, 0)
	for k, _ := range cfh.item2test {
		ret[k] = cfh.GetTest4Item(k)
	}
	return ret
}

//
// return the json block for a given testitem refid
// boolean return value indicates if a value was found
//
func (cfh CodeframeHelper) GetItem(refid string) (bool, []byte) {

	item, ok := cfh.data["NAPTestItem"][refid]
	if !ok {
		// fmt.Println("cfh unable to find TestItem", refid)
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
func (cfh CodeframeHelper) GetContainersForItem(refid string) map[string][]string {

	c := make(map[string][]string, 0)
	c = cfh.reverseLookup[refid]

	return c

}

//
// find location in stage for testlet
// comes from codeframe not testlet object
//
func (cfh CodeframeHelper) GetTestletLocationInStage(testletrefid string) string {

	lis, ok := cfh.locationInStage[testletrefid]
	if !ok {
		return ""
	}

	return lis
}

//
// for a given item refid, returns the set of refids (string) of the item
// this item substitutes for, returned as a set (map[string]struct{}) so can
// be tested for membership.
// boolean return can be used simply to determine if this item is
// a substitute
//
func (cfh CodeframeHelper) IsSubstituteItem(itemRefid string) (map[string]struct{}, bool) {

	subsForRefId, ok := cfh.substitutes[itemRefid]
	if !ok {
		return map[string]struct{}{}, ok
	}
	return subsForRefId, ok
}

// for a given item refid, returns the set of refids (string) of the items
// that can substitute for it, returned as a set (map[string]struct{}) so can
// be tested for membership.
// boolean return can be used simply to determine if this item has
// a substitute
func (cfh CodeframeHelper) HasSubstituteItem(itemRefid string) (map[string]struct{}, bool) {

	subsForRefId, ok := cfh.reverseSubstitutes[itemRefid]
	if !ok {
		return map[string]struct{}{}, ok
	}
	return subsForRefId, ok
}

//
// pass a json path to retrieve the value at that location as a
// string
// refid - the identifier of the object to be queried
// queryPath the gjson query string to apply to the object
//
func (cfh CodeframeHelper) GetCodeframeObjectValueString(refid, queryPath string) string {

	//
	// get the root object
	//
	objName := strings.Split(queryPath, ".")[0]
	record, ok := cfh.data[objName][refid]
	if !ok {
		// log.Println("GetCodeframeObjectValueString() cannot find value for path:", queryPath, refid)
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
func (cfh CodeframeHelper) GetItemTestletSequenceNumber(itemrefid, testletrefid string) string {

	sqnum, ok := cfh.itemSequence[itemrefid][testletrefid]
	if !ok {
		// log.Println("No sequence number found for item:testlet pair:", itemrefid, testletrefid)
		return ""
	}

	return sqnum

}
