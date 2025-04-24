package soap

import (
	"fmt"
	"io"
	"reflect"
	"strings"
)

// SOAPEnvelopeStart represents the opening part of a SOAP envelope
const SOAPEnvelopeStart = `<?xml version="1.0" encoding="utf-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
`

// SOAPEnvelopeEnd represents the closing part of a SOAP envelope
const SOAPEnvelopeEnd = `  </soap:Body>
</soap:Envelope>`

// DefaultEncoder implements the SOAPEncoder interface
type DefaultEncoder struct {
	writer io.Writer
	buffer []byte
}

// NewEncoder creates a new SOAP encoder that writes to the specified writer
func NewEncoder(w io.Writer) SOAPEncoder {
	return &DefaultEncoder{
		writer: w,
		buffer: []byte{},
	}
}

// Encode converts a Go struct to SOAP XML format and adds it to the buffer
func (e *DefaultEncoder) Encode(v interface{}) error {
	// Get the value and type of the interface
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	// Check if the struct has XMLName field
	xmlName, namespace := getXMLNameAndNamespace(val)
	if xmlName == "" {
		return fmt.Errorf("struct must have XMLName field")
	}

	// Start with the envelope
	if len(e.buffer) == 0 {
		e.buffer = append(e.buffer, []byte(SOAPEnvelopeStart)...)
	}

	// Create the root element with namespace
	rootStart := fmt.Sprintf("    <%s xmlns=\"%s\">", xmlName, namespace)
	e.buffer = append(e.buffer, []byte(rootStart)...)

	// Process all fields except XMLName
	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// Skip XMLName field
		if fieldType.Name == "XMLName" {
			continue
		}

		// Get the XML tag name
		xmlTag := fieldType.Tag.Get("xml")
		parts := strings.Split(xmlTag, ",")
		fieldName := parts[0]

		// Skip if field is empty and omitempty is specified
		if field.IsZero() && strings.Contains(xmlTag, "omitempty") {
			continue
		}

		// Format the field value
		var fieldValue string
		switch field.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			fieldValue = fmt.Sprintf("%d", field.Int())
		case reflect.String:
			fieldValue = field.String()
		default:
			fieldValue = fmt.Sprintf("%v", field.Interface())
		}

		// Add the field with empty namespace
		fieldXML := fmt.Sprintf("\n      <%s xmlns=\"\">%s</%s>", fieldName, fieldValue, fieldName)
		e.buffer = append(e.buffer, []byte(fieldXML)...)
	}

	// Close the root element
	rootEnd := fmt.Sprintf("\n    </%s>", xmlName)
	e.buffer = append(e.buffer, []byte(rootEnd)...)

	return nil
}

// Flush writes the buffered XML to the writer and clears the buffer
func (e *DefaultEncoder) Flush() error {
	if len(e.buffer) > 0 {
		// Add the envelope end
		e.buffer = append(e.buffer, []byte("\n"+SOAPEnvelopeEnd)...)

		// Write to the output
		_, err := e.writer.Write(e.buffer)
		if err != nil {
			return err
		}

		// Clear the buffer
		e.buffer = []byte{}
	}

	return nil
}

// getXMLNameAndNamespace extracts the XML element name and namespace from the XMLName field
func getXMLNameAndNamespace(val reflect.Value) (string, string) {
	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		if field.Name == "XMLName" {
			// Extract namespace and element name from the xml tag
			xmlTag := field.Tag.Get("xml")
			parts := strings.Split(xmlTag, " ")
			if len(parts) == 2 {
				return parts[1], parts[0]
			}
			return field.Name, ""
		}
	}
	return "", ""
}
