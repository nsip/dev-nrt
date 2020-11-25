package nrt

import (
	"fmt"
	"sync"
	"time"

	"github.com/gosuri/uiprogress"
	"github.com/gosuri/uiprogress/util/strutil"
	"github.com/nsip/dev-nrt/records"
	"github.com/nsip/dev-nrt/reports"
	repo "github.com/nsip/dev-nrt/repository"
)

//
// extracts streams of results from the repository
// and feeds them through pipelines of reports
// accepts a stas map to calibrate the progress bars
// (one is produced by IndestResults())
//
func StreamResults(stats map[string]int) error {

	defer timeTrack(time.Now(), "StreamResults()")

	//
	// create the reporting pipelines
	//
	fmt.Printf("\n\n--- Initialising Reports:\n")
	epl := reports.NewEventPipeline(
		reports.WritingExtractReport(),
		// reports.WritingExtractReport(),//nb may have to add w/e whitelist filter.
	)

	//
	// get the results data repository
	//
	r, err := repo.OpenExistingBadgerRepo("./kv/")
	if err != nil {
		return err
	}
	defer r.Close()

	//
	// create the emitter
	//
	opts := []records.Option{records.EmitterRepository(r)}
	em, err := records.NewEmitter(opts...)
	if err != nil {
		return err
	}

	//
	// set up the progress bars
	//
	var uip *uiprogress.Progress
	var eventBar, studentBar *uiprogress.Bar
	uip = uiprogress.New()
	eventBar = uip.AddBar(stats["NAPEventStudentLink"]) // Add a new bar
	eventBar.AppendCompleted().PrependElapsed()
	studentBar = uip.AddBar(stats["StudentPersonal"])
	studentBar.AppendCompleted().PrependElapsed()
	eventBar.PrependFunc(func(b *uiprogress.Bar) string {
		return strutil.Resize(" Event-based reports:", 25)
	})
	studentBar.PrependFunc(func(b *uiprogress.Bar) string {
		return strutil.Resize(" Student-based reports:", 25)
	})

	//
	// launch pipelines & emit events into them
	//
	fmt.Printf("\n\n--- Running Reports:\n\n")
	uip.Start()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		//
		// register an output handler for pipeline, used for progress-bar
		// but could also be audit sink, backup of processed records etc.
		//
		// NOTE: must be handler here even with empty body
		// otherwise exit channel blocks for pipeline
		//
		go epl.Dequeue(func(eor *records.EventOrientedRecord) {
			// easy win no-op, also reclaims memory
			eor = nil
			eventBar.Incr()
		})
		defer epl.Close()
		defer wg.Done()
		for eor := range em.EventBasedStream() {
			epl.Enqueue(eor)
		}
	}()

	wg.Wait()
	uip.Stop()

	fmt.Printf("\n All report streams completed.\n\n")

	return nil

}

//
//
//
