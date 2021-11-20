package reports

import (
	"encoding/csv"
	"fmt"

	"github.com/iancoleman/strcase"
	"github.com/nsip/dev-nrt/records"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

type QaSchoolsWritingExtract struct {
	baseReport // embed common setup capability
	schools
	students
	studentCounter
	yrLevelCounter
	testLevelCounter
	testAttemptsCounter
	domainAttemptsCounter
	participationCounter
	disruptionsCounter
	writingExtractCounter
}

//
// Summary of student distributions accross year-levels and tests with participation data
// variant of qaSchools focussed on just writing test
//
// NOTE: This is a Collector report, it harvests data from the stream but
// only writes it out once all data has passed through.
//
func QaSchoolsWritingExtractReport() *QaSchoolsWritingExtract {

	r := QaSchoolsWritingExtract{
		schools:               make(schools, 0),
		students:              make(students, 0),
		studentCounter:        make(studentCounter, 0),
		yrLevelCounter:        make(yrLevelCounter, 0),
		testLevelCounter:      make(testLevelCounter, 0),
		testAttemptsCounter:   make(testAttemptsCounter, 0),
		domainAttemptsCounter: make(domainAttemptsCounter, 0),
		participationCounter:  make(participationCounter, 0),
		disruptionsCounter:    make(disruptionsCounter, 0),
		writingExtractCounter: make(writingExtractCounter, 0),
	}

	r.initialise("./config/QaSchoolsWritingExtract.toml")
	r.printStatus()

	return &r

}

//
// implement the EventPipe interface, core work of the
// report engine.
//
func (r *QaSchoolsWritingExtract) ProcessEventRecords(in chan *records.EventOrientedRecord) chan *records.EventOrientedRecord {

	out := make(chan *records.EventOrientedRecord)
	go func() {
		defer close(out)
		// open the csv file writer, and set the header
		w := csv.NewWriter(r.outF)
		defer r.outF.Close()
		w.Write(r.config.header)
		defer w.Flush()

		for eor := range in {
			if !r.config.activated { // only process if activated
				out <- eor
				continue
			}
			r.collectSummaryInitial(eor)

			if !eor.IsWritingResponse() {
				out <- eor
				continue
			}

			//
			// gather data from the record
			//
			r.collectSummary(eor)

			out <- eor // pass on to next stage

			// this is a collector so aggregated data is written out when the
			// input channel closes, see below...

		}

		//
		// write out collected data
		//
		for schoolid, schoolinfo := range r.schools {
			// create a new reporting object
			summ := &qaWeSummary{CalculatedFields: []byte{}}
			//
			// generate any calculated fields required
			//
			summ.CalculatedFields = r.calculateFields(schoolid)
			summ.SchoolInfo = schoolinfo

			//
			// now loop through the ouput definitions to create a
			// row of results
			//
			var result string
			var row []string = make([]string, 0, len(r.config.queries))
			for _, query := range r.config.queries {
				result = summ.GetValueString(query)
				row = append(row, result)
			}
			// write the row to the output file
			if err := w.Write(row); err != nil {
				fmt.Println("Warning: error writing record to csv:", r.config.name, err)
			}
		}

	}()
	return out
}

//
// generates a block of json that can be added to the
// record containing values that are not in the original data
//
//
func (r *QaSchoolsWritingExtract) calculateFields(schoolAcaraId schoolId) []byte {

	json := []byte{}
	var path string

	// the school
	json, _ = sjson.SetBytes(json, "CalculatedFields.QaSchoolsWritingExtract.SchoolACARAId", schoolAcaraId)

	// studentCounter
	studentCount := r.studentCounter[schoolAcaraId]
	path = "CalculatedFields.QaSchoolsWritingExtract.TotalStudents"
	json, _ = sjson.SetBytes(json, path, studentCount)

	// yrLevelCounter
	for ssyl, count := range r.yrLevelCounter {
		if ssyl.school != schoolAcaraId {
			continue
		}
		path = fmt.Sprintf("CalculatedFields.QaSchoolsWritingExtract.YearLevel.%s.TotalStudents", ssyl.year)
		json, _ = sjson.SetBytes(json, path, count)
	}

	// testLevelCounter
	for sstl, count := range r.testLevelCounter {
		if sstl.school != schoolAcaraId {
			continue
		}
		path = fmt.Sprintf("CalculatedFields.QaSchoolsWritingExtract.TestLevel.%s.TotalStudents", sstl.level)
		json, _ = sjson.SetBytes(json, path, count)
	}

	// testAttemptsCounter
	attemptsCount := r.testAttemptsCounter[schoolAcaraId]
	path = "CalculatedFields.QaSchoolsWritingExtract.TotalTestAttempts"
	json, _ = sjson.SetBytes(json, path, attemptsCount)

	// domainAttemptsCounter
	for sstd, count := range r.domainAttemptsCounter {
		if sstd.school != schoolAcaraId {
			continue
		}
		path = fmt.Sprintf("CalculatedFields.QaSchoolsWritingExtract.TestLevel.%s.TestDomain.%s.TotalAttempts", sstd.level, sstd.domain)
		json, _ = sjson.SetBytes(json, path, count)
	}

	// participationCounter
	for ssp, count := range r.participationCounter {
		if ssp.school != schoolAcaraId {
			continue
		}
		path = fmt.Sprintf("CalculatedFields.QaSchoolsWritingExtract.ParticipationCode.%s.TotalAttempts", ssp.pcode)
		json, _ = sjson.SetBytes(json, path, count)
	}

	// disruptionsCounter
	disruptionsCount := r.disruptionsCounter[schoolAcaraId]
	path = "CalculatedFields.QaSchoolsWritingExtract.TotalDisruptions"
	json, _ = sjson.SetBytes(json, path, disruptionsCount)

	// writingExtractCounter
	for sstlx, count := range r.writingExtractCounter {
		if sstlx.school != schoolAcaraId {
			continue
		}
		path = fmt.Sprintf("CalculatedFields.QaSchoolsWritingExtract.TestLevel.%s.TotalExtracts", sstlx.level)
		json, _ = sjson.SetBytes(json, path, count)
	}

	return json
}

// summary statistics applicable to all records
func (r *QaSchoolsWritingExtract) collectSummaryInitial(eor *records.EventOrientedRecord) {
	schoolACARAId := eor.GetValueString("NAPEventStudentLink.SchoolACARAId")
	studentId := eor.GetValueString("NAPEventStudentLink.PlatformStudentIdentifier")
	ss := studentSchool{school: schoolACARAId, student: studentId}
	if _, ok := r.students[ss]; !ok { // only update student counters on first encounter
		r.students[ss] = struct{}{}
		r.studentCounter[schoolACARAId]++
	}
}

//
// extract summary statistics from the record
//
func (r *QaSchoolsWritingExtract) collectSummary(eor *records.EventOrientedRecord) {

	// schools    []schoolId
	schoolACARAId := eor.GetValueString("NAPEventStudentLink.SchoolACARAId")
	r.schools[schoolACARAId] = eor.SchoolInfo

	// student counters
	studentTestLevel := eor.GetValueString("StudentPersonal.MostRecent.TestLevel.Code")
	// yrLevelCounter
	studentYrLevel := eor.GetValueString("StudentPersonal.MostRecent.YearLevel.Code")
	ssyl := studentSchoolYrLevel{school: schoolACARAId, year: studentYrLevel}
	r.yrLevelCounter[ssyl]++
	// testLevelCounter
	sstl := studentSchoolTestLevel{school: schoolACARAId, level: studentTestLevel}
	r.testLevelCounter[sstl]++

	// testAttemptsCounter
	r.testAttemptsCounter[schoolACARAId]++

	// domainAttemptsCounter
	if eor.ParticipatedInTest() {
		domain := eor.GetValueString("NAPTest.TestContent.Domain")
		ccDomain := strcase.ToCamel(domain)
		sstd := studentSchoolTestLevelDomain{school: schoolACARAId, level: studentTestLevel, domain: testDomain(ccDomain)}
		r.domainAttemptsCounter[sstd]++
	}

	// participationCounter
	studentPcode := eor.GetValueString("NAPEventStudentLink.ParticipationCode")
	ssp := studentSchoolParticipation{school: schoolACARAId, pcode: studentPcode}
	r.participationCounter[ssp]++

	// disruptionsCounter
	disruptions := gjson.GetBytes(eor.NAPEventStudentLink, "NAPEventStudentLink.TestDisruptionList.TestDisruption").Array()
	for range disruptions {
		r.disruptionsCounter[schoolACARAId]++
	}

	// writingExtractCounter
	switch studentPcode {
	case "P", "F", "R", "S": //, "E":
		sstlx := studentSchoolTestLevelExtract{school: schoolACARAId, level: studentTestLevel}
		r.writingExtractCounter[sstlx]++
	}

}
