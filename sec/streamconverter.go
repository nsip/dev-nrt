package sec

import (
	"bufio"
	"errors"
	"fmt"
	"io"

	"strings"

	"github.com/gosuri/uiprogress"
	jsoniter "github.com/json-iterator/go"
	xmlparser "github.com/tamerh/xml-stream-parser"
)

//
// number of objects to extract (mostly when testing/experimenting)
// set to 0 for no restriction
//
var defaultSampleSize int = 0

//
// when element has attributes that become json keys
// use this token to identify the original innertext
// of the element
// (defaults to PESC json 'value')
//
var defaultContentToken string = "value"

//
// optional identifier prepended to attributes
// that have been made into json oject keys
// for easier identification
//
var defaultAttributePrefix string = ""

//
// Captures output from the converter
// Name: the class/type/object name of the converted entity
// Json: the generated json from the converter
//
type Result struct {
	Name string
	Json []byte
}

//
// Reads from an xml stream (typically a file), extracts selected object types
// and converts them to xml.
// Steam() exposes a channel on which results are published as the
// stream is parsed
//
type StreamExtractConverter struct {
	reader        io.Reader   // the file/stream of xml to process
	dataObjects   []string    // list of data objects (e.g. StudentPersonal, SchoolInfo etc.) to extract
	resultChannel chan Result // channel to export json blobs
	sampleSize    int         // number of objects to extract, -1 for no restrictions
	attrPrefix    string      // flag attributes with this prefix
	contentToken  string      // identify element content to distinguish from attributes
	progressBar   bool        // show a progress bar for reading xml input
	streamSize    int         // size of target input stream (typically file) for use with progress bar
}

//
// Initilaises the extractor with the stream to read and the
// data objects of interest
//
func NewStreamExtractConverter(r io.Reader, opts ...Option) (*StreamExtractConverter, error) {

	// initialise default converter
	sec := &StreamExtractConverter{
		reader:        r,
		dataObjects:   []string{},
		resultChannel: make(chan Result, 256),
		progressBar:   false,
		sampleSize:    defaultSampleSize,
		attrPrefix:    defaultAttributePrefix,
		contentToken:  defaultContentToken,
	}

	// appply all options
	if err := sec.setOptions(opts...); err != nil {
		return nil, err
	}

	return sec, nil
}

//
// Invokes the parsing and converting
// of the input stream, and channel can be ranged over
// to collect json blob results
//
func (sec *StreamExtractConverter) Stream() chan Result {

	go sec.extractAndConvert()

	return sec.resultChannel
}

//
// parses the xml stream and converts the xml elements to json
//
func (sec *StreamExtractConverter) extractAndConvert() {

	defer close(sec.resultChannel)

	var uip *uiprogress.Progress
	var bar *uiprogress.Bar
	if sec.progressBar {
		uip = uiprogress.New()
		defer uip.Stop()
		uip.Start()                      // start rendering
		bar = uip.AddBar(sec.streamSize) // Add a new bar
		bar.AppendCompleted().PrependElapsed()
	}

	br := bufio.NewReaderSize(sec.reader, 65536)
	parser := xmlparser.NewXMLParser(br, sec.dataObjects...)

	count := 0
	for xml := range parser.Stream() {

		result := convertXML(xml, sec.attrPrefix, sec.contentToken)
		sec.resultChannel <- result

		if sec.progressBar {
			bar.Incr()
			bar.Set(int(parser.TotalReadSize))
		}

		count++
		if sec.sampleSize > 0 && count == sec.sampleSize {
			break
		}
	}
}

//
// convenience alias for generic json type
//
type JsonMap map[string]interface{}

//
// Converts a node supplied by the xml parser into
// a well-formed block of canonical json as byte array
// for writing to file or datastore.
//
// xml: element supplied from the parser
// returns: []byte json block
//
func convertXML(xml *xmlparser.XMLElement, attrPrefix, contentToken string) Result {

	name, data := convertNode(*xml, attrPrefix, contentToken) //defer pointer to make recursive method easier

	// var json = jsoniter.ConfigCompatibleWithStandardLibrary
	var json = jsoniter.ConfigFastest
	b, _ := json.Marshal(data)

	return Result{Name: name, Json: b}

}

//
//
// converts between the internal node structure returned
// by the xml-parser, and a 'flattened' regular json structure
// observes pesc convention of making element innertext into a
// 'value' member if a node has attributes.
//
// target: node from the xml parser
// returns:
// string: Name of the converted object
// JsonMap -> alias for classic golang generic json map[string]interface{} representation
//
//
func convertNode(target xmlparser.XMLElement, attrPrefix, contentToken string) (string, JsonMap) {

	// initialise the json strutcure for this node
	node := JsonMap{target.Name: JsonMap{}}
	// convenience reference of data map to fill under this node's main key
	jm := node[target.Name].(JsonMap)

	//
	// remove empty nodes
	// - comment out this block to have all elements from
	// original xml in output json as key:{} elements
	//
	if target.Attrs["xsi:nil"] == "true" {
		return target.Name, nil
	}
	if len(target.Attrs) == 0 && len(target.Childs) == 0 && target.InnerText == "" {
		return target.Name, nil
	}

	//
	// remove unnecessary attributes
	//
	delete(target.Attrs, "xsi:nil")
	delete(target.Attrs, "xmlns:xsd")
	delete(target.Attrs, "xmlns:xsi")
	delete(target.Attrs, "xmlns")

	//
	// handle remaining attributes
	//
	if len(target.Attrs) > 0 {
		for k, v := range target.Attrs {
			k2 := fmt.Sprintf("%s%s", attrPrefix, k)
			jm[k2] = v
		}
		if target.InnerText != "" { // attributes under PESC force innertext to be assigned to a value member
			jm[contentToken] = target.InnerText
			return target.Name, node // if there's content, we're a terminal leaf node
		}
	}

	// check if we are a terminating leaf
	if target.InnerText != "" {
		node[target.Name] = target.InnerText
		return target.Name, node
	}

	// iterate subtree
	for key, elements := range target.Childs {
		switch {
		case len(elements) > 1 || strings.HasSuffix(target.Name, "List"): // elements designated as list under PESC must still be a list even if only one item
			// handler for elements that contain arrays
			list := []JsonMap{}
			for _, e := range elements {
				_, childData := convertNode(e, attrPrefix, contentToken)
				for k, v := range childData {
					// can be k/v pair or full object
					v2, ok := v.(JsonMap)
					if ok {
						list = append(list, v2)
					} else {
						list = append(list, JsonMap{k: v})
					}
				}
			}
			if len(list) > 0 {
				jm[key] = list
			}
		default:
			// handler for individual objects
			e := elements[0]
			_, childData := convertNode(e, attrPrefix, contentToken)
			for k, v := range childData {
				jm[k] = v
			}

		}
	}

	return target.Name, node

}

type Option func(*StreamExtractConverter) error

//
// apply all supplied options to the converter
// returns any error encountered while applying the options
//
func (sec *StreamExtractConverter) setOptions(options ...Option) error {
	for _, opt := range options {
		if err := opt(sec); err != nil {
			return err
		}
	}
	return nil
}

//
// Attributes are flattened to become regular
// members of the converted json object.
// The supplied prefix will be added to the
// attribute name for visibility.
//
func AttributePrefix(prefix string) Option {
	return func(sec *StreamExtractConverter) error {
		sec.attrPrefix = prefix
		return nil
	}
}

//
// When attribtues become regular members of the converted json
// the xml's original value (innertext) needs a key. This token
// will be used for that - defaults to "value" in accordance with
// PESC json.
//
func ContentToken(token string) Option {
	return func(sec *StreamExtractConverter) error {
		sec.contentToken = token
		return nil
	}
}

//
// Working with big files it can be useful to limit output
// to a few examples, this sets how many objects will be
// output from the converter
// set size to 0 (default) for no restrictions
//
func SampleSize(size int) Option {
	return func(sec *StreamExtractConverter) error {
		sec.sampleSize = size
		return nil
	}
}

//
// Pass the size of the stream to be read in order
// to display a progress-bar
// streamSize of 0 will mean no progress-bar displayed
//
// Default is no progress bar
//
func ProgressBar(streamSize int) Option {
	return func(sec *StreamExtractConverter) error {
		if streamSize == 0 {
			sec.progressBar = false
			return nil
		}
		sec.streamSize = streamSize
		sec.progressBar = true
		return nil
	}
}

//
// You must specify the objects/xml Types to be
// extracted and converted from the stream
//
func ObjectsToExtract(dataObjects []string) Option {
	return func(sec *StreamExtractConverter) error {
		if len(dataObjects) == 0 {
			return errors.New("must specify objects to extract")
		}
		sec.dataObjects = append(sec.dataObjects, dataObjects...)
		return nil
	}
}
