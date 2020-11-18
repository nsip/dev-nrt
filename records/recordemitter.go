package records

import (
	"errors"
	"fmt"
	"log"

	"github.com/dgraph-io/badger/v2"
	repo "github.com/nsip/dev-nrt/repository"
	"github.com/tidwall/gjson"
)

var (
	ErrMissingStudent  = errors.New("No student found for event")
	ErrMissingSchool   = errors.New("No school found for event")
	ErrMissingTest     = errors.New("No test found for event")
	ErrMissingResponse = errors.New("No student response found for event")
)

//
// convenience function pass any json data blob and the
// gjson path to retrieve the value at that location as a
// string
//
func GetValueString(data []byte, path string) string {
	return gjson.GetBytes(data, path).String()
}

//
// iterates datastore and emits stream of results records.
// event-oriented are one data collection per test event
// student-oriented would be one record per student, but all test results for that student
//
type Emitter struct {
	eorstream chan *EventOrientedRecord
	sorstream chan *StudentOrientedRecord
	repo      *repo.BadgerRepo
}

//
// create a new emitter, accepts data repository via options
//
func NewEmitter(opts ...Option) (*Emitter, error) {

	e := &Emitter{
		eorstream: make(chan *EventOrientedRecord, 256),
		sorstream: make(chan *StudentOrientedRecord, 256),
	}

	// appply all options
	if err := e.setOptions(opts...); err != nil {
		return nil, err
	}

	// double-check a repository has been supplied
	if e.repo == nil {
		return nil, errors.New("Emitter requires a valid repository.")
	}

	return e, nil

}

//
// provides channel iterator for event-oriented records
//
func (e *Emitter) EventBasedStream() chan *EventOrientedRecord {

	go e.emitEventOrientedRecords()

	return e.eorstream

}

func (e *Emitter) emitEventOrientedRecords() {

	defer close(e.eorstream)

	var event, student, school, test, response *badger.Item
	var nesl, sp, si, nt, nsrs []byte
	var txnErr error
	var studentkey, schoolkey, testkey, responsekey string

	//
	// db transaction to iterate events and collect
	// associated result data
	//
	err := e.repo.DB().View(func(txn *badger.Txn) error {
		//
		// iterate the event-links
		//
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		prefix := []byte("NAPEventStudentLink")
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			eor := &EventOrientedRecord{}
			//
			// get the event
			//
			event = it.Item()
			nesl, txnErr = event.ValueCopy(nil)
			if txnErr != nil {
				return txnErr
			}
			eor.NAPEventStudentLink = nesl
			//
			// get the student
			//
			studentkey = fmt.Sprintf("StudentPersonal:%s", eor.StudentPersonalRefId())
			student, txnErr = txn.Get([]byte(studentkey))
			if txnErr != nil {
				if txnErr == badger.ErrKeyNotFound {
					eor.Err = ErrMissingStudent
					e.eorstream <- eor
					continue
				} else {
					return txnErr
				}
			}
			sp, txnErr = student.ValueCopy(nil)
			if txnErr != nil {
				return txnErr
			}
			eor.StudentPersonal = sp
			//
			// get the school
			//
			schoolkey = fmt.Sprintf("SchoolInfo:%s", eor.SchoolInfoRefId())
			school, txnErr = txn.Get([]byte(schoolkey))
			if txnErr != nil {
				if txnErr == badger.ErrKeyNotFound {
					eor.Err = ErrMissingSchool
					e.eorstream <- eor
					continue
				} else {
					return txnErr
				}
			}
			si, txnErr = school.ValueCopy(nil)
			if txnErr != nil {
				return txnErr
			}
			eor.SchoolInfo = si
			//
			// get the test
			//
			testkey = fmt.Sprintf("NAPTest:%s", eor.NAPTestRefId())
			test, txnErr = txn.Get([]byte(testkey))
			if txnErr != nil {
				if txnErr == badger.ErrKeyNotFound {
					eor.Err = ErrMissingTest
					e.eorstream <- eor
					continue
				} else {
					return txnErr
				}
			}
			nt, txnErr = test.ValueCopy(nil)
			if txnErr != nil {
				return txnErr
			}
			eor.NAPTest = nt
			//
			// get the response
			//
			responsekey = fmt.Sprintf("NAPStudentResponseSet:%s:%s", eor.StudentPersonalRefId(), eor.NAPTestRefId())
			response, txnErr = txn.Get([]byte(responsekey))
			if txnErr != nil {
				if txnErr == badger.ErrKeyNotFound {
					eor.Err = ErrMissingResponse
					e.eorstream <- eor
					continue
				} else {
					return txnErr
				}
			}
			nsrs, txnErr = response.ValueCopy(nil)
			if txnErr != nil {
				return txnErr
			}
			eor.NAPStudentResponseSet = nsrs

			e.eorstream <- eor
		}
		return nil
	})

	if err != nil {
		log.Println("Error iterating event-links:", err)
	}

}

//
// provides channel iterator for student-oriented records
//
func (e *Emitter) StudentBasedStream() chan *StudentOrientedRecord {

	// go do streambuilding here

	return e.sorstream
}

type Option func(*Emitter) error

//
// apply all supplied options to the emitter
// returns any error encountered while applying the options
//
func (e *Emitter) setOptions(options ...Option) error {
	for _, opt := range options {
		if err := opt(e); err != nil {
			return err
		}
	}
	return nil
}

//
// Data source for the emitter. Pass in an
// existing/opened repository.
//
func EmitterRepository(repo *repo.BadgerRepo) Option {
	return func(e *Emitter) error {
		e.repo = repo
		return nil
	}
}
