package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/nsip/dev-nrt/files"
	"github.com/nsip/dev-nrt/sec"
)

func main() {

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

	fileName := "../../../testdata/n2sif.xml"
	// fileName := "../../testdata/rrd.xml"

	err := ConvertXMLToJsonFile(fileName, "./out/rrd.json", dataTypes...)
	if err != nil {
		log.Println("error converting xml file:", err)
	}
	fmt.Println("--- Conversion to file complete.")

}

//
// Takes an input stream of xml, converts to json and
// writes json to a file.
//
// xmlFileName: input file of xml results data
// jsonFileName: the ouput file for the converted json data
// dataObjects: the data types to extract from the stream (e.g. StudentPersonal, SchoolInfo etc.)
//
func ConvertXMLToJsonFile(xmlFileName string, jsonFileName string, dataObjects ...string) error {

	// open the xml file
	size, xmlStream, err := files.OpenXMLFile(xmlFileName)
	if err != nil {
		return err
	}

	// create the output file
	err = os.MkdirAll(filepath.Dir(jsonFileName), os.ModePerm)
	if err != nil {
		return err
	}
	f, err := os.Create(jsonFileName)
	if err != nil {
		return err
	}
	defer f.Close()

	bw := bufio.NewWriterSize(f, 65536)
	bw.WriteString(`[`)

	opts := []sec.Option{
		sec.ObjectsToExtract(dataObjects),
		// sec.SampleSize(2),
		// sec.AttributePrefix("ATTR_"),
		// sec.ContentToken("innerText"),
		sec.ProgressBar(size),
	}
	sec, err := sec.NewStreamExtractConverter(xmlStream, opts...)
	if err != nil {
		return err
	}

	count := 0
	for result := range sec.Stream() {

		if count > 0 {
			bw.WriteString(",\n")
		}

		bw.Write(result.Json)

		count++
	}

	bw.WriteString(`]`)
	bw.Flush()
	log.Printf("%d data-objects parsed\n\n", count)

	return nil

}
