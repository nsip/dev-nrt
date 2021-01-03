package utils

import (
	"log"
	"time"
)

// timetrack.go

//
// small utility function embedded in major ops like
// queries to print a performance indicator.
//
func TimeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	log.Printf("%s took %s", name, elapsed.Truncate(time.Millisecond).String())

}
