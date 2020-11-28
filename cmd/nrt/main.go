package main

import (
	"fmt"
	"log"

	nrt "github.com/nsip/dev-nrt"
)

func main() {

	// create the output folder

	resultsFolder := "../../testdata/"

	totals, err := nrt.IngestResults(resultsFolder)
	if err != nil {
		log.Fatal(err)
	}
	// show the ingest stats
	fmt.Println()
	for k, v := range totals {
		fmt.Printf("\t%s: %d\n", k, v)
	}
	fmt.Println()

	err = nrt.StreamResults(totals)
	if err != nil {
		log.Fatal(err)
	}

	//
	// stream results
	// multi-call vs. multi goroutines speed check
	//
	// then 2 streams; event-based, student-based
	//

	//
	// attach reports
	//

	//
	// split
	//
	//

	//
	// remove null
	//

	//
	// save run and clean-up
	//

	//
	//
	//

}
