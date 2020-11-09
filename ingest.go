package nrt

import (
	"fmt"

	"github.com/nsip/dev-nrt/files"
	"github.com/nsip/dev-nrt/repo"
	"github.com/nsip/dev-nrt/sec"
)

func IngestResults(folderName string) error {

	r, err := repo.NewBadgerRepo("./kv/")
	if err != nil {
		return err
	}
	defer r.Close()

	resultsFiles := files.ParseResultsDirectory(folderName)
	for _, file := range resultsFiles {
		err := streamToRepo(file, r)
		if err != nil {
			return err
		}
	}
	return nil
}

//
// Takes an input stream of xml, converts to json and
// writes json into kv repository (badger)
//
// For SIF objects each is given the key of its RefId
//
// xmlFileName: input file/stream of xml results data
// repo: the repository to write the converted data into
//
func streamToRepo(xmlFileName string, db *repo.BadgerRepo) error {

	// open the xml file
	size, xmlStream, err := files.OpenXMLFile(xmlFileName)
	if err != nil {
		return err
	}

	//
	// superset of data objects we can extract from the
	// stream
	//
	var dataTypes = []string{
		"NAPStudentResponseSet",
		"NAPEventStudentLink",
		"StudentPersonal",
		"NAPTestlet",
		"NAPTestItem",
		"NAPTest",
		"NAPCodeFrame",
		"SchoolInfo",
		"NAPTestScoreSummary",
	}

	// initialise the extractor
	opts := []sec.Option{
		sec.ObjectsToExtract(dataTypes),
		sec.ProgressBar(size),
	}
	sec, err := sec.NewStreamExtractConverter(xmlStream, opts...)
	if err != nil {
		return err
	}
	// iterate the xml stream and save each object to db
	count := 0
	totals := map[string]int{}
	for result := range sec.Stream() {
		r := result
		switch r.Name {
		case "NAPEventStudentLink":
			db.Store(r, repo.IdxEventByStudentAndTest())
		default:
			db.Store(r, repo.IdxSifObjectByTypeAndRefId())
		}
		totals[r.Name]++
		count++
	}
	fmt.Printf("\n\t%d data-objects parsed\n\n", count)
	for k, v := range totals {
		fmt.Printf("\t%s: %d\n", k, v)
	}
	fmt.Println()

	return nil
}
