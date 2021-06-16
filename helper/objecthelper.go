//
// processing of data connection
//
// This package created to provide a single encapsulated helper that can be fed once
// with the object data and then answer all object linkage related needs.
//
//
package helper

import (
	"github.com/tidwall/gjson"

	"github.com/nsip/dev-nrt/pipelines"
	"github.com/nsip/dev-nrt/records"
	"github.com/nsip/dev-nrt/repository"
)

//
// Encapsulates data and helper methods to make
// working with objects easier
//
type ObjectHelper struct {
	data        map[string]map[string][]byte
	incodeframe map[string]bool
}

//
// Creates a new object helper instance.
// r - a repository containing the rrd data
//
func NewObjectHelper(r *repository.BadgerRepo) (ObjectHelper, error) {

	h := ObjectHelper{
		data:        make(map[string]map[string][]byte, 0),
		incodeframe: make(map[string]bool),
	} // initialise the internal maps

	// wrap repo in emitter
	opts := []records.Option{records.EmitterRepository(r)}
	em, err := records.NewEmitter(opts...)
	if err != nil {
		return h, err
	}

	// create a simple one-element pipeline, with this helper as the only processor
	cfpl := pipelines.NewObjectPipeline(h)
	defer cfpl.Close()
	// spawn a no-op reader to consume pipeline output
	go cfpl.Dequeue(func(cfr *records.ObjectRecord) { cfr = nil })
	// iterate the codeframe dataset, will be handled by Process... method
	for cfr := range em.ObjectStream() {
		cfpl.Enqueue(cfr)
	}

	return h, nil

}

// implement the object pipe interface, so this can be attached to a
// object emitter.
//
func (cfh ObjectHelper) ProcessObjectRecords(in chan *records.ObjectRecord) chan *records.ObjectRecord {
	out := make(chan *records.ObjectRecord)
	go func() {
		defer close(out)

		// collect all object data
		for cfr := range in {
			// watch out for null nodes in map
			if _, ok := cfh.data[cfr.RecordType]; !ok {
				cfh.data[cfr.RecordType] = make(map[string][]byte, 0)
			}
			cfh.data[cfr.RecordType][cfr.RefId()] = cfr.Json

			if cfr.RecordType == "NAPCodeFrame" {
				testid := cfr.GetValueString("NAPCodeFrame.NAPTestRefId")
				cfh.incodeframe[testid] = true

				gjson.GetBytes(cfr.Json, "NAPCodeFrame.TestletList.Testlet").
					ForEach(func(key, value gjson.Result) bool {
						testletRefId := value.Get("NAPTestletRefId").String()
						cfh.incodeframe[testletRefId] = true
						//
						// now iterate testlet item responses
						//
						value.Get("TestItemList.TestItem").
							ForEach(func(key, value gjson.Result) bool {
								//
								// get item identifiers
								//
								itemRefId := value.Get("TestItemRefId").String()
								cfh.incodeframe[itemRefId] = true
								return true // keep iterating, move on to next item response
							})
						return true // keep iterating
					})

			}

			out <- cfr
		}

	}()
	return out

}

// given GUID, return its type
func (cfh ObjectHelper) GetTypeFromGuid(guid string) string {
	var ok bool
	types := []string{"NAPTest", "NAPTestlet", "NAPTestItem", "NAPCodeFrame", "NAPEventStudentLink", "NAPTestScoreSummary", "NAPStudentResponseSet", "SchoolInfo", "StudentPersonal"}
	for _, t := range types {
		if _, ok = cfh.data[t][guid]; ok {
			return t
		}
	}
	return ""
}

// is this object referenced by the codeframe?
func (cfh ObjectHelper) InCodeFrame(guid string) bool {
	_, ok := cfh.incodeframe[guid]
	return ok
}
