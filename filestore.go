package nrt

import (
	"bufio"
	"log"
	"os"
	"path/filepath"
)

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
	size, xmlStream, err := OpenXMLFile(xmlFileName)
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

	opts := []Option{
		ObjectsToExtract(dataObjects),
		// SampleSize(5),
		// AttributePrefix("ATTR_"),
		// ContentToken("innerText"),
		ProgressBar(size),
	}
	sec, err := NewStreamExtractConverter(xmlStream, opts...)
	if err != nil {
		return err
	}

	count := 0
	for jsonBytes := range sec.Stream() {

		if count > 0 {
			bw.WriteString(",\n")
		}

		bw.Write(jsonBytes)

		count++
	}

	bw.WriteString(`]`)
	bw.Flush()
	log.Printf("%d data-objects parsed\n\n", count)

	return nil

}
