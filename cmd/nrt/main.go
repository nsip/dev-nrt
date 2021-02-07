package main

import (
	"flag"
	"log"

	nrt "github.com/nsip/dev-nrt"
)

func main() {

	//
	// set of command-line flags to support, inlcuding n2 flags as aliases
	//
	var qa, ingest, itemLevel, coreReports, writingExtract, showProgress, forceIngest, skipIngest, allReports bool
	var inputFolder string
	flag.BoolVar(&qa, "qa", false, "include QA reports in results processing")
	flag.BoolVar(&ingest, "ingest", false, "halt processing after data has been ingested")
	flag.BoolVar(&forceIngest, "forceIngest", true, "always ingest data from files in the input folder")
	flag.BoolVar(&skipIngest, "skipIngest", false, "don't ingest data again, move directly to processing of reports")
	flag.BoolVar(&itemLevel, "itemprint", false, "include item-level reports - very detailed (alias for nrt -itemLevel flag)")
	flag.BoolVar(&itemLevel, "itemLevel", false, "include item-level reports - very detailed (alias for n2 -itemprint flag)")
	flag.BoolVar(&coreReports, "report", false, "run core naplan reports - (alias for nrt -coreReports flag)")
	flag.BoolVar(&coreReports, "coreReports", false, "run core naplan reports - (alias for n2 -reports flag)")
	flag.BoolVar(&writingExtract, "writingextract", false, "run writing extract reports; generates writing inputs for marking systems")
	flag.BoolVar(&writingExtract, "writingExtract", false, "(alias) run writing extract reports; generates writing inputs for marking systems")
	flag.BoolVar(&showProgress, "showProgress", true, "show progress bars in console for report processing")
	flag.StringVar(&inputFolder, "inputFolder", "./in/", "folder containing results data files for processing")
	flag.BoolVar(&allReports, "allReports", false, "runs every report including qa reports")

	//
	// read any supplied flags
	//
	flag.Parse()

	//
	// convert supplied flags into nrt options
	opts := []nrt.Option{}
	if qa {
		opts = append(opts, nrt.QAReports(qa))
	}
	if ingest {
		opts = append(opts, nrt.StopAfterIngest(ingest))
	}
	if !forceIngest {
		opts = append(opts, nrt.ForceIngest(forceIngest))
	}
	if skipIngest {
		opts = append(opts, nrt.SkipIngest(skipIngest))
	}
	if itemLevel {
		opts = append(opts, nrt.ItemLevelReports(itemLevel))
	}
	if writingExtract {
		opts = append(opts, nrt.WritingExtractReports(writingExtract))
	}
	if coreReports { // always add after writingExtract to support running both at same time if required
		opts = append(opts, nrt.CoreReports(coreReports))
	}
	if !showProgress {
		opts = append(opts, nrt.ShowProgress(showProgress))
	}
	if inputFolder != "./in/" {
		opts = append(opts, nrt.InputFolder(inputFolder))
	}
	if allReports { // run everything
		all := []nrt.Option{
			nrt.ItemLevelReports(true),
			nrt.WritingExtractReports(true),
			nrt.CoreReports(true),
			nrt.ItemLevelReports(true),
			nrt.QAReports(true),
		}
		opts = append(opts, all...)
	}


	tr, err := nrt.NewTransformer(opts...)
	if err != nil {
		log.Println("Error building transformer", err)
	}

	err = tr.Run()
	if err != nil {
		log.Println("Error running transformer", err)
	}

}
