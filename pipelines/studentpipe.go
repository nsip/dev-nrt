package pipelines

import "github.com/nsip/dev-nrt/records"

//
// interface to be implemented by any report/processor
// working with stream of Object records
//
type StudentPipe interface {
	ProcessStudentOrientedRecords(in chan *records.StudentOrientedRecord) chan *records.StudentOrientedRecord
}

//
// pipleline to attach multiple reports to the
// Object stream
//
type StudentPipeline struct {
	head chan *records.StudentOrientedRecord
	tail chan *records.StudentOrientedRecord
}

//
// construct a linked pipeline of reports that process
// Object records
//
func NewStudentPipeline(pipes ...StudentPipe) StudentPipeline {
	head := make(chan *records.StudentOrientedRecord)
	var next_chan chan *records.StudentOrientedRecord
	for _, pipe := range pipes {
		if next_chan == nil {
			next_chan = pipe.ProcessStudentOrientedRecords(head)
		} else {
			next_chan = pipe.ProcessStudentOrientedRecords(next_chan)
		}
	}
	return StudentPipeline{head: head, tail: next_chan}
}

//
// allows items to be injected into the pipeline from a source such
// as a reports.Emitter{}
//
func (p *StudentPipeline) Enqueue(item *records.StudentOrientedRecord) {
	p.head <- item
}

//
// allows the attachment of a terminating reader at the end of the pipeline
// such as progress-bar update or audit log.
//
func (p *StudentPipeline) Dequeue(handler func(*records.StudentOrientedRecord)) {
	for i := range p.tail {
		handler(i)
	}
}

//
// closes the pipeline and cascades termination to all
// individual reports
//
func (p *StudentPipeline) Close() {
	close(p.head)
}
