package nrt

import (
	"bufio"
	"io"
	"log"
	"os"
	"path/filepath"
)

//
// Takes an input stream of xml, converts to json and
// writes json to a file.
//
// xmlStream: input file/stream of xml results data
// jsonFileName: the ouput file for the converted json data
// dataObjects: the data types to extract from the stream (e.g. StudentPersonal, SchoolInfo etc.)
//
func StreamToJsonFile(xmlStream io.Reader, jsonFileName string, dataObjects ...string) error {

	err := os.MkdirAll(filepath.Dir(jsonFileName), os.ModePerm)
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

	sec := NewStreamExtractConverter(xmlStream, dataObjects...)
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
	log.Printf("%d data-objects parsed\n", count)

	return nil

}
