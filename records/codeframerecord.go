package records

import (
	"strings"

	"github.com/tidwall/gjson"
)

//
// simple wrapper for objects associated with the
// codeframe, can be tests, testlets and testitems
//
//
type CodeframeRecord struct {
	RecordType       string
	Json             []byte
	CalculatedFields []byte
}

func (cfr *CodeframeRecord) RefId() string {
	return gjson.GetBytes(cfr.Json, "*.RefId").String()
}

//
// pass a json path to retrieve the value at that location as a
// string
//
func (cfr *CodeframeRecord) GetValueString(queryPath string) string {

	//
	// get the root object
	//
	objName := strings.Split(queryPath, ".")[0]
	var data []byte
	switch objName {
	case "CalculatedFields":
		data = cfr.CalculatedFields
	default:
		data = cfr.Json
	}

	return gjson.GetBytes(data, queryPath).String()
}
