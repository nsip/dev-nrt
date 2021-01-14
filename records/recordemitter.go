package records

import (
	"errors"
	"fmt"
	"log"

	"github.com/dgraph-io/badger/v2"
	repo "github.com/nsip/dev-nrt/repository"
)

var (
	// event errors
	ErrMissingStudent  = errors.New("No student found for event")
	ErrMissingSchool   = errors.New("No school found for event")
	ErrMissingTest     = errors.New("No test found for event")
	ErrMissingResponse = errors.New("No student response found for event")
	// student errors
	ErrMissingEvent   = errors.New("No events found for student")
	ErrMissingSummary = errors.New("No test summary found for student")
)

//
// iterates datastore and emits stream of results records.
// event-oriented are one data collection per test event
// student-oriented would be one record per student, but all test results for that student
//
type Emitter struct {
	eorstream chan *EventOrientedRecord
	sorstream chan *StudentOrientedRecord
	cfstream  chan *CodeframeRecord
	objstream chan *ObjectRecord
	repo      *repo.BadgerRepo
}

//
// create a new emitter, accepts data repository via options
//
func NewEmitter(opts ...Option) (*Emitter, error) {

	e := &Emitter{
		eorstream: make(chan *EventOrientedRecord, 256),
		sorstream: make(chan *StudentOrientedRecord, 256),
		cfstream:  make(chan *CodeframeRecord, 256),
		objstream: make(chan *ObjectRecord, 256),
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
// provides channel iterator for student-oriented records,
// each record has; student, school, all tests seen by student, all responses
// and all events in one record.
//
func (e *Emitter) StudentBasedStream() chan *StudentOrientedRecord {

	go e.emitStudentOrientedRecords()

	return e.sorstream
}

//
// iterates the db and constructs the student-oriented records
//
func (e *Emitter) emitStudentOrientedRecords() {

	defer close(e.sorstream)

	var event, student, school, test, response, summary *badger.Item
	var nesl, sp, si, nt, nsrs, ntss []byte
	var txnErr error
	var schoolkey, testkey, responsekey, summarykey string

	//
	// logically need to go, student, events, school, responses, tests
	//

	//
	// db transaction to iterate events and collect
	// associated result data
	//
	err := e.repo.DB().View(func(txn *badger.Txn) error {
		//
		// outer iteration is by the student records
		//
		itStudents := txn.NewIterator(badger.DefaultIteratorOptions)
		defer itStudents.Close()
		//
		// inner iterator to find all events associated with student
		//
		itEvents := txn.NewIterator(badger.DefaultIteratorOptions)
		defer itEvents.Close()

		prefix := []byte("StudentPersonal")
		for itStudents.Seek(prefix); itStudents.ValidForPrefix(prefix); itStudents.Next() {
			sor := NewStudentOrientedRecord()
			//
			// get the student
			//
			student = itStudents.Item()
			sp, txnErr = student.ValueCopy(nil)
			if txnErr != nil {
				return txnErr
			}
			sor.StudentPersonal = sp
			sor.HasStudentPersonal = true
			//
			// get the events
			//
			evtPrefix := []byte(fmt.Sprintf("NAPEventStudentLink:%s", sor.StudentPersonalRefId()))
			for itEvents.Seek(evtPrefix); itEvents.ValidForPrefix(evtPrefix); itEvents.Next() {
				event = itEvents.Item()
				nesl, txnErr = event.ValueCopy(nil)
				if txnErr != nil {
					sor.Err = ErrMissingEvent
					continue
				}
				sor.AddEvent(nesl)
			}
			//
			// get the school
			//
			schoolkey = fmt.Sprintf("SchoolInfo:%s", sor.SchoolInfoRefId())
			school, txnErr = txn.Get([]byte(schoolkey))
			if txnErr != nil {
				if txnErr == badger.ErrKeyNotFound {
					sor.Err = ErrMissingSchool
					e.sorstream <- sor
					continue
				} else {
					return txnErr
				}
			}
			si, txnErr = school.ValueCopy(nil)
			if txnErr != nil {
				return txnErr
			}
			sor.SchoolInfo = si
			sor.HasSchoolInfo = true
			//
			// get the tests
			//
			for _, testid := range sor.GetNAPTestRefIds() {
				testkey = fmt.Sprintf("NAPTest:%s", testid)
				test, txnErr = txn.Get([]byte(testkey))
				if txnErr != nil {
					if txnErr == badger.ErrKeyNotFound {
						sor.Err = ErrMissingTest
						continue
					} else {
						return txnErr
					}
				}
				nt, txnErr = test.ValueCopy(nil)
				if txnErr != nil {
					return txnErr
				}
				sor.AddTest(nt)
			}
			//
			// get the responses
			//
			for _, testid := range sor.GetNAPTestRefIds() {
				responsekey = fmt.Sprintf("NAPStudentResponseSet:%s:%s", sor.StudentPersonalRefId(), testid)
				response, txnErr = txn.Get([]byte(responsekey))
				if txnErr != nil {
					if txnErr == badger.ErrKeyNotFound {
						sor.Err = ErrMissingResponse
						continue
					} else {
						return txnErr
					}
				}
				nsrs, txnErr = response.ValueCopy(nil)
				if txnErr != nil {
					return txnErr
				}
				sor.AddResponse(nsrs)
			}
			//
			// get the score summaries
			//
			for _, testid := range sor.GetNAPTestRefIds() {
				summarykey = fmt.Sprintf("NAPTestScoreSummary:%s:%s", sor.SchoolInfoRefId(), testid)
				summary, txnErr = txn.Get([]byte(summarykey))
				if txnErr != nil {
					if txnErr == badger.ErrKeyNotFound {
						sor.Err = ErrMissingSummary
						continue
					} else {
						return txnErr
					}
				}
				ntss, txnErr = summary.ValueCopy(nil)
				if txnErr != nil {
					return txnErr
				}
				sor.AddScoreSummary(ntss)
			}

			e.sorstream <- sor
		}
		return nil
	})

	if err != nil {
		log.Println("Error iterating student records:", err)
	}

}

//
// provides channel iterator for event-oriented records
// each record contains; student, school, test, event and response
// information
//
func (e *Emitter) EventBasedStream() chan *EventOrientedRecord {

	go e.emitEventOrientedRecords()

	return e.eorstream

}

//
// iterates the db and constructs the event-oriented records
//
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
			eor := NewEventOrientedRecord()
			//
			// get the event
			//
			event = it.Item()
			nesl, txnErr = event.ValueCopy(nil)
			if txnErr != nil {
				return txnErr
			}
			eor.NAPEventStudentLink = nesl
			eor.HasNAPEventStudentLink = true
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
			eor.HasStudentPersonal = true
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
			eor.HasSchoolInfo = true
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
			eor.HasNAPTest = true
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
			eor.HasNAPStudentResponseSet = true

			e.eorstream <- eor
		}
		return nil
	})

	if err != nil {
		log.Println("Error iterating event-links:", err)
	}

}

//
// provides channel iterator for event-oriented records
//
func (e *Emitter) CodeframeStream() chan *CodeframeRecord {

	go e.emitCodeframeRecords()

	return e.cfstream

}

//
// iterates and exports the naplan types that make up the
// codeframe - Test, Testlets and Items
//
func (e *Emitter) emitCodeframeRecords() {

	defer close(e.cfstream)

	cfObjects := []string{"NAPTest", "NAPTestlet", "NAPTestItem", "NAPCodeFrame"}

	var it *badger.Iterator
	var txnErr error
	var jsonBytes []byte

	err := e.repo.DB().View(func(txn *badger.Txn) error {
		it = txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		for _, objType := range cfObjects {
			prefix := []byte(objType + ":")
			for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
				jsonBytes, txnErr = it.Item().ValueCopy(nil)
				if txnErr != nil {
					return txnErr
				}
				cfr := CodeframeRecord{RecordType: objType, Json: jsonBytes}
				e.cfstream <- &cfr
			}
		}
		return nil
	})

	if err != nil {
		log.Println("error iterating codeframe objects:", err)
	}

}

//
// provides channel iterator for simple object records
//
func (e *Emitter) ObjectStream() chan *ObjectRecord {

	go e.emitObjectRecords()

	return e.objstream

}

//
// iterates and exports simple types
// objects that are just directly transformed into csv with no
// interpretation, business logic or record joins
//
func (e *Emitter) emitObjectRecords() {

	defer close(e.objstream)

	// list determiined by needs of current reports, can be extended to any data objects
	orObjects := []string{"SchoolInfo", "StudentPersonal", "NAPTestScoreSummary"}

	var it *badger.Iterator
	var txnErr error
	var jsonBytes []byte

	err := e.repo.DB().View(func(txn *badger.Txn) error {
		it = txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		for _, objType := range orObjects {
			prefix := []byte(objType + ":")
			for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
				jsonBytes, txnErr = it.Item().ValueCopy(nil)
				if txnErr != nil {
					return txnErr
				}
				or := ObjectRecord{RecordType: objType, Json: jsonBytes}
				e.objstream <- &or
			}
		}
		return nil
	})

	if err != nil {
		log.Println("error iterating data objects:", err)
	}

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
