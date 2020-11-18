package repository

import (
	"errors"
	"fmt"

	"github.com/nsip/dev-nrt/sec"
	"github.com/tidwall/gjson"
)

//
// signature for an indexing function to
// use on the data objects;
// takes in a conversion result, returns the key for that
// json blob as bytes.
//
type IndexFunc func(r sec.Result) ([]byte, error)

//
// index func to retrieve sif object refid.
// Creates an index key in the form ObjectName:RefId
// - only index needed for majority of the objects
//
func IdxSifObjectByTypeAndRefId() IndexFunc {

	return func(r sec.Result) ([]byte, error) {
		refid := gjson.GetBytes(r.Json, "*.RefId")
		if !refid.Exists() {
			return nil, errors.New("could not find RefId")
		}
		idxKey := fmt.Sprintf("%s:%s", r.Name, refid.String())

		return []byte(idxKey), nil
	}

}

//
// index object (e.g. result-set) by student and test
// Creates an index key in the form ObjectName:StudentPersonalRefId:NAPTestRefId
//
func IdxByTypeStudentAndTest() IndexFunc {

	return func(r sec.Result) ([]byte, error) {
		sprefid := gjson.GetBytes(r.Json, "*.StudentPersonalRefId")
		if !sprefid.Exists() {
			return nil, errors.New("could not find StudentPersonalRefId")
		}
		testrefid := gjson.GetBytes(r.Json, "*.NAPTestRefId")
		if !testrefid.Exists() {
			return nil, errors.New("could not find NAPTestRefId")
		}
		idxKey := fmt.Sprintf("%s:%s:%s", r.Name, sprefid.String(), testrefid.String())

		return []byte(idxKey), nil
	}

}

//
//
//
