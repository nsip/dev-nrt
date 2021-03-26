package nrt

import (
	"fmt"
	"os"

	"github.com/gosuri/uiprogress"
	splitter "github.com/nsip/dev-nrt-splitter"
	"github.com/nsip/dev-nrt/codeframe"
	"github.com/nsip/dev-nrt/repository"
)

// note, if repo nil, then post-ingest cannot continue
// if repo existing, go straight to reports
// logic???
//
// ingest currently stops flow.
//

//
// the core nrt engine, passes the streams of
// rrd data through pipelines of report processors
// creating tabluar and fixed-width reports from the
// xml data
//
type Transformer struct {
	repository       *repository.BadgerRepo
	inputFolder      string
	uip              *uiprogress.Progress
	helper           codeframe.Helper
	qaReports        bool
	itemLevelReports bool
	coreReports      bool
	wxReports        bool
	forceIngest      bool
	skipIngest       bool
	stopAfterIngest  bool
	showProgress     bool
	stats            repository.ObjectStats
}

func NewTransformer(userOpts ...Option) (*Transformer, error) {

	// generate the standard options
	opts := defaultOptions()

	// now add any user/caller options
	opts = append(opts, userOpts...)

	tr := Transformer{uip: uiprogress.New()}
	tr.setOptions(opts...)

	return &tr, nil

}

//
// Ingest and process data from RRD files
//
func (tr *Transformer) Run() error {

	// ======================================
	//
	// Create or open data repository
	//
	var r *repository.BadgerRepo
	var err error
	if tr.forceIngest {
		r, err = repository.NewBadgerRepo("./kv/")
		if err != nil {
			return err
		}
		tr.repository = r
	}

	if tr.skipIngest {
		fmt.Println("--- Skipping ingest, using existing data:")
		r, err = repository.OpenExistingBadgerRepo("./kv/")
		if err != nil {
			return err
		}
		tr.repository = r
	}

	if !tr.skipIngest {
		fmt.Println("--- Ingesting results data:")
		err = ingestResults(tr.inputFolder, r)
		if err != nil {
			return err
		}
	}
	defer tr.repository.Close()

	if tr.stopAfterIngest {
		fmt.Println("--- Halting post-ingest.")
		return nil
	}

	// =======================================
	//
	// Get the data stats, and show to user
	//
	fmt.Println("--- Data stats:")
	fmt.Println()
	tr.stats = r.GetStats()
	for k, v := range tr.stats {
		fmt.Printf("\t%s: %d\n", k, v)
	}
	fmt.Println()

	// ====================================
	//
	// Build the codeframe helper
	//
	cfh, err := codeframe.NewHelper(r)
	if err != nil {
		return err
	}
	tr.helper = cfh

	// ==================================
	//
	// run the report processing streams
	//
	err = tr.streamResults()
	if err != nil {
		return err
	}

	// =================================
	//
	// split
	splitter.EnableProgBar(tr.showProgress)
	err = splitter.NrtSplit("./config_split/config.toml")
	if err != nil {
		return err
	}

	// =================================
	//
	// tidy up
	err = os.RemoveAll("./out/null/")
	if err != nil {
		return err
	}

	return nil
}

// ==========================================
//
// Options
//

//
// set the default options for a transformer
// basic reports, no pause after ingest, does show progress bars
//
func defaultOptions() []Option {
	return []Option{
		InputFolder("./in/"),
		QAReports(false),
		ItemLevelReports(false),
		CoreReports(true),
		WritingExtractReports(false),
		ForceIngest(true),
		StopAfterIngest(false),
		ShowProgress(true),
	}
}

type Option func(*Transformer) error

//
// apply all supplied options to the transformer
// returns any error encountered while applying the options
//
func (tr *Transformer) setOptions(options ...Option) error {
	for _, opt := range options {
		if err := opt(tr); err != nil {
			return err
		}
	}
	return nil
}

//
// Show progress bars for report processing.
// (The progress bar for data ingest is always shown)
// Defaults to true.
// Reasons to disable are: to get clear visibility of any
// std.Out console messages so they don't mix with the console
// progress bars. Also if piping the output to a file
// the progress bars are witten out sequentially producing a lot
// of unnecessary noise data in the file.
//
func ShowProgress(sp bool) Option {
	return func(tr *Transformer) error {
		tr.showProgress = sp
		return nil
	}
}

//
// Make transformer stop once data ingest is complete
// various report configurations can then be run
// independently without reloading the results data
// Default is false, tranformer will ingest data and move
// directly to report processing
//
func StopAfterIngest(sai bool) Option {
	return func(tr *Transformer) error {
		tr.stopAfterIngest = sai
		return nil
	}
}

//
// Even if an existing datastore has been created
// in the past, this option ensures that the old data
// is removed and the ingest cycle is run again
// reading all data files from the input folder.
// Default is true.
//
func ForceIngest(fi bool) Option {
	return func(tr *Transformer) error {
		tr.forceIngest = fi
		return nil
	}
}

//
// Tells NRT to go straight the the report processing
// activity, as data has already been ingested at
// an earlier point in time
//
func SkipIngest(si bool) Option {
	return func(tr *Transformer) error {
		tr.skipIngest = si
		tr.forceIngest = false // must also be set
		return nil
	}
}

//
// indicate whether the writing-extract reports
// (input to downstream writing marking systems)
// will be included in this run of the transformer
//
func WritingExtractReports(wx bool) Option {
	return func(tr *Transformer) error {
		tr.wxReports = wx
		if wx {
			tr.coreReports = false // set so can run just wx, or add core as later option to re-instate
		}
		return nil
	}
}

//
// indicate whether the most heavyweight/detailed
// reports will be included in this run of the
// transformer, including these has the largest effect
// on overall processing time
//
func ItemLevelReports(ilevel bool) Option {
	return func(tr *Transformer) error {
		tr.itemLevelReports = ilevel
		return nil
	}
}

//
// indicate whether the most-used common
// reports will be included in this run of the
// transformer
//
func CoreReports(core bool) Option {
	return func(tr *Transformer) error {
		tr.coreReports = core
		return nil
	}
}

//
// indicate whether qa reports will be included
// in this run of the transformer
//
func QAReports(qa bool) Option {
	return func(tr *Transformer) error {
		tr.qaReports = qa
		return nil
	}
}

//
// the folder contaning RRD xml data files for
// processing
//
func InputFolder(path string) Option {
	return func(tr *Transformer) error {
		// try to create the folder if it doesn't exist
		if _, err := os.Stat(path); os.IsNotExist(err) {
			if err := os.Mkdir(path, os.ModePerm); err != nil {
				return err
			}
		}
		tr.inputFolder = path
		return nil
	}
}

//
// the key-value store cotaining the ingested RRD data
//
func Repository(repo *repository.BadgerRepo) Option {
	return func(tr *Transformer) error {
		tr.repository = repo
		return nil
	}
}
