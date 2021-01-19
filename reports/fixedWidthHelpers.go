package reports

// slightly modified from original helpers from the excellent:
// https://github.com/gosuri/uiprogress/blob/master/util/strutil/strutil.go

import (
	"bytes"
	"time"
)

var defaultPaddingToken byte = ' '

var ItemCorrectness = map[string]string{
	"NotAttempted": "9",
	"NotInPath":    "9",
	"Correct":      "1",
	"Incorrect":    "0",
}

//
// helper for pearson/acara format fixed-width reports
// converts an item correctness token to a single number
//
func AcaraEncodeItemCorrectness(correctness string) string {

	code, ok := ItemCorrectness[correctness]
	if !ok {
		return "0" // same default for unknown values as n2
	}

	return code

}

// PadRight returns a new string of a specified length in which the end of the current string is padded with spaces or with a specified Unicode character.
func PadRight(str string, length int, pad byte) string {
	if len(str) >= length {
		return str[:length] // if actual string is longer, truncate
	}
	buf := bytes.NewBufferString(str)
	for i := 0; i < length-len(str); i++ {
		buf.WriteByte(pad)
	}
	return buf.String()
}

// PadLeft returns a new string of a specified length in which the beginning of the current string is padded with spaces or with a specified Unicode character.
func PadLeft(str string, length int, pad byte) string {
	if len(str) >= length {
		return str[:length] // if actual string is longer, truncate
	}
	var buf bytes.Buffer
	for i := 0; i < length-len(str); i++ {
		buf.WriteByte(pad)
	}
	buf.WriteString(str)
	return buf.String()
}

// Resize resizes the string with the given length. It ellipses with '...' when the string's length exceeds
// the desired length or pads spaces to the right of the string when length is smaller than desired
func Resize(s string, length uint) string {
	n := int(length)
	if len(s) == n {
		return s
	}
	// Pads only when length of the string smaller than len needed
	s = PadRight(s, n, ' ')
	if len(s) > n {
		b := []byte(s)
		var buf bytes.Buffer
		for i := 0; i < n-3; i++ {
			buf.WriteByte(b[i])
		}
		buf.WriteString("...")
		s = buf.String()
	}
	return s
}

// PrettyTime returns the string representation of the duration. It rounds the time duration to a second and returns a "---" when duration is 0
func PrettyTime(t time.Duration) string {
	if t == 0 {
		return "---"
	}
	return (t - (t % time.Second)).String()
}
