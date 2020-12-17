package records

import "github.com/tidwall/gjson"

//
// simple wrapper for objects associated with the
// codeframe, can be tests, telstlets and testitems
//
//
type CodeframeRecord struct {
	RecordType string
	Json       []byte
}

func (cfr *CodeframeRecord) RefID() string {
	return gjson.GetBytes(cfr.Json, "*.RefId").String()
}


//
// pass a json path to retrieve the value at that location as a
// string
//
func (cfr *CodeframeRecord) GetValueString(queryPath string) string {

	return gjson.GetBytes(cfr.Json, queryPath).String()
}