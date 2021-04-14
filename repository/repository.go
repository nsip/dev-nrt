package repository

import (
	"log"
	"os"
	"path/filepath"

	"github.com/dgraph-io/badger/v2"
	jsoniter "github.com/json-iterator/go"
	"github.com/nsip/dev-nrt/sec"
)

//
// Allow definition of query at call-site by
// abstracting to function
//
type BadgerQueryFunc func(txn *badger.Txn) error

//
// keep totals for objects added to db
//
type ObjectStats map[string]int

//
// Data repositoryusing badger kv db engine
//
type BadgerRepo struct {
	db *badger.DB
	wb *badger.WriteBatch
}

//
// accessor for stored object stats in the repo itself
//
const STATS_KEY = "REPO-STATS"

//
// initiaise for marshal/unmarshal json
//
var json = jsoniter.ConfigFastest

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
	opts := badger.DefaultOptions(dbFolderName)
	opts = opts.WithLoggingLevel(badger.WARNING)
	// db, err := badger.Open(badger.DefaultOptions(dbFolderName))
	db, err := badger.Open(opts)
	if err != nil {
		return nil, err
	}
	// create a (fast) writebatch on the db
	wb := db.NewWriteBatch()

	return &BadgerRepo{db: db, wb: wb}, nil

}

//
// open a repo that's already got data in it
//
func OpenExistingBadgerRepo(dbFolderName string) (*BadgerRepo, error) {

	// create new badger instance
	opts := badger.DefaultOptions(dbFolderName)
	opts = opts.WithLoggingLevel(badger.WARNING)
	// db, err := badger.Open(badger.DefaultOptions(dbFolderName))
	db, err := badger.Open(opts)
	if err != nil {
		return nil, err
	}
	// create a (fast) writebatch on the db
	wb := db.NewWriteBatch()

	return &BadgerRepo{db: db, wb: wb}, nil

}

//
// access to underlying data store for iterator queries
//
func (br *BadgerRepo) DB() *badger.DB {
	return br.db
}

//
// ensure all writes to disk are completed
//
func (br *BadgerRepo) Close() {
	br.wb.Flush()
	br.db.Close()
}

//
// on small data files write-batch is so fast it doesn't
// commit prior to reporting starting, this method can be used
// to force a flush of the current write-batch
//
func (br *BadgerRepo) Commit() {
	br.wb.Flush()
	br.wb = br.db.NewWriteBatch()
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

	// add to the writebatch to store
	return br.wb.Set(key, r.Json)
}

//
// persists a copy of the recorded object counts back into
// the repository, particularly used for sizing progress
// bars etc. when reports are run against the repo with
// no prior ingest phase.
//
func (br *BadgerRepo) SaveStats(s ObjectStats) error {

	//
	// convert to json
	//
	jsonStats, err := json.Marshal(&s)
	if err != nil {
		return err
	}

	//
	// store in db
	//
	err = br.db.Update(func(txn *badger.Txn) error {
		err := txn.Set([]byte(STATS_KEY), jsonStats)
		return err
	})

	return err

}

//
// returns the last reocrded set of ingest object counts
// for this repo
//
func (br *BadgerRepo) GetStats() ObjectStats {
	var statsBytes []byte
	err := br.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(STATS_KEY))
		if err != nil {
			return err
		}

		statsBytes, err = item.ValueCopy(nil)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		// no current recorded stats
		return ObjectStats{}
	}

	var objst ObjectStats
	err = json.Unmarshal(statsBytes, &objst)
	if err != nil {
		log.Println("error unmarshalling stats from db:", err)
		return ObjectStats{}
	}

	return objst

}
