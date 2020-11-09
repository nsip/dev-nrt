package files

import (
	"archive/zip"
	"io"
	"log"
	"os"
	"path/filepath"
)

//
// Utility function to open a reader on a results
// data file.
// File can be zipped or regular, but this method
// does not handle password protected zip files.
//
// returns fileSize in bytes, reader for file, any errors
//
func OpenXMLFile(fname string) (int, io.Reader, error) {

	if isZipFile(fname) {
		return openDataFileZip(fname)
	}
	return openDataFile(fname)

}

func isZipFile(fname string) bool {

	xmlZipFile, err := zip.OpenReader(fname)
	if err != nil {
		return false
	}
	defer xmlZipFile.Close()

	return true

}

func openDataFileZip(fname string) (int, io.Reader, error) {

	xmlZipFile, err := zip.OpenReader(fname)
	if err != nil {
		return 0, nil, err
	}
	// assume only one file in the archive
	xmlFile, err := xmlZipFile.File[0].Open()
	if err != nil {
		return 0, nil, err
	}
	size := xmlZipFile.File[0].FileHeader.UncompressedSize64

	return int(size), xmlFile, nil

}

func openDataFile(fname string) (int, io.Reader, error) {

	xmlFile, err := os.Open(fname)
	if err != nil {
		return 0, nil, err
	}
	info, err := xmlFile.Stat()
	if err != nil {
		return 0, nil, err
	}
	size := info.Size()

	return int(size), xmlFile, nil

}

//
// look for results data files in specified folder
//
func ParseResultsDirectory(folderName string) []string {

	files := make([]string, 0)
	fn := filepath.Dir(folderName)
	zipFiles, _ := filepath.Glob(fn + "/*.zip")
	xmlFiles, _ := filepath.Glob(fn + "/*.xml")

	files = append(files, zipFiles...)
	files = append(files, xmlFiles...)
	if len(files) == 0 {
		log.Fatalln("No results data *.zip *.xml.zip or *.xml files found in input folder", fn)
	}

	return files

}
