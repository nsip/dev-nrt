package reports

import (
	"encoding/csv"
	"fmt"
	"regexp"
	"strings"

	"github.com/nsip/dev-nrt/records"
)

type SystemExtraneousCharactersStudents struct {
	baseReport // embed common setup capability
	re         *regexp.Regexp
}

//
// Outputs student info for any student who has non-standard characters in their
// name attributes
//
func SystemExtraneousCharactersStudentsReport() *SystemExtraneousCharactersStudents {

	r := SystemExtraneousCharactersStudents{}
	r.initialise("./config/SystemExtraneousCharactersStudents.toml")
	r.printStatus()
	r.getRegex()

	return &r

}

//
// implement the ...Pipe interface, core work of the
// report engine.
//
func (r *SystemExtraneousCharactersStudents) ProcessObjectRecords(in chan *records.ObjectRecord) chan *records.ObjectRecord {

	out := make(chan *records.ObjectRecord)
	go func() {
		defer close(out)
		// open the csv file writer, and set the header
		w := csv.NewWriter(r.outF)
		defer r.outF.Close()
		w.Write(r.config.header)
		defer w.Flush()

		for or := range in {
			if !r.config.activated { // only process if activated
				out <- or
				continue
			}

			if or.RecordType != "StudentPersonal" { // only inspect studentpersonals
				out <- or
				continue
			}

			if r.validCharacters(or) {
				out <- or
				continue
			}

			//
			// generate any calculated fields required
			//
			or.CalculatedFields = r.calculateFields(or)

			//
			// now loop through the ouput definitions to create a
			// row of results
			//
			var result string
			var row []string = make([]string, 0, len(r.config.queries))
			for _, query := range r.config.queries {
				result = or.GetValueString(query)
				row = append(row, result)
			}
			// write the row to the output file
			if err := w.Write(row); err != nil {
				fmt.Println("Warning: error writing record to csv:", r.config.name, err)
			}

			out <- or
		}
	}()
	return out
}

//
// returns true if all names meet specification, false otherwise
//
func (r *SystemExtraneousCharactersStudents) validCharacters(or *records.ObjectRecord) bool {

	var b strings.Builder
	b.WriteString(or.GetValueString("StudentPersonal.PersonInfo.Name.FamilyName"))
	b.WriteString(or.GetValueString("StudentPersonal.PersonInfo.Name.GivenName"))
	b.WriteString(or.GetValueString("StudentPersonal.PersonInfo.Name.MiddleName"))
	b.WriteString(or.GetValueString("StudentPersonal.PersonInfo.Name.PreferredGivenName"))

	if len(r.re.FindString(b.String())) > 0 {
		return false
	}

	return true

}

//
// gets the name-checking regex from the report's config if available
// if none found uses the default "[^a-zA-Z' -]"
//
func (r *SystemExtraneousCharactersStudents) getRegex() {

	var regEx, defaultRegex *regexp.Regexp
	defaultRegex = regexp.MustCompile("[^a-zA-Z' -]") // regex of allowed characters

	// get the regex from the config if available
	configRegex, ok := r.config.tree.Get("options.regex").(string)
	if !ok {
		regEx = defaultRegex // nothing found use the default
	}

	regEx, err := regexp.Compile(configRegex)
	if err != nil {
		regEx = defaultRegex // invalid regex use the default
	}

	r.re = regEx
}

//
// generates a block of json that can be added to the
// record containing values that are not in the original data
//
//
func (r *SystemExtraneousCharactersStudents) calculateFields(or *records.ObjectRecord) []byte {

	return or.CalculatedFields
}
