package main

import (
	"flag"
	"io"
	"log"
	"os"

	nrt "github.com/nsip/dev-nrt"
)

func main() {

	f := setuplog()
	defer f.Close()

	//
	// set of command-line flags to support, inlcuding n2 flags as aliases
	//
	var qa, ingest, itemLevel, coreReports, writingExtract, xmlExtract, showProgress, forceIngest, skipIngest, allReports, split, version bool
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
	flag.BoolVar(&xmlExtract, "xml", false, "run XML redaction reports; exports XML for ingested records with nominated fields removed")
	flag.BoolVar(&showProgress, "showProgress", true, "show progress bars in console for report processing")
	flag.StringVar(&inputFolder, "inputFolder", "./in/", "folder containing results data files for processing")
	flag.BoolVar(&allReports, "allReports", false, "runs every report including qa reports")
	flag.BoolVar(&split, "split", false, "runs trimmer/splitter only")
	flag.BoolVar(&version, "version", false, "print code version information")

	//
	// read any supplied flags
	//
	flag.Parse()

	// preeempt execution if version flag
	if version {
		showVersion()
		os.Exit(0)
	}

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
	if xmlExtract {
		opts = append(opts, nrt.XmlExtractReports(xmlExtract))
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
	if split {
		opts = append(opts, nrt.Split(split))
	}
	if allReports { // run everything
		all := []nrt.Option{
			nrt.ItemLevelReports(true),
			nrt.WritingExtractReports(true),
			nrt.XmlExtractReports(true),
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

func setuplog() *os.File {
	f, err := os.OpenFile("./naprrql.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	wrt := io.MultiWriter(os.Stdout, f)
	log.SetOutput(wrt)
	return f
}
