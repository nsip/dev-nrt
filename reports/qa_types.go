package reports

import (
	"strings"

	"github.com/tidwall/gjson"
)

//
// helper classes fro qa reports especially qaSchools and qaSchoolsWritingExtract
//

//
// helper types for data reporting
//
// aliases to help keep purpose clear
type schoolId = string
type studentId = string
type yearLevel = string
type testLevel = string
type testDomain string
type pCode = string // participation

//
// schools - unique set
type schools = map[schoolId][]byte

//
// school - total students
type studentSchool struct {
	school  schoolId
	student studentId
}

//
// students - unique set
type students = map[studentSchool]struct{}

//
// student counter
type studentCounter = map[schoolId]int

//
// school - students by year-level
type studentSchoolYrLevel struct {
	school schoolId
	year   yearLevel
}
type yrLevelCounter = map[studentSchoolYrLevel]int

//
// school - students by test-level
type studentSchoolTestLevel struct {
	school schoolId
	level  testLevel
}
type testLevelCounter = map[studentSchoolTestLevel]int

//
// school - total test attempts
type testAttemptsCounter = map[schoolId]int

//
// school - year & domain attempts
type studentSchoolTestLevelDomain struct {
	school schoolId
	level  testLevel
	domain testDomain
}
type domainAttemptsCounter = map[studentSchoolTestLevelDomain]int

//
// school - attempts by participation status
type studentSchoolParticipation struct {
	school schoolId
	pcode  pCode
}
type participationCounter = map[studentSchoolParticipation]int

//
// school - disruptions
type disruptionsCounter = map[schoolId]int

//
// school - writing extracts by year
type studentSchoolTestLevelExtract struct {
	school schoolId
	level  testLevel
}
type writingExtractCounter = map[studentSchoolTestLevelExtract]int

//
// summary object to contain json report strucures
//
type qaWeSummary struct {
	CalculatedFields []byte
	SchoolInfo       []byte
}

//
// pass a gjson path to retrieve the value at that location as a
// string, makes the summary conform to the standard csv rendering loop
//
func (s *qaWeSummary) GetValueString(queryPath string) string {

	//
	// get the root object
	//
	objName := strings.Split(queryPath, ".")[0]
	var data []byte
	switch objName {
	case "CalculatedFields":
		data = s.CalculatedFields
	case "SchoolInfo":
		data = s.SchoolInfo
	default:
		data = []byte{}
	}

	return gjson.GetBytes(data, queryPath).String()
}
