package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	badger "github.com/dgraph-io/badger/v2"
	"github.com/nsip/dev-nrt/files"
	"github.com/nsip/dev-nrt/repo"
	"github.com/nsip/dev-nrt/sec"
)

func main() {

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

	fileName := "../../../testdata/n2sif.xml"
	// fileName := "../../../testdata/rrd.xml"

	err := StreamToKVStore(fileName, "./kv/", repo.IdxSifObjectByTypeAndRefId(), dataTypes...)
	if err != nil {
		log.Println("error converting xml file:", err)
	}
	fmt.Println("--- Storage to kv db complete.")

}

//
// Takes an input stream of xml, converts to json and
// writes json into kv datastore (badger)
//
// For SIF objects each is given the key of its RefId
//
// xmlFileName: input file/stream of xml results data
// dbFolderName: the directory to create the datastore in
// idxf: index functio to use to generate keys for these data objects in the k/v store
// dataObjects: the data types to extract from the stream (e.g. StudentPersonal, SchoolInfo etc.)
//
func StreamToKVStore(xmlFileName string, dbFolderName string, idxf repo.IndexFunc, dataObjects ...string) error {

	// open the xml file
	size, xmlStream, err := files.OpenXMLFile(xmlFileName)
	if err != nil {
		return err
	}

	// remove any existing dbs
	err = os.RemoveAll(filepath.Dir(dbFolderName))
	if err != nil {
		return err
	}
	// recreate the working directory
	err = os.MkdirAll(filepath.Dir(dbFolderName), os.ModePerm)
	if err != nil {
		return err
	}

	// create new badger instance
	db, err := badger.Open(badger.DefaultOptions(dbFolderName))
	if err != nil {
		return err
	}
	defer db.Close()
	// create a (fast) writebatch on the db
	wb := db.NewWriteBatch()
	defer wb.Cancel()

	// initialise the extractor
	opts := []sec.Option{
		sec.ObjectsToExtract(dataObjects),
		sec.ProgressBar(size),
	}
	sec, err := sec.NewStreamExtractConverter(xmlStream, opts...)
	if err != nil {
		return err
	}
	// iterate the xml stream and save each object to db
	count := 0
	for result := range sec.Stream() {

		key, err := idxf(result)
		if err != nil {
			return err
		}
		err = wb.Set(key, result.Json)
		if err != nil {
			return err
		}
		count++
	}
	wb.Flush()
	log.Printf("%d data-objects parsed\n\n", count)

	return nil
}
