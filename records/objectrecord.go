package records

import (
	"strings"

	"github.com/tidwall/gjson"
)

//
// simple wrapper for data objects
// used for reports that simply format data-objects
// into csv tables
//
type ObjectRecord struct {
	RecordType       string
	Json             []byte
	CalculatedFields []byte
}

func (cfr *ObjectRecord) RefId() string {
	return gjson.GetBytes(cfr.Json, "*.RefId").String()
}

//
// pass a gjson path to retrieve the value at that location as a
// string
//
func (cfr *ObjectRecord) GetValueString(queryPath string) string {

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
