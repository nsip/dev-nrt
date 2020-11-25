package reports

import (
	"fmt"
	"os"
	"path/filepath"

	toml "github.com/pelletier/go-toml"
)

//
// options common to all reports
//
type baseConfig struct {
	// whether the report proceses data..
	// if a report is de-activated it can still be part of a
	// processing pipeline, it will just act as a simple
	// passthrough
	activated bool
	// the file to write processing results to
	outputFileName string
	// human-friendly report name
	name string
	// the array of column headers for a typical csv output
	header []string
	// the associated array of json queries to extract field values for
	// each column of the report
	queries []string
}

//
// embed in any actual report to handle common
// setup tasks
//
type baseReport struct {
	// location of report's config file
	configFileName string
	// internal config block
	config *baseConfig
	// file to write results to
	outF *os.File
}

//
// handles boilerplate setup of each report to avoid
// repetition.
// Any new report should embed baseReport and call this
// method as part of its constructor.
//
// Method reads the config settings for the report
// and opens the output file to be used.
//
// Any failures with reading the config or opening the
// output file will be logged, and the report will be set to
// a de-activated state to prevent any further side-effects
//
// configFilePath: locationof config .toml file for this report
//
func (br *baseReport) initialise(configFilePath string) {

	br.configFileName = configFilePath

	//
	// read the report config
	//
	br.config = readConfigFromFile(configFilePath)
	if !br.config.activated { // nothing else to do
		return
	}

	//
	// open the ouput file
	//
	var err error
	br.outF, err = createOutputFile(br.config.outputFileName)
	if err != nil {
		fmt.Println("Warning: could not create report output file:", br.config.outputFileName)
		fmt.Printf("-- %s report will be disabled on this run.\n", br.config.name)
		br.config.activated = false
		return
	}

	// set splitter block

}

//
// brief summary of config, does not print json queries as too
// wordy
//
func (br *baseReport) printStatus() {
	fmt.Println("--------", br.config.name, "report status:")
	fmt.Printf("Config:\n-activated:\t%t\n-output-file:\t%s\n-config-file:\t%s\n",
		br.config.activated, br.config.outputFileName, br.configFileName)
	fmt.Println("--------")
}

//
// create the file this report will write records into
//
func createOutputFile(fileName string) (*os.File, error) {

	//
	// make the full file-path, in case doesn't exist
	//
	err := os.MkdirAll(filepath.Dir(fileName), os.ModePerm)
	if err != nil {
		return nil, err
	}

	//
	// create the file
	//
	return os.Create(fileName)

}

//
// reads the config .toml file for this report and populates
// the returned config structure
// if any errors are encoutered a default config with the report
// de-activated will be returned to prevent side-effects
//
func readConfigFromFile(configFilePath string) *baseConfig {

	// create a safe default config
	config := baseConfig{
		activated:      false,
		outputFileName: "",
		header:         make([]string, 0),
		queries:        make([]string, 0),
	}

	//
	// any errors in loading config just return default
	// with activated set to false
	//
	tomlConfig, err := toml.LoadFile(configFilePath)
	if err != nil {
		return &config
	}

	//
	// fetch config properties
	//
	if activated, ok := tomlConfig.Get("usage.activated").(bool); ok {
		config.activated = activated
	}
	//
	if filename, ok := tomlConfig.Get("usage.outputFileName").(string); ok {
		config.outputFileName = filename
	}
	//
	if rname, ok := tomlConfig.Get("usage.reportName").(string); ok {
		config.name = rname
	}

	//
	// get the query / header definitions
	// if missing, de-activate report
	//
	fields := tomlConfig.Get("fields")
	if _, ok := fields.([]*toml.Tree); !ok {
		fmt.Printf("\n\t Warning: no report fields defined in %s\n", configFilePath)
		fmt.Printf("-------- %s report will be disabled on this run.\n", config.name)
		config.activated = false
		return &config
	}
	// iterate definitions
	for _, f := range fields.([]*toml.Tree) {
		m := f.ToMap()
		for k, v := range m {
			config.header = append(config.header, k)
			config.queries = append(config.queries, v.(string))
		}
	}

	if len(config.header) == 0 || len(config.queries) == 0 {
		fmt.Printf("\n\t Warning: report fields not defined correctly in %s\n", configFilePath)
		fmt.Printf("-------- %s report will be disabled on this run.\n", config.name)
		config.activated = false
		return &config
	}

	return &config

}
