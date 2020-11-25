package nrt

import (
	"fmt"

	"github.com/nsip/dev-nrt/files"
	repo "github.com/nsip/dev-nrt/repository"
	"github.com/nsip/dev-nrt/sec"
)

func IngestResults(folderName string) (map[string]int, error) {

	//
	// create a repo for the data
	//
	r, err := repo.NewBadgerRepo("./kv/")
	if err != nil {
		return nil, err
	}
	defer r.Close()

	//
	// capture stats from each file ingested
	//
	stats := []map[string]int{}

	//
	// parse all results files in folder
	//
	resultsFiles := files.ParseResultsDirectory(folderName)
	for _, file := range resultsFiles {
		fmt.Printf("\nProcessing XML File:\t(%s)\n", file)
		stat, err := streamToRepo(file, r)
		if err != nil {
			return nil, err
		}
		stats = append(stats, stat)
	}
	return cumulativeStats(stats)
}

//
// if multiple files were ingested, accumulate the stats about
// objects stored
//
func cumulativeStats(s []map[string]int) (map[string]int, error) {

	// quick optimisation for single-file case
	if len(s) == 1 {
		return s[0], nil
	}

	cs := map[string]int{}
	for _, stats := range s {
		for k, v := range stats {
			cs[k] = cs[k] + v
		}
	}

	return cs, nil

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
// returns a summary stats map of object-types and their counts
//
func streamToRepo(xmlFileName string, db *repo.BadgerRepo) (map[string]int, error) {

	// open the xml file
	size, xmlStream, err := files.OpenXMLFile(xmlFileName)
	if err != nil {
		return nil, err
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
		return nil, err
	}
	// iterate the xml stream and save each object to db
	count := 0
	totals := map[string]int{}
	for result := range sec.Stream() {
		r := result
		switch r.Name {
		case "NAPStudentResponseSet":
			db.Store(r, repo.IdxByTypeStudentAndTest())
		default:
			db.Store(r, repo.IdxSifObjectByTypeAndRefId())
		}
		totals[r.Name]++
		count++
	}
	fmt.Printf("\n\t%d data-objects parsed\n\n", count)

	return totals, nil
}
