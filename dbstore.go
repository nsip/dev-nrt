package nrt

import (
	"errors"
	"io"
	"log"
	"os"
	"path/filepath"

	badger "github.com/dgraph-io/badger/v2"
	"github.com/tidwall/gjson"
)

//
// signature for an indexing function to
// use on the data objects;
// takes in a json blob, returns the key for that
// blob as bytes.
//
type IndexFunc func([]byte) ([]byte, error)

//
// Takes an input stream of xml, converts to json and
// writes json into kv datastore (badger)
//
// For SIF objects each is given the key of its RefId
//
// xmlStream: input file/stream of xml results data
// dbFolderName: the directory to create the datastore in
// idxf: index functio to use to generate keys for these data objects in the k/v store
// dataObjects: the data types to extract from the stream (e.g. StudentPersonal, SchoolInfo etc.)
//
func StreamToKVStore(xmlStream io.Reader, dbFolderName string, idxf IndexFunc, dataObjects ...string) error {

	// remove any existing dbs
	err := os.RemoveAll(filepath.Dir(dbFolderName))
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

	// iterate the xml stream and save each object to db
	sec := NewStreamExtractConverter(xmlStream, dataObjects...)
	count := 0
	for jsonBytes := range sec.Stream() {

		key, err := idxf(jsonBytes)
		if err != nil {
			return err
		}
		err = wb.Set(key, jsonBytes)
		if err != nil {
			return err
		}
		count++
	}
	wb.Flush()
	log.Printf("%d data-objects parsed\n", count)

	return nil
}

//
// index func to retrieve sif object refid
// only index needed for majority of the objects
//
func IdxSifObjectByRefId() IndexFunc {

	return func(json []byte) ([]byte, error) {
		refid := gjson.GetBytes(json, "*.RefId")
		if !refid.Exists() {
			return nil, errors.New("could not find RefId")
		}
		return []byte(refid.String()), nil
	}

}
