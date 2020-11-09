package repo

import (
	"os"
	"path/filepath"

	"github.com/dgraph-io/badger/v2"
	"github.com/nsip/dev-nrt/sec"
)

//
// Allow definition of query at call-site by
// abstraxcting to function
//
type BadgerQueryFunc func(txn *badger.Txn) error

//
// Data repositoryusing badger kv db engine
//
type BadgerRepo struct {
	db *badger.DB
	wb *badger.WriteBatch
}

//
// create a new badger repo in the nominated directory.
//
func NewBadgerRepo(dbFolderName string) (*BadgerRepo, error) {
	// remove any existing dbs
	err := os.RemoveAll(filepath.Dir(dbFolderName))
	if err != nil {
		return nil, err
	}
	// recreate the working directory
	err = os.MkdirAll(filepath.Dir(dbFolderName), os.ModePerm)
	if err != nil {
		return nil, err
	}

	// create new badger instance
	db, err := badger.Open(badger.DefaultOptions(dbFolderName))
	if err != nil {
		return nil, err
	}
	// create a (fast) writebatch on the db
	wb := db.NewWriteBatch()

	return &BadgerRepo{db: db, wb: wb}, nil

}

//
// ensure all writes to disk are completed
//
func (br *BadgerRepo) Close() {
	br.wb.Flush()
	br.db.Close()
}

//
// saves data into the badger db, uses the indexfunc to
// create the lookup key for the data item
//
func (br *BadgerRepo) Store(r sec.Result, idxf IndexFunc) error {

	// generate the key
	key, err := idxf(r)
	if err != nil {
		return err
	}
	// add to the writebatch
	err = br.wb.Set(key, r.Json)
	if err != nil {
		return err
	}

	return nil
}

//
// pass in a query to retrieve data from the repository
// returns an arrray of byte-array records containing the json
// objects returned from the db
//
func (br *BadgerRepo) Fetch(qf BadgerQueryFunc) ([][]byte, error) {
	return nil, nil
}
