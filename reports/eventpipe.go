package reports

import "github.com/nsip/dev-nrt/records"

//
// interface to be implemented by any report
// working with stream of event records
//
type EventPipe interface {
	ProcessEventRecords(in chan *records.EventOrientedRecord) chan *records.EventOrientedRecord
}

//
// pipleline to attach multiple reports to the
// event stream
//
type EventPipeline struct {
	head chan *records.EventOrientedRecord
	tail chan *records.EventOrientedRecord
}

//
// construct a linked pipeline of reports that process
// event-based records
//
func NewEventPipeline(pipes ...EventPipe) EventPipeline {
	head := make(chan *records.EventOrientedRecord)
	var next_chan chan *records.EventOrientedRecord
	for _, pipe := range pipes {
		if next_chan == nil {
			next_chan = pipe.ProcessEventRecords(head)
		} else {
			next_chan = pipe.ProcessEventRecords(next_chan)
		}
	}
	return EventPipeline{head: head, tail: next_chan}
}

//
// allows items to be injected into the pipeline from a source such
// as a reports.Emitter{}
//
func (p *EventPipeline) Enqueue(item *records.EventOrientedRecord) {
	p.head <- item
}

//
// allows the attachment of a terminating reader at the end of the pipeline
// such as progress-bar update or audit log.
//
func (p *EventPipeline) Dequeue(handler func(*records.EventOrientedRecord)) {
	for i := range p.tail {
		handler(i)
	}
}

//
// closes the pipeline and cascades termination to all
// individual reports
//
func (p *EventPipeline) Close() {
	close(p.head)
}
