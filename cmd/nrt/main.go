package main

import (
	"log"

	nrt "github.com/nsip/dev-nrt"
)

func main() {

	resultsFolder := "../../testdata/"

	err := nrt.IngestResults(resultsFolder)
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
	//
	//

}
