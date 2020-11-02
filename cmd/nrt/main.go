package main

import (
	"log"
	"os"

	nrt "github.com/nsip/dev-nrt"
)

func main() {

	//
	// obtain reader for file of interest
	//
	f, _ := os.Open("../../testdata/n2sif.xml") // normal sample xml file
	// f, _ := os.Open("../../testdata/rrd.xml") // large 500Mb sample file
	defer f.Close()

	//
	// superset of data objects we can extract from the
	// stream
	//
	var dataTypes = []string{
		"NAPStudentResponseSet",
		"NAPEventStudentLink",
		"StudentPersonal",
		"NAPTestlet",
		"NAPTestItem",
		"NAPTest",
		"NAPCodeFrame",
		"SchoolInfo",
		"NAPTestScoreSummary",
	}

	// err := nrt.StreamToJsonFile(f, "./out/rrd.json", dataTypes...)
	err := nrt.StreamToKVStore(f, "./kv/", nrt.IdxSifObjectByRefId(), dataTypes...)
	if err != nil {
		log.Println("error converting xml file:", err)
	}

}
