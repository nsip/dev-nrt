package pipelines

import "github.com/nsip/dev-nrt/records"

//
// interface to be implemented by any report/processor
// working with stream of codeframe records
//
type CodeframePipe interface {
	ProcessCodeframeRecords(in chan *records.CodeframeRecord) chan *records.CodeframeRecord
}

//
// pipeline to attach multiple reports to the
// codeframe stream
//
type CodeframePipeline struct {
	head chan *records.CodeframeRecord
	tail chan *records.CodeframeRecord
}

//
// construct a linked pipeline of reports that process
// codeframe records
//
func NewCodeframePipeline(pipes ...CodeframePipe) CodeframePipeline {
	head := make(chan *records.CodeframeRecord)
	var next_chan chan *records.CodeframeRecord
	for _, pipe := range pipes {
		if next_chan == nil {
			next_chan = pipe.ProcessCodeframeRecords(head)
		} else {
			next_chan = pipe.ProcessCodeframeRecords(next_chan)
		}
	}
	return CodeframePipeline{head: head, tail: next_chan}
}

//
// allows items to be injected into the pipeline from a source such
// as a reports.Emitter{}
//
func (p *CodeframePipeline) Enqueue(item *records.CodeframeRecord) {
	p.head <- item
}

//
// allows the attachment of a terminating reader at the end of the pipeline
// such as progress-bar update or audit log.
//
func (p *CodeframePipeline) Dequeue(handler func(*records.CodeframeRecord)) {
	for i := range p.tail {
		handler(i)
	}
}

//
// closes the pipeline and cascades termination to all
// individual reports
//
func (p *CodeframePipeline) Close() {
	close(p.head)
}
