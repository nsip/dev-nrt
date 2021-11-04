package reports

import (
	"bufio"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/nsip/dev-nrt/records"
	"github.com/tidwall/sjson"
)

type NswAllPearsonY7 struct {
	baseReport // embed common setup capability
}

//
// Pearson/ACARA fixed width encoding of domain item response correctness
//
func NswAllPearsonY7Report() *NswAllPearsonY7 {

	r := NswAllPearsonY7{}
	r.initialise("./config/NswAllPearsonY7.toml")
	r.printStatus()

	return &r

}

//
// implement the ...Pipe interface, core work of the
// report engine.
//
func (r *NswAllPearsonY7) ProcessStudentRecords(in chan *records.StudentOrientedRecord) chan *records.StudentOrientedRecord {

	out := make(chan *records.StudentOrientedRecord)
	go func() {
		defer close(out)
		// open the fixed format file writer - not a csv file
		w := bufio.NewWriter(r.outF)
		defer r.outF.Close()
		defer w.Flush()

		var result, paddedResult string
		var length int
		var convErr error
		var row strings.Builder // single string record for this format

		for sor := range in {
			if !r.config.activated { // only process if activated
				out <- sor
				continue
			}

			if sor.GetValueString("CalculatedFields.YrLevel") != "7" { // only for yr 3
				out <- sor
				continue
			}

			//
			// generate any calculated fields required
			//
			sor.CalculatedFields = r.calculateFields(sor)

			//
			// now loop through the ouput definitions to create a
			// row of results
			//
			for i, query := range r.config.queries {
				result = sor.GetValueString(query)
				length, convErr = strconv.Atoi(r.config.header[i])
				if convErr != nil {
					log.Println("WARNING: unexpected value for field length in ", r.configFileName, r.config.header[i])
				}
				paddingToken := defaultPaddingToken
				if r.config.header[i][0] == '0' {
					paddingToken = zeroPaddingToken
				}
				paddedResult = PadLeft(result, length, paddingToken)
				row.WriteString(paddedResult)
			}
			// write the row to the output file
			if _, err := fmt.Fprintln(w, row.String()); err != nil {
				fmt.Println("Warning: error writing record to output file:", r.config.name, err)
			}
			row.Reset()

			out <- sor
		}
	}()
	return out
}

//
// generates a block of json that can be added to the
// record containing values that are not in the original data
//
//
func (r *NswAllPearsonY7) calculateFields(sor *records.StudentOrientedRecord) []byte {

	json := sor.CalculatedFields // keep any existing data

	// create truncated format birthdate
	fullDob := sor.GetValueString("StudentPersonal.PersonInfo.Demographics.BirthDate")
	truncDob := strings.ReplaceAll(fullDob, "-", "")
	json, _ = sjson.SetBytes(json, "CalculatedFields.TruncatedDOB", truncDob)

	return json
}
