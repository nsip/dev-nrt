package nrt

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gosuri/uiprogress"
	"github.com/gosuri/uiprogress/util/strutil"
	"github.com/nsip/dev-nrt/pipelines"
	"github.com/nsip/dev-nrt/records"
	"github.com/nsip/dev-nrt/reports"
	"github.com/nsip/dev-nrt/utils"
	"golang.org/x/sync/errgroup"
)

//
// extracts streams of results from the repository
// and feeds them through pipelines of reports
//
func (tr *Transformer) streamResults() error {

	defer utils.TimeTrack(time.Now(), "Report processing")

	if tr.qaReports {
		fmt.Printf("\n\n--- Running Reports (+QA reports):\n\n")
	} else {
		fmt.Printf("\n\n--- Running Reports (-QA reports):\n\n")
	}
	if tr.showProgress {
		tr.uip.Start()
	}

	//
	// start the different processing pipelines concurrently
	// under control of a group
	//
	g, _ := errgroup.WithContext(context.Background())

	// in core
	if tr.coreReports {
		g.Go(tr.simpleObjectReports)
		g.Go(tr.studentReports)
		g.Go(tr.codeframeReports)
		g.Go(tr.eventReports)
	}

	// item
	if tr.itemLevelReports {
		g.Go(tr.studentItemReports)
		g.Go(tr.eventItemReports)
	}

	// writing extract
	if tr.wxReports {
		g.Go(tr.writingExtractReports)
	}

	// xml extract
	if tr.xmlReports {
		g.Go(tr.xmlExtractReports)
	}

	// wait for report streams to end
	if g.Wait() != nil {
		return g.Wait()
	}

	//
	// allow time for flush of large csv files to complete
	//
	time.Sleep(time.Millisecond * 1200)

	// we have a separate wait group just for xml reports, because of how time consuming they are
	if tr.xmlReports {
		//tr.xmlWaitGroup.Wait()
	}

	if tr.showProgress {
		tr.uip.Stop()
		fmt.Println()
	}

	return nil

}

//
// Student-oriented reports processor - heavy item level reports
// each record contains all test responses and events
// for a given student
//
func (tr *Transformer) studentItemReports() error {

	// create a record emitter
	em, err := records.NewEmitter(records.EmitterRepository(tr.repository))
	if err != nil {
		return err
	}
	// create the object report pipeline
	pl := pipelines.NewStudentPipeline(
		//
		// pre-processors
		//
		reports.StudentRecordSplitterBlockReport(),
		reports.DomainParticipationReport(),
		reports.DomainResponsesScoresReport(),
		reports.DomainDACReport(tr.helper),
		reports.DomainItemResponsesReport("Reading"),
		reports.DomainItemResponsesReport("Spelling"),
		reports.DomainItemResponsesReport("GrammarAndPunctuation"),
		reports.DomainItemResponsesReport("Numeracy"),
		reports.DomainItemResponsesReport("Writing"),
		reports.DomainItemResponsesWritingRubricsReport(), // gives further writing breakdown by subscore
		//
		// reports
		//
		reports.NswAllPearsonY3Report(),
		reports.NswAllPearsonY5Report(),
		reports.NswAllPearsonY7Report(),
		reports.NswAllPearsonY9Report(),
		//
		reports.PrintAllReport(),
	)
	// create a progress bar
	bar := tr.uip.AddBar(tr.stats["StudentPersonal"])
	bar.AppendCompleted().PrependElapsed()
	bar.PrependFunc(func(b *uiprogress.Bar) string {
		return strutil.Resize(" Item-level (student-based):", 35)
	})

	//
	// register an output handler for pipeline, used for progress-bar
	// but could also be audit sink, backup of processed records etc.
	//
	// NOTE: must be handler here even with empty body
	// otherwise exit channel blocks for pipeline
	//
	go pl.Dequeue(func(sor *records.StudentOrientedRecord) {
		// easy win no-op, also reclaims memory
		// sor = nil
		bar.Incr()
	})
	defer pl.Close()
	//
	// now iterate the object records, passing them through
	// the processing pipeline
	//
	for sor := range em.StudentBasedStream() {
		pl.Enqueue(sor)
	}

	return nil
}

// Simple object reports processor
// takes data objects and converts to csv
// with minimal business logic
func (tr *Transformer) simpleObjectReports() error {

	// create a record emitter
	em, err := records.NewEmitter(records.EmitterRepository(tr.repository))
	if err != nil {
		return err
	}

	// create the pipeline members
	// regular reports
	rpt := []pipelines.ObjectPipe{
		reports.SystemObjectsCountReport(),
		reports.QcaaNapoSchoolsReport(),
		reports.QcaaNapoStudentsReport(),
		reports.QcaaNapoItemsReport(),
		reports.NswItemDescriptorsReport(),
		reports.QcaaTestScoreSummaryReport(),
		reports.SystemSchoolsReport(),
		reports.SystemScoreSummariesReport(),
		reports.QldStudentReport(),
	}
	// include qa reports if requested
	if tr.qaReports {
		qa := []pipelines.ObjectPipe{
			reports.SystemExtraneousCharactersStudentsReport(), // qa
			reports.QaSystemScoreSummariesReport(tr.helper),    // qa
			reports.OrphanScoreSummariesReport(),               // qa
			reports.QaGuidCheckReport(tr.objecthelper),         // qa
			reports.QaCodeframeCheckReport(tr.objecthelper),    // qa
		}
		rpt = append(rpt, qa...)
	}

	// create the object report pipeline
	objpl := pipelines.NewObjectPipeline(rpt...)

	// create a progress bar
	barsize := tr.stats["SchoolInfo"] + tr.stats["StudentPersonal"] + tr.stats["NAPTestScoreSummary"]
	bar := tr.uip.AddBar(barsize)
	bar.AppendCompleted().PrependElapsed()
	bar.PrependFunc(func(b *uiprogress.Bar) string {
		return strutil.Resize(" Simple reports:", 35)
	})

	//
	// register an output handler for pipeline, used for progress-bar
	// but could also be audit sink, backup of processed records etc.
	//
	// NOTE: must be handler here even with empty body
	// otherwise exit channel blocks for pipeline
	//
	go objpl.Dequeue(func(cfr *records.ObjectRecord) {
		// easy win no-op, also reclaims memory
		// cfr = nil
		bar.Incr()
	})
	defer objpl.Close()
	//
	// now iterate the object records, passing them through
	// the processing pipeline
	//
	for or := range em.ObjectStream() {
		objpl.Enqueue(or)
	}

	return nil

}

//
// Student-oriented reports processor
// each record contains all test responses and events
// for a given student
//
func (tr *Transformer) studentReports() error {

	// create a record emitter
	em, err := records.NewEmitter(records.EmitterRepository(tr.repository))
	if err != nil {
		return err
	}

	// create report processor sequences
	// reports & processors are deliberately sequenced for greatest efficiency
	rpt := []pipelines.StudentPipe{
		// pre-processors
		//
		reports.StudentRecordSplitterBlockReport(),
		reports.DomainParticipationReport(),
		reports.DomainResponsesScoresReport(),
		// reports
		//
		reports.SystemParticipationReport(),
		reports.IsrPrintingReport(),
	}
	// insert qa report if reuested
	if tr.qaReports {
		rpt = append(rpt, reports.SystemObjectFrequencyReport())
	}
	// construct second half of processor chain
	rpt2 := []pipelines.StudentPipe{
		// pre-processors
		//
		reports.DomainDACReport(tr.helper),
		// reports
		//
		reports.IsrPrintingExpandedReport(),
		reports.NswPrintReport(),
		reports.StudentProficiencyReport(),
	}
	// create single processor list, in desired order
	rpt = append(rpt, rpt2...)

	// create the object report pipeline
	pl := pipelines.NewStudentPipeline(rpt...)

	// create a progress bar
	bar := tr.uip.AddBar(tr.stats["StudentPersonal"])
	bar.AppendCompleted().PrependElapsed()
	bar.PrependFunc(func(b *uiprogress.Bar) string {
		return strutil.Resize(" Student-based reports:", 35)
	})

	//
	// register an output handler for pipeline, used for progress-bar
	// but could also be audit sink, backup of processed records etc.
	//
	// NOTE: must be handler here even with empty body
	// otherwise exit channel blocks for pipeline
	//
	go pl.Dequeue(func(sor *records.StudentOrientedRecord) {
		// easy win no-op, also reclaims memory
		// sor = nil
		bar.Incr()
	})
	defer pl.Close()
	//
	// now iterate the object records, passing them through
	// the processing pipeline
	//
	for sor := range em.StudentBasedStream() {
		pl.Enqueue(sor)
	}

	return nil

}

//
// CodeFrame reports processor
// reports that use just the objects from the codeframe
// test, teslet, item etc.
//
func (tr *Transformer) codeframeReports() error {

	// create a record emitter
	em, err := records.NewEmitter(records.EmitterRepository(tr.repository))
	if err != nil {
		return err
	}

	// create report processor sequences
	// reports & processors are deliberately sequenced for greatest efficiency
	rpt := []pipelines.CodeframePipe{
		//
		// report
		//
		reports.QcaaNapoTestletsReport(tr.helper),
		//
		// pre-processor
		// remaining codeframe reports need htis item-test-link multiplexer to
		// come first in the pipeline sequence
		reports.ItemTestLinkReport(tr.helper),
		//
		// reports
		//
		reports.QcaaNapoTestsReport(),
		reports.QcaaNapoTestletItemsReport(),
		reports.SystemCodeframeReport(tr.helper),
	}
	// add qa reports if requested
	if tr.qaReports {
		rpt = append(rpt, reports.QaSystemCodeframeReport(tr.helper))
	}
	// create second half of processor sequence
	rpt2 := []pipelines.CodeframePipe{
		//
		// report
		//
		reports.QldTestDataReport(tr.helper),
		//
		// keep this pair at end of pipeline to minimize data expansion
		// pre-processor
		//
		reports.ItemRubricExtractorReport(),
		//
		// report
		//
		reports.QcaaNapoWritingRubricReport(),
	}
	// create single processor list, in desired order
	rpt = append(rpt, rpt2...)

	// create the codeframe report pipeline
	cfpl := pipelines.NewCodeframePipeline(rpt...)

	// create a progress bar
	barSize := tr.stats["NAPCodeFrame"] + tr.stats["NAPTest"] + tr.stats["NAPTestlet"] + tr.stats["NAPTestItem"]
	codeframeBar := tr.uip.AddBar(barSize)
	codeframeBar.AppendCompleted().PrependElapsed()
	codeframeBar.PrependFunc(func(b *uiprogress.Bar) string {
		return strutil.Resize(" Codeframe reports:", 35)
	})

	//
	// register an output handler for pipeline, used for progress-bar
	// but could also be audit sink, backup of processed records etc.
	//
	// NOTE: must be handler here even with empty body
	// otherwise exit channel blocks for pipeline
	//
	go cfpl.Dequeue(func(cfr *records.CodeframeRecord) {
		// easy win no-op, also reclaims memory
		// cfr = nil
		codeframeBar.Incr()
	})
	defer cfpl.Close()
	//
	// now iterate the codeframe records, passing them through
	// the processing pipeline
	//
	for cfr := range em.CodeframeStream() {
		cfpl.Enqueue(cfr)
	}

	return nil

}

//
//
// Item-level reports processor
// Special case of event-oriented record processor
// produces huge number of rows, one for every item seen
// by any student in any test
//
func (tr *Transformer) eventItemReports() error {

	// create a record emitter
	em, err := records.NewEmitter(records.EmitterRepository(tr.repository))
	if err != nil {
		return err
	}
	// create the item reporting pipeline
	pl := pipelines.NewEventPipeline(
		//
		// pre-processors to set up reports, need to be kept in this order
		//
		reports.EventRecordSplitterBlockReport(),
		reports.ItemResponseExtractorReport(),
		reports.ItemDetailReport(tr.helper),
		//
		// actual reports
		//
		reports.SystemItemCountsReport(tr.helper),
		reports.NswItemPrintingReport(),
		reports.QcaaNapoStudentResponsesReport(),
		reports.ItemPrintingReport(),
	)
	// create a progress bar
	bar := tr.uip.AddBar(tr.stats["NAPEventStudentLink"])
	bar.AppendCompleted().PrependElapsed()
	bar.PrependFunc(func(b *uiprogress.Bar) string {
		return strutil.Resize(" Item-level (event-based):", 35)
	})
	//
	// register an output handler for pipeline, used for progress-bar
	// but could also be audit sink, backup of processed records etc.
	//
	// NOTE: must be handler here even with empty body
	// otherwise exit channel blocks for pipeline
	//
	var procsize uint64
	go pl.Dequeue(func(eor *records.EventOrientedRecord) {
		// easy win no-op, also reclaims memory
		// eor = nil
		atomic.AddUint64(&procsize, 1)
		bar.Set(int(procsize))
		bar.Incr() // update the progress bar
	})
	defer pl.Close()
	//
	// now iterate the codeframe records, passing them through
	// the processing pipeline
	//
	for eor := range em.EventBasedStream() {
		pl.Enqueue(eor)
	}

	return nil

}

//
// Event-based report processor
// each record is the test, school, student, event & response
//
func (tr *Transformer) eventReports() error {

	// create a record emitter
	em, err := records.NewEmitter(records.EmitterRepository(tr.repository))
	if err != nil {
		return err
	}

	rpt := []pipelines.EventPipe{
		reports.EventRecordSplitterBlockReport(),
		reports.ActSystemDomainScoresReport(),
		reports.QldStudentScoreReport(),
		reports.SystemDomainScoresReport(),
		reports.CompareRRDtestsReport(),
		reports.SaHomeschooledTestsReport(),
		reports.CompareItemWritingReport(),
		reports.NswWritingPearsonY3Report(tr.helper),
		reports.NswWritingPearsonY5Report(tr.helper),
		reports.NswWritingPearsonY7Report(tr.helper),
		reports.NswWritingPearsonY9Report(tr.helper),
		reports.SystemPNPEventsReport(),
		reports.QcaaNapoEventStudentLinkReport(),
		reports.QcaaNapoStudentResponseSetReport(),
	}
	//
	// add qa reports if requested
	//
	if tr.qaReports {
		qa := []pipelines.EventPipe{
			reports.QaSchoolsReport(),                                   // qa
			reports.SystemTestCompletenessReport(),                      // qa
			reports.QaStudentResultsCheckReport(tr.helper),              // qa
			reports.OrphanStudentsReport(),                              // qa
			reports.SystemResponsesReport(),                             // qa
			reports.OrphanEventsReport(),                                // qa
			reports.SystemMissingTestletsReport(),                       // qa
			reports.SystemParticipationCodeImpactsReport(),              // qa
			reports.SystemParticipationCodeItemImpactsReport(tr.helper), // qa
			reports.SystemTestAttemptsReport(),                          // qa
			reports.SystemTestIncidentsReport(),                         // qa
			reports.SystemTestTypeImpactsReport(),                       // qa
			reports.SystemTestTypeItemImpactsReport(tr.helper),          // qa
			reports.SystemStudentEventAcaraIdDiscrepanciesReport(),      // qa
			reports.SystemTestYearLevelDiscrepanciesReport(),            // qa
			reports.SystemRubricSubscoreMatchesReport(tr.helper),        // qa
			reports.ItemWritingPrintingReport(tr.helper),                // qa
			reports.ItemExpectedResponsesReport(tr.helper),              // qa
		}
		rpt = append(rpt, qa...)
	}

	// create the item reporting pipeline
	pl := pipelines.NewEventPipeline(rpt...)

	// create a progress bar
	bar := tr.uip.AddBar(tr.stats["NAPEventStudentLink"]) // Add a new bar
	bar.AppendCompleted().PrependElapsed()
	bar.PrependFunc(func(b *uiprogress.Bar) string {
		return strutil.Resize(" Event-based reports:", 35)
	})
	//
	// register an output handler for pipeline, used for progress-bar
	// but could also be audit sink, backup of processed records etc.
	//
	// NOTE: must be handler here even with empty body
	// otherwise exit channel blocks for pipeline
	//
	go pl.Dequeue(func(eor *records.EventOrientedRecord) {
		// easy win no-op, also reclaims memory
		// eor = nil
		bar.Incr() // update the progress bar
	})
	defer pl.Close()
	//
	// now iterate the codeframe records, passing them through
	// the processing pipeline
	//
	for eor := range em.EventBasedStream() {
		pl.Enqueue(eor)
	}

	return nil

}

//
// specilaised event-based processor to create
// input files for jurisdictional marking systems
//
func (tr *Transformer) writingExtractReports() error {

	// create a record emitter
	em, err := records.NewEmitter(records.EmitterRepository(tr.repository))
	if err != nil {
		return err
	}
	// create the item reporting pipeline
	pl := pipelines.NewEventPipeline(
		reports.QaSchoolsWritingExtractReport(), // qa report but always created with writing extract
		//
		// TODO: insert w/e greelist/redlist filters here...
		// filter should come only before writing-extract reports
		//
		reports.WritingExtractReport(),
		reports.WritingExtractQaPSIReport(),
	)
	// create a progress bar
	bar := tr.uip.AddBar(tr.stats["NAPEventStudentLink"]) // Add a new bar
	bar.AppendCompleted().PrependElapsed()
	bar.PrependFunc(func(b *uiprogress.Bar) string {
		return strutil.Resize(" Writing Extract reports:", 35)
	})
	//
	// register an output handler for pipeline, used for progress-bar
	// but could also be audit sink, backup of processed records etc.
	//
	// NOTE: must be handler here even with empty body
	// otherwise exit channel blocks for pipeline
	//
	go pl.Dequeue(func(eor *records.EventOrientedRecord) {
		// easy win no-op, also reclaims memory
		// eor = nil
		bar.Incr() // update the progress bar
	})
	defer pl.Close()
	//
	// now iterate the codeframe records, passing them through
	// the processing pipeline
	//
	for eor := range em.EventBasedStream() {
		pl.Enqueue(eor)
	}

	return nil

}

// XML Extract reports. Patterns after Simple object reports processor
// takes data objects and converts to XML
// with minimal business logic
func (tr *Transformer) xmlExtractReports() error {

	// create a record emitter
	em, err := records.NewEmitter(records.EmitterRepository(tr.repository))
	if err != nil {
		return err
	}
	tr.xmlWaitGroup = new(sync.WaitGroup)

	// create the pipeline members
	// regular reports
	rpt := []pipelines.ObjectPipe{
		// pre-processors
		reports.XmlRedactionReport(),

		// reports
		reports.XmlSingleOutputReport(tr.xmlWaitGroup),
		reports.XmlPerSchoolOutputReport(tr.objecthelper, tr.xmlWaitGroup),
	}

	// create the object report pipeline
	objpl := pipelines.NewObjectPipeline(rpt...)

	// create a progress bar
	barsize := tr.stats["SchoolInfo"] + tr.stats["StudentPersonal"] + tr.stats["NAPTestScoreSummary"] +
		tr.stats["NAPCodeFrame"] + tr.stats["NAPTest"] + tr.stats["NAPTestlet"] + tr.stats["NAPTestItem"] +
		tr.stats["NAPEventStudentLink"] + tr.stats["NAPStudentRsponseSet"]
	bar := tr.uip.AddBar(barsize)
	bar.AppendCompleted().PrependElapsed()
	bar.PrependFunc(func(b *uiprogress.Bar) string {
		return strutil.Resize(" Simple reports:", 35)
	})

	//
	// register an output handler for pipeline, used for progress-bar
	// but could also be audit sink, backup of processed records etc.
	//
	// NOTE: must be handler here even with empty body
	// otherwise exit channel blocks for pipeline
	//
	go objpl.Dequeue(func(cfr *records.ObjectRecord) {
		// easy win no-op, also reclaims memory
		// cfr = nil
		bar.Incr()
	})
	defer objpl.Close()
	//
	// now iterate the object records, passing them through
	// the processing pipeline
	//
	for or := range em.ObjectStream() {
		objpl.Enqueue(or)
	}

	return nil

}
