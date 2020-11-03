package main

import (
	"fmt"
	"log"

	nrt "github.com/nsip/dev-nrt"
)

func main() {

	//
	// superset of data objects we can extract from the
	// stream
	//
	var dataTypes = []string{
		// "NAPStudentResponseSet",
		// "NAPEventStudentLink",
		// "StudentPersonal",
		// "NAPTestlet",
		// "NAPTestItem",
		// "NAPTest",
		// "NAPCodeFrame",
		"SchoolInfo",
		// "NAPTestScoreSummary",
	}

	fileName := "../../testdata/n2sif.xml"
	// fileName := "../../testdata/rrd.xml"

	// err := nrt.ConvertXMLToJsonFile(fileName, "./out/rrd.json", dataTypes...)
	// if err != nil {
	// 	log.Println("error converting xml file:", err)
	// }
	// fmt.Println("--- Conversion to file complete.")

	err = nrt.StreamToKVStore(fileName, "./kv/", nrt.IdxSifObjectByRefId(), dataTypes...)
	if err != nil {
		log.Println("error converting xml file:", err)
	}
	fmt.Println("--- Storage to kv db complete.")

}
