package nrt

import (
	"context"
	"fmt"

	"time"

	"golang.org/x/sync/errgroup"

	"github.com/gosuri/uiprogress"
	"github.com/gosuri/uiprogress/util/strutil"
	"github.com/nsip/dev-nrt/codeframe"
	"github.com/nsip/dev-nrt/pipelines"
	"github.com/nsip/dev-nrt/records"
	"github.com/nsip/dev-nrt/reports"
	repo "github.com/nsip/dev-nrt/repository"
	"github.com/nsip/dev-nrt/utils"
)

//
// extracts streams of results from the repository
// and feeds them through pipelines of reports
//
func StreamResults(r *repo.BadgerRepo) error {

	defer utils.TimeTrack(time.Now(), "StreamResults()")

	//
	// get cardinality of data objects from repo
	//
	stats := r.GetStats()

	//
	// create the codeframe helper tool
	//
	cfh, err := codeframe.NewHelper(r)
	if err != nil {
		return err
	}

	//
	// set up the progress bar display manager
	//
	var uip *uiprogress.Progress
	uip = uiprogress.New()
	defer uip.Stop()
	fmt.Printf("\n\n--- Running Reports:\n\n")
	uip.Start()
	// uip.Stop()

	//
	// start the different processing pipelines concurrently
	// under control of a group
	//
	g, _ := errgroup.WithContext(context.Background())

	//
	// Simple object reports processor
	// takes data objects and converts to csv
	// with no business logic
	//
	g.Go(func() error {

		// create a record emitter
		em, err := records.NewEmitter(records.EmitterRepository(r))
		if err != nil {
			return err
		}
		// create the object report pipeline
		objpl := pipelines.NewObjectPipeline(
			reports.SystemObjectsCountReport(),
			reports.QcaaNapoSchoolsReport(),
			reports.QcaaNapoStudentsReport(),
			reports.QcaaTestScoreSummaryReport(),
			reports.SystemSchoolsReport(),
			reports.SystemScoreSummariesReport(),
			reports.QldStudentReport(),
			reports.SystemExtraneousCharactersStudentsReport(),
			reports.QaSystemScoreSummariesReport(cfh),
		)
		// create a progress bar
		objObjectsCount := stats["SchoolInfo"] + stats["StudentPersonal"] + stats["NAPTestScoreSummary"]
		objBar := uip.AddBar(objObjectsCount)
		objBar.AppendCompleted().PrependElapsed()
		objBar.PrependFunc(func(b *uiprogress.Bar) string {
			return strutil.Resize(" Simple reports:", 25)
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
			objBar.Incr()
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
	})

	//
	// Student-oriented reports processor
	// each record contains all test responses and events
	// for a given student
	//
	g.Go(func() error {

		// return nil

		// create a record emitter
		em, err := records.NewEmitter(records.EmitterRepository(r))
		if err != nil {
			return err
		}
		// create the object report pipeline
		pl := pipelines.NewStudentPipeline(
			// pre-processors
			//
			reports.StudentRecordSplitterBlockReport(),
			reports.DomainParticipationReport(),
			reports.DomainResponsesScoresReport(),
			// reports
			//
			reports.SystemParticipationReport(),
			reports.IsrPrintingReport(),
			//
			// pre-processors
			//
			reports.DomainDACReport(cfh),
			// reports
			//
			reports.IsrPrintingExpandedReport(),
			reports.NswPrintReport(),
			//
			// pre- processors
			//
			reports.DomainItemResponsesReport("Reading"),
			reports.DomainItemResponsesReport("Spelling"),
			reports.DomainItemResponsesReport("GrammarAndPunctuation"),
			reports.DomainItemResponsesReport("Numeracy"),
			reports.DomainItemResponsesReport("Writing"),
			reports.DomainItemResponsesWritingRubricsReport(), // gives further writing breakdown by subscore
			//
			// reports
			reports.NswAllPearsonY3Report(),
			reports.NswAllPearsonY5Report(),
			reports.NswAllPearsonY7Report(),
			reports.NswAllPearsonY9Report(),
			//
			reports.PrintAllReport(),
		)
		// create a progress bar
		stuBar := uip.AddBar(stats["StudentPersonal"])
		stuBar.AppendCompleted().PrependElapsed()
		stuBar.PrependFunc(func(b *uiprogress.Bar) string {
			return strutil.Resize(" Student-based reports:", 25)
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
			stuBar.Incr()
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
	})

	//
	// CodeFrame reports processor
	// reports that use just the objects from the codeframe
	// test, teslet, item etc.
	//
	g.Go(func() error {

		// create a record emitter
		em, err := records.NewEmitter(records.EmitterRepository(r))
		if err != nil {
			return err
		}
		// create the codeframe report pipeline
		cfpl := pipelines.NewCodeframePipeline(
			reports.QcaaNapoTestletsReport(cfh),
			// remaining codeframe reports need item-test-link multiplexer to
			// come first in the pipeline sequence
			reports.ItemTestLinkReport(cfh),
			reports.QcaaNapoItemsReport(),
			reports.QcaaNapoTestsReport(),
			reports.QcaaNapoTestletItemsReport(),
			reports.SystemCodeframeReport(cfh),
			reports.QaSystemCodeframeReport(cfh),
			reports.QldTestDataReport(cfh),
			reports.QcaaNapoWritingRubricReport(),
			// keep this pair at end of pipeline to minimize data expansion
			reports.ItemRubricExtractorReport(),
			reports.QcaaNapoWritingRubricReport(),
		)
		// create a progress bar
		cfObjectsCount := stats["NAPCodeFrame"] + stats["NAPTest"] + stats["NAPTestlet"] + stats["NAPTestItem"]
		codeframeBar := uip.AddBar(cfObjectsCount)
		codeframeBar.AppendCompleted().PrependElapsed()
		codeframeBar.PrependFunc(func(b *uiprogress.Bar) string {
			return strutil.Resize(" Codeframe reports:", 25)
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
	})

	//
	// Item reports processor
	// Special case of event-oriented record processor
	//
	g.Go(func() error {

		// create a record emitter
		em, err := records.NewEmitter(records.EmitterRepository(r))
		if err != nil {
			return err
		}
		// create the item reporting pipeline
		pl := pipelines.NewEventPipeline(
			// processors to set up reports, need to be kept in this order
			reports.EventRecordSplitterBlockReport(),
			reports.ItemResponseExtractorReport(),
			reports.ItemDetailReport(cfh),
			// actual reports
			reports.NswItemPrintingReport(),
			reports.QcaaNapoStudentResponsesReport(),
			reports.ItemPrintingReport(),
		)
		// create a progress bar
		itemBar := uip.AddBar(stats["NAPEventStudentLink"] * 25) // 25 is expansion factor of events to items
		itemBar.AppendCompleted().PrependElapsed()
		itemBar.PrependFunc(func(b *uiprogress.Bar) string {
			return strutil.Resize(" Item-level reports:", 25)
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
			itemBar.Incr() // update the progress bar
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
	})

	//
	// Event-based report processor
	// each record is the test, school, student, event & response
	//
	g.Go(func() error {

		// create a record emitter
		em, err := records.NewEmitter(records.EmitterRepository(r))
		if err != nil {
			return err
		}
		// create the item reporting pipeline
		pl := pipelines.NewEventPipeline(
			//
			//
			reports.EventRecordSplitterBlockReport(),
			reports.ActSystemDomainScoresReport(),
			reports.QldStudentScoreReport(),
			reports.SystemDomainScoresReport(),
			reports.CompareRRDtestsReport(),
			//
			// TODO: insert w/e greelist/redlist filters here...
			// filter should come only before writing-extract reports
			//
			reports.WritingExtractReport(),
			reports.WritingExtractQaPSIReport(),
			reports.SaHomeschooledTestsReport(),
			reports.CompareItemWritingReport(),
			reports.NswWritingPearsonY3Report(cfh),
			reports.NswWritingPearsonY5Report(cfh),
			reports.NswWritingPearsonY7Report(cfh),
			reports.NswWritingPearsonY9Report(cfh),
			reports.SystemPNPEventsReport(),
			reports.QcaaNapoEventStudentLinkReport(),
			reports.QcaaNapoStudentResponseSetReport(),
			reports.OrphanStudentsReport(),
			reports.QaCodeframeCheckReport(cfh),
			reports.SystemResponsesReport(),
			reports.OrphanEventsReport(),
			reports.SystemMissingTestletsReport(),
			reports.SystemParticipationCodeImpactsReport(),
			reports.SystemParticipationCodeItemImpactsReport(cfh),
		)
		// create a progress bar
		eventBar := uip.AddBar(stats["NAPEventStudentLink"]) // Add a new bar
		eventBar.AppendCompleted().PrependElapsed()
		eventBar.PrependFunc(func(b *uiprogress.Bar) string {
			return strutil.Resize(" Event-based reports:", 25)
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
			eventBar.Incr() // update the progress bar
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
	})

	// wait for report streams to end
	if g.Wait() != nil {
		return g.Wait()
	}

	//
	// allow time for flush of large csv files to complete
	//
	time.Sleep(time.Millisecond * 1200)

	// n.b. output stats as objectcount report

	return nil

}

//
//
//
