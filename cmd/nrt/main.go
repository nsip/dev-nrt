package main

import (
	"fmt"
	"log"

	nrt "github.com/nsip/dev-nrt"
	repo "github.com/nsip/dev-nrt/repository"
)

func main() {

	// create/open  a repo
	//
	// create a repo for the data
	//
	r, err := repo.NewBadgerRepo("./kv/")
	if err != nil {
		log.Println("cannot create repo", err)
	}
	defer r.Close()

	// run the ingest process
	resultsFolder := "../../testdata/"
	err = nrt.IngestResults(resultsFolder, r)
	if err != nil {
		log.Fatalln("ingest error:", err)
	}

	// show the ingest stats - from repo
	fmt.Println()
	objectStats := r.GetStats()
	for k, v := range objectStats {
		fmt.Printf("\t%s: %d\n", k, v)
	}
	fmt.Println()

	// fmt.Printf("\n%v\n", cfh.WritingRubricTypes())

	//
	// run the reports, pass repo
	//
	err = nrt.StreamResults(r)
	// err = nrt.StreamResults(r)
	if err != nil {
		log.Fatal(err)
	}

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
