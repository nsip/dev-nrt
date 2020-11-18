package records

import (
	"strings"

	"github.com/tidwall/gjson"
)

type EventOrientedRecord struct {
	NAPEventStudentLink   []byte
	StudentPersonal       []byte
	SchoolInfo            []byte
	NAPTest               []byte
	NAPStudentResponseSet []byte
	Err                   error
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
	pc := gjson.GetBytes(eor.NAPTest, "NAPEventStudentLink.ParticipationCode").String()
	return strings.EqualFold(pc, "P")
}
