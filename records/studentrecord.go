package records

import (
	"errors"
	"strings"

	"github.com/iancoleman/strcase"
	"github.com/tidwall/gjson"
)

//
// As this record aggregates, these aliases and constants are used
// to create a data structure that holds artefacts by domain, where
// strucure is [NAPTest RefId][record type - event, response,...] = json blob
//
type RecordType int
type outputs map[string]map[RecordType][]byte

// fixed set of possible artefacts
const (
	rtEvent RecordType = iota
	rtResponse
	rtTest
)

//
// For each student accumulates all responses, events etc. for
// processing by reports.
//
type StudentOrientedRecord struct {
	StudentPersonal    []byte
	CalculatedFields   []byte
	SchoolInfo         []byte
	naplanOutputs      outputs
	Err                error
	HasStudentPersonal bool
	HasSchoolInfo      bool
}

//
// returns a new empty student record
// If you choose to initialise an instance manually
// rather than with this method
// remember to instantiate the naplanOuptuts map
// to avoid nil pointer errors
//
func NewStudentOrientedRecord() *StudentOrientedRecord {
	sor := StudentOrientedRecord{
		CalculatedFields: []byte{},
		naplanOutputs:    make(outputs, 0),
	}
	return &sor
}

//
// returns the refid of the student represented in this record
//
func (sor *StudentOrientedRecord) StudentPersonalRefId() string {
	return gjson.GetBytes(sor.StudentPersonal, "*.RefId").String()
}

//
// returns a school identifier for this student
//
func (sor *StudentOrientedRecord) SchoolInfoRefId() string {

	// iterate naplan data
	for _, data := range sor.naplanOutputs {
		// find an event & extract the school refid
		eventJson := data[rtEvent]
		if schoolRefId := gjson.GetBytes(eventJson, "NAPEventStudentLink.SchoolInfoRefId"); schoolRefId.Exists() {
			return schoolRefId.String() // return as soon as we have one
		}

	}

	// if no events found, return empty string as we have no way
	// of knowing which school the student was tested at
	return ""
}

//
// pass a json path to retrieve the value at that location as a
// string
//
func (sor *StudentOrientedRecord) GetValueString(queryPath string) string {

	//
	// get the root query object
	//
	objName := strings.Split(queryPath, ".")[0]
	var data []byte
	switch objName {
	case "StudentPersonal":
		data = sor.StudentPersonal
	case "SchoolInfo":
		data = sor.SchoolInfo
	case "CalculatedFields":
		data = sor.CalculatedFields
	default:
		// if not an object, then assume extended codeframe refrence
		return sor.codeframeValueString(queryPath)
	}

	return gjson.GetBytes(data, queryPath).String()
}

func (sor *StudentOrientedRecord) codeframeValueString(queryPath string) string {
	return ""
}

//
// Returns a map of events for this student where
// map key is the CamelCase name of the test doamin
// map value is the json blob of the event
//
func (sor *StudentOrientedRecord) GetEventsByDomain() map[string][]byte {

	ebd := make(map[string][]byte, 0)

	for _, records := range sor.naplanOutputs {
		// get the domain from the test record
		test := records[rtTest]
		domain := gjson.GetBytes(test, "NAPTest.TestContent.Domain").String()
		// camel-case the daomin name for use in json 'Grammar and Punctuation' -> 'GrannarAndPunctuation'
		ccdomain := strcase.ToCamel(domain)
		// create the domain/event pair
		ebd[ccdomain] = records[rtEvent]
	}

	return ebd
}

//
// Adds a NAPTestEvent to the record
//
func (sor *StudentOrientedRecord) AddEvent(jsonNAPEventStudentLink []byte) error {

	testRefId := gjson.GetBytes(jsonNAPEventStudentLink, "NAPEventStudentLink.NAPTestRefId").String()
	if testRefId == "" {
		return errors.New("no naptest refid found in event")
	}
	if _, ok := sor.naplanOutputs[testRefId]; !ok { // watch out for empty members
		sor.naplanOutputs[testRefId] = make(map[RecordType][]byte, 0)
	}
	// store the event
	sor.naplanOutputs[testRefId][rtEvent] = jsonNAPEventStudentLink

	return nil

}

//
// Adds a NAPStudentResponseSet to the record
//
func (sor *StudentOrientedRecord) AddResponse(jsonNAPStudentResponseSet []byte) error {

	testRefId := gjson.GetBytes(jsonNAPStudentResponseSet, "NAPStudentResponseSet.NAPTestRefId").String()
	if testRefId == "" {
		return errors.New("no naptest refid found in response")
	}
	if _, ok := sor.naplanOutputs[testRefId]; !ok { // watch out for empty members
		sor.naplanOutputs[testRefId] = make(map[RecordType][]byte, 0)
	}
	// store the response
	sor.naplanOutputs[testRefId][rtResponse] = jsonNAPStudentResponseSet

	return nil

}

//
// Adds a NAPTest to the record
//
func (sor *StudentOrientedRecord) AddTest(jsonNAPTest []byte) error {

	testRefId := gjson.GetBytes(jsonNAPTest, "*.RefId").String()
	if testRefId == "" {
		return errors.New("no refid found in test")
	}
	if _, ok := sor.naplanOutputs[testRefId]; !ok { // watch out for empty members
		sor.naplanOutputs[testRefId] = make(map[RecordType][]byte, 0)
	}
	// store the response
	sor.naplanOutputs[testRefId][rtTest] = jsonNAPTest

	return nil

}

//
// returns the rerfids of the tests taken by this student
//
func (sor *StudentOrientedRecord) GetNAPTestRefIds() []string {

	testids := []string{}

	for testid := range sor.naplanOutputs {
		testids = append(testids, testid)
	}

	return testids

}
