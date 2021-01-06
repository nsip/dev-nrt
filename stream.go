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
	// set up the progress bar display manager
	//
	var uip *uiprogress.Progress
	uip = uiprogress.New()
	defer uip.Stop()
	fmt.Printf("\n\n--- Running Reports:\n\n")
	uip.Start()

	//
	// create the codeframe helper tool
	//
	cfh, err := codeframe.NewHelper(r)
	if err != nil {
		return err
	}

	//
	// start the different processing pipelines concurrently
	// under control of a group
	//
	g, _ := errgroup.WithContext(context.Background())

	//
	// CodeFrame reports processor
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
			// mulitplexer must come before items report
			reports.ItemTestLinkReport(cfh),
			reports.QcaaNapoItemsReport(),
			reports.QcaaNapoTestsReport(),
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
			cfr = nil
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
	//
	g.Go(func() error {
		// create a record emitter
		em, err := records.NewEmitter(records.EmitterRepository(r))
		if err != nil {
			return err
		}
		// create the item reporting pipeline
		pl := pipelines.NewEventPipeline(
			// processors to set up reports
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
			eor = nil
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
			//
			// insert w/e filters here...
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
			eor = nil
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

	//
	// group waits for all goroutines to complete & returns any error encountered
	//
	return g.Wait()

}

//
//
//
