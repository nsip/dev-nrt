package nrt

import (
	"bufio"
	"io"

	"strings"

	jsoniter "github.com/json-iterator/go"
	xmlparser "github.com/tamerh/xml-stream-parser"
)

//
// number of objects to extract (mostly when testing/experimenting)
// set to -1 for no restriction
//
var defaultSampleSize int = -1

//
// Reads from an xml stream (typically a file), extracts selected object types
// and converts them to xml.
// Steam() exposes a channel on which results are published as the
// stream is parsed
//
type StreamExtractConverter struct {
	reader        io.Reader   // the file/stream of xml to process
	dataObjects   []string    // list of data objects (e.g. StudentPersonal, ScholInfo etc.) to extract
	resultChannel chan []byte // channel to export json blobs
	sampleSize    int         // numer of objects to extract, -1 for no restrictions
}

//
// Initilaises the extractor with the stream to read and the
// data objects of interest
//
func NewStreamExtractConverter(r io.Reader, dataObjects ...string) *StreamExtractConverter {

	sec := &StreamExtractConverter{
		reader:        r,
		dataObjects:   []string{},
		resultChannel: make(chan []byte, 256),
		sampleSize:    defaultSampleSize,
	}
	for _, obj := range dataObjects {
		sec.dataObjects = append(sec.dataObjects, obj)
	}
	return sec
}

//
// Invokes the parsing and converting
// of the input stream, and channal can be ranged over
// to collect json blob results
//
func (sec *StreamExtractConverter) Stream() chan []byte {

	go sec.extractAndConvert(sec.sampleSize)

	return sec.resultChannel
}

//
// parses the xml stream ans converts the xml elements to json
//
func (sec *StreamExtractConverter) extractAndConvert(sampleSize int) {

	defer close(sec.resultChannel)
	br := bufio.NewReaderSize(sec.reader, 65536)

	parser := xmlparser.NewXMLParser(br, sec.dataObjects...)

	count := 0
	for xml := range parser.Stream() {

		jsonBytes := convertXML(xml)
		sec.resultChannel <- jsonBytes

		count++
		if sampleSize != -1 && count == sampleSize {
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
func convertXML(xml *xmlparser.XMLElement) []byte {

	result := convertNode(*xml) //deref pointer to make recursive method easier

	// var json = jsoniter.ConfigCompatibleWithStandardLibrary
	var json = jsoniter.ConfigFastest
	b, _ := json.Marshal(result)

	return b

}

//
//
// converts between the internal node structure returned
// by the xml-parser, and a 'flattened' regular json structure
// observes pesc convention of making element innertext into a
// 'value' member if a node has attributes.
//
// target: node from the xml parser
// returns: JsonMap -> alias for classic golang generic json map[string]interface{} representation
//
//
func convertNode(target xmlparser.XMLElement) JsonMap {

	// initialise the json strucure for this node
	node := JsonMap{target.Name: JsonMap{}}
	// convenience reference of data map to fill under this node's main key
	jm := node[target.Name].(JsonMap)

	//
	// remove empty nodes
	// - comment out this block to have all elements from
	// original xml in output json as key:{} elements
	//
	if target.Attrs["xsi:nil"] == "true" {
		return nil
	}
	if len(target.Attrs) == 0 && len(target.Childs) == 0 && target.InnerText == "" {
		return nil
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
			jm[k] = v
		}
		if target.InnerText != "" { // attribues under PESC force innertext to be assigned to a value member
			jm["value"] = target.InnerText
			return node // if thre's content, we're a terminal leaf node
		}
	}

	// check if we are a terminatng leaf
	if target.InnerText != "" {
		node[target.Name] = target.InnerText
		return node
	}

	// iterate subtree
	for key, elements := range target.Childs {
		switch {
		case len(elements) > 1 || strings.HasSuffix(target.Name, "List"):
			// handler for elements that contain arrays
			list := []JsonMap{}
			for _, e := range elements {
				for k, v := range convertNode(e) {
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
			for k, v := range convertNode(e) {
				jm[k] = v
			}

		}
	}

	return node

}
