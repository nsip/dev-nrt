package records

import (
	"log"
	"strings"

	"github.com/tidwall/gjson"
)

type EventOrientedRecord struct {
	NAPEventStudentLink      []byte
	StudentPersonal          []byte
	SchoolInfo               []byte
	NAPTest                  []byte
	NAPStudentResponseSet    []byte
	CalculatedFields         []byte
	Err                      error
	HasNAPEventStudentLink   bool
	HasStudentPersonal       bool
	HasSchoolInfo            bool
	HasNAPTest               bool
	HasNAPStudentResponseSet bool
}

func (eor *EventOrientedRecord) StudentPersonalRefId() string {
	return gjson.GetBytes(eor.NAPEventStudentLink, "*.StudentPersonalRefId").String()
}

func (eor *EventOrientedRecord) SchoolInfoRefId() string {
	return gjson.GetBytes(eor.NAPEventStudentLink, "*.SchoolInfoRefId").String()
}

func (eor *EventOrientedRecord) NAPTestRefId() string {
	return gjson.GetBytes(eor.NAPEventStudentLink, "*.NAPTestRefId").String()
}

func (eor *EventOrientedRecord) IsWritingResponse() bool {
	td := gjson.GetBytes(eor.NAPTest, "NAPTest.TestContent.Domain").String()
	return strings.EqualFold(td, "writing")
}

func (eor *EventOrientedRecord) ParticipatedInTest() bool {
	pc := gjson.GetBytes(eor.NAPEventStudentLink, "NAPEventStudentLink.ParticipationCode").String()
	return strings.EqualFold(pc, "P")
}

//
// pass a json path to retrieve the value at that location as a
// string
//
func (eor *EventOrientedRecord) GetValueString(queryPath string) string {

	//
	// get the root object
	//
	objName := strings.Split(queryPath, ".")[0]
	var data []byte
	switch objName {
	case "NAPEventStudentLink":
		data = eor.NAPEventStudentLink
	case "StudentPersonal":
		data = eor.StudentPersonal
	case "SchoolInfo":
		data = eor.SchoolInfo
	case "NAPTest":
		data = eor.NAPTest
	case "NAPStudentResponseSet":
		data = eor.NAPStudentResponseSet
	case "CalculatedFields":
		data = eor.CalculatedFields
	default:
		log.Println("Event record cannot find value for path:", queryPath)
		return ""
	}

	return gjson.GetBytes(data, queryPath).String()
}
