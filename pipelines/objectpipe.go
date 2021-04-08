package pipelines

import "github.com/nsip/dev-nrt/records"

//
// interface to be implemented by any report/processor
// working with stream of Object records
//
type ObjectPipe interface {
	ProcessObjectRecords(in chan *records.ObjectRecord) chan *records.ObjectRecord
}

//
// pipeline to attach multiple reports to the
// Object stream
//
type ObjectPipeline struct {
	head chan *records.ObjectRecord
	tail chan *records.ObjectRecord
}

//
// construct a linked pipeline of reports that process
// Object records
//
func NewObjectPipeline(pipes ...ObjectPipe) ObjectPipeline {
	head := make(chan *records.ObjectRecord)
	var next_chan chan *records.ObjectRecord
	for _, pipe := range pipes {
		if next_chan == nil {
			next_chan = pipe.ProcessObjectRecords(head)
		} else {
			next_chan = pipe.ProcessObjectRecords(next_chan)
		}
	}
	return ObjectPipeline{head: head, tail: next_chan}
}

//
// allows items to be injected into the pipeline from a source such
// as a reports.Emitter{}
//
func (p *ObjectPipeline) Enqueue(item *records.ObjectRecord) {
	p.head <- item
}

//
// allows the attachment of a terminating reader at the end of the pipeline
// such as progress-bar update or audit log.
//
func (p *ObjectPipeline) Dequeue(handler func(*records.ObjectRecord)) {
	for i := range p.tail {
		handler(i)
	}
}

//
// closes the pipeline and cascades termination to all
// individual reports
//
func (p *ObjectPipeline) Close() {
	close(p.head)
}
