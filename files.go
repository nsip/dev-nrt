package nrt

import (
	"archive/zip"
	"io"
	"log"
	"os"
)

//
// Utility function to open a reader on a results
// data file.
// File can be zipped or regular, but this method
// does not handle password protected zip files.
//
func OpenXMLFile(fname string) (io.Reader, error) {
	var xmlFile io.Reader
	var ferr error

	if isZipFile(fname) {
		xmlFile, ferr = openDataFileZip(fname)
	} else {
		xmlFile, ferr = openDataFile(fname)
	}
	if ferr != nil {
		return xmlFile, ferr
	}

	return xmlFile, ferr

}

func isZipFile(fname string) bool {

	xmlZipFile, err := zip.OpenReader(fname)
	if err != nil {
		return false
	}
	defer xmlZipFile.Close()

	return true

}

func openDataFileZip(fname string) (io.Reader, error) {

	xmlZipFile, err := zip.OpenReader(fname)
	if err != nil {
		log.Println("Unable to open zip file: ", err)
		return nil, err
	}
	// assume only one file in the archive
	xmlFile, err := xmlZipFile.File[0].Open()
	if err != nil {
		return xmlFile, err
	}

	return xmlFile, nil

}

func openDataFile(fname string) (io.Reader, error) {

	xmlFile, err := os.Open(fname)
	if err != nil {
		return xmlFile, err
	}
	return xmlFile, nil

}

// //
// // look for results data files
// //
// func parseResultsFileDirectory() []string {

// 	files := make([]string, 0)

// 	zipFiles, _ := filepath.Glob("./in/*.zip")
// 	xmlFiles, _ := filepath.Glob("./in/*.xml")

// 	files = append(files, zipFiles...)
// 	files = append(files, xmlFiles...)
// 	if len(files) == 0 {
// 		log.Fatalln("No results data *.zip *.xml.zip or *.xml files found in input folder /in.")
// 	}

// 	return files

// }
