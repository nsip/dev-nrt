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
