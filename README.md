# dev-nrt
temporary home for early code relating to nrt

Contains some initial experiments to speed up data ingest for NAPLAN reporting.

Sample data to really exercise the code needs to be decent size.
Have run coparative tests against exisitng n2 ingest using a sample file
created with the perl script which can be found here:

https://github.com/nsip/naplan-results-reporting/tree/master/SampleData
(Note, you'll need to install CPAN and the modules listed in the top of the
file to get it working if you don't have a comprehensive perl environment 
already on youur machine).

I ran this as :
```
perl nap_platformdata_generator.pl 50 100
```
to generate file of >500Mb

on n2, ingest is around 40 seconds
with nrt (basic) ingest is around 14 seconds.

The utilities expose 2 methods to handle the conversion
one sends the json to a k/v store as normal (badger in this case).
The other writes the ouptut to file/files, useful for debugging.
The list of objects to consume from the xml stream is also configurable, so
you could (for example) create one file with all codeframe data, one with results, and one with students etc. etc. Default sample code writes everything to one file.

```go
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

```

So here's the question, given that the largest dataset within the xml file is the student results, parsing just those objects only takes around 10 seconds.

So in theory if I wrap the methods above in goroutines (one for each data object say, creating a file of output for each), then I should be able to get the whole process to complete in the time of the longest extraction, given that all the others are shorter.

However, this is much slower! I'm thinking becasue of the cpu usage and the small number of cores on my laptop; more than one process means too much timeslicing, would like to see what happens and what's possible
on larger machines -  we can golinear or concurrent based on detecting
the size of the host machine??









