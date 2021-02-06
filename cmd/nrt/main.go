package main

import (
	"log"

	nrt "github.com/nsip/dev-nrt"
)

func main() {

	opts := []nrt.Option{
		// nrt.ShowProgress(true),
		// nrt.SkipIngest(true),
		// nrt.StopAfterIngest(true),
		// nrt.WritingExtractReports(true),
		// nrt.ItemLevelReports(true),
	}
	tr, err := nrt.NewTransformer(opts...)
	if err != nil {
		log.Println("Error building transformer", err)
	}

	err = tr.Run()
	if err != nil {
		log.Println("Error running transformer", err)
	}

}
