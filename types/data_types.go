package types

import (
	"strings"
	"time"
)

// Data Types in Iterable-Go
//
// This package provides type validation and handling for Iterable's supported field data types.
// See: https://support.iterable.com/hc/en-us/articles/208183076-Field-Data-Types
//
// Available Types:
// - String:  Basic string type
// - Date:    Timestamps in specific ISO-8601 formats
// - Long:    Integer numbers (int, int8, int16, int32, int64)
// - Double:  Floating point numbers (float32, float64)
// - Boolean: True/false values
// - Geo:     Geographic location with lat/lon coordinates
//
// Usage:
// 1. Type Validation:
//    ```go
//    // Check if a value matches a specific type
//    isValid := types.FieldTypes.Is("fieldName", value, FieldTypes.String)
//
//    // Parse type from string name
//    fieldType := types.FieldTypes.Parse("string")
//
//    // Check if type is known
//    isKnown := types.FieldTypes.IsKnown(fieldType)
//    ```
//
// 2. Date Handling:
//    Supported date formats:
//    - "2006-01-02"                    // Date only
//    - "2006-01-02 15:04:05"           // Date + time
//    - "2006-01-02 15:04:05 -07:00"    // Date + time + timezone
//    - "2006-01-02T15:04:05.000-07:00" // Full ISO with milliseconds
//    - "2006-01-02T15:04:05.000Z"      // UTC ISO format
//
// 3. Geo Location:
//    - Field name must end with "_geo_location"
//    - Value must be a map with "lat" and "lon" float values
//    ```go
//    geo := map[string]float64{
//        "lat": 37.7749,
//        "lon": -122.4194,
//    }
//    isGeo := types.FieldTypes.Is("sf_geo_location", geo, types.FieldTypes.Geo)
//    ```
//
// Important Notes:
// - Date validation is strict and requires exact format matches
// - Fractional seconds are only allowed in ISO formats with 'T' separator
// - Geo location fields must have the "_geo_location" suffix
// - Unknown types will always return false for validation

const (
	fieldTypeNameString  = "string"
	fieldTypeNameDate    = "date"
	fieldTypeNameLong    = "long"
	fieldTypeNameDouble  = "double"
	fieldTypeNameBoolean = "Boolean"
	fieldTypeNameGeo     = "geo_location"
	fieldTypeNameObject  = "object"
	fieldTypeNameNested  = "nested"
	fieldTypeNameArray   = "array"
	fieldTypeNameUnknown = "unknown"
)

type FieldType interface {
	is(string, any) bool
	Name() string
}

type fieldTypes struct {
	String  FieldType
	Date    FieldType
	Long    FieldType
	Double  FieldType
	Boolean FieldType
	Geo     FieldType
	Unknown FieldType
}

var (
	FieldTypes = fieldTypes{
		fieldTypeString{},
		fieldTypeDate{},
		fieldTypeLong{},
		fieldTypeDouble{},
		fieldTypeBoolean{},
		fieldTypeGeo{},
		fieldTypeUnknown{},
	}
)

func (f fieldTypes) Parse(name string) FieldType {
	switch name {
	case f.String.Name():
		return f.String
	case f.Date.Name():
		return f.Date
	case f.Long.Name():
		return f.Long
	case f.Double.Name():
		return f.Double
	case f.Boolean.Name():
		return f.Boolean
	case f.Geo.Name():
		return f.Geo
	}
	return f.Unknown
}

func (f fieldTypes) IsKnown(t FieldType) bool {
	return t.Name() != f.Unknown.Name()
}

func (f fieldTypes) Is(fieldName string, fieldValue any, t FieldType) bool {
	return t.is(fieldName, fieldValue)
}

type fieldTypeString struct{}
type fieldTypeDate struct{}
type fieldTypeLong struct{}
type fieldTypeBoolean struct{}
type fieldTypeDouble struct{}
type fieldTypeGeo struct{}
type fieldTypeObject struct{}
type fieldTypeUnknown struct{}

var (
	_ FieldType = fieldTypeString{}
	_ FieldType = fieldTypeDate{}
	_ FieldType = fieldTypeLong{}
	_ FieldType = fieldTypeBoolean{}
	_ FieldType = fieldTypeDouble{}
	_ FieldType = fieldTypeGeo{}
)

func (s fieldTypeString) Name() string {
	return fieldTypeNameString
}

func (s fieldTypeDate) Name() string {
	return fieldTypeNameDate
}

func (s fieldTypeLong) Name() string {
	return fieldTypeNameLong
}

func (s fieldTypeBoolean) Name() string {
	return fieldTypeNameBoolean
}

func (s fieldTypeDouble) Name() string {
	return fieldTypeNameDouble
}

func (s fieldTypeGeo) Name() string {
	return fieldTypeNameGeo
}

func (s fieldTypeUnknown) Name() string {
	return fieldTypeNameUnknown
}

func (s fieldTypeString) is(_ string, val any) bool {
	if _, ok := val.(string); ok {
		return true
	}

	return false
}

const (
	dateLayout1 = "2006-01-02"
	dateLayout2 = "2006-01-02 15:04:05"
	dateLayout3 = "2006-01-02 15:04:05 -07:00"
	dateLayout4 = "2006-01-02T15:04:05.000-07:00"
	dateLayout5 = "2006-01-02T15:04:05.000Z"
)

var (
	dateLayoutsNoFractionalSec   = []string{dateLayout1, dateLayout2, dateLayout3}
	dateLayoutsWithFractionalSec = []string{dateLayout4, dateLayout5}
)

func (s fieldTypeDate) is(_ string, val any) bool {
	if str, ok := val.(string); ok {
		// Iterable uses very specific date formats, described here:
		// https://support.iterable.com/hc/en-us/articles/208183076-Field-Data-Types#date
		//
		// Supported formats suppose to follow ISO-8601 standard.
		// However, Golang's implementation of the standard (time.RFC3339),
		// doesn't really fit into what's expected by the Iterable API.
		//
		// Hence, we need to loop over custom date formats to make sure
		// they match exactly with what's expected.
		//
		// Additionally, there's no way to enforce the use of milliseconds in Golang.
		// For example, the following date/time layout in Golang:
		//     "2006-01-02 15:04:05"
		// allows strings like this:
		//     "2020-04-13 13:37:19.039"
		// However, this breaks on the server side, because
		// ".039" is not expected in this case.
		// Hence, to cover all use-cases of how the Date type is
		// formatted on the server side, we need to split date/time
		// format layouts into 2 categories:
		//     * with "."
		//     * without "."
		var formats []string
		if strings.Contains(str, ".") {
			formats = dateLayoutsWithFractionalSec
		} else {
			formats = dateLayoutsNoFractionalSec
		}

		for _, f := range formats {
			_, err := time.Parse(f, str)
			if err == nil {
				return true
			}
		}
	}

	return false
}

func (s fieldTypeLong) is(_ string, val any) bool {
	switch val.(type) {
	case int, int8, int16, int32, int64:
		return true
	}
	return false
}

func (s fieldTypeDouble) is(_ string, val any) bool {
	switch val.(type) {
	case float32, float64:
		return true
	}
	return false
}

func (s fieldTypeBoolean) is(_ string, val any) bool {
	if _, ok := val.(bool); ok {
		return true
	}

	return false
}

func (s fieldTypeGeo) is(fieldName string, val any) bool {
	if !strings.HasSuffix(fieldName, "_geo_location") {
		return false
	}
	var m map[string]any
	switch mm := val.(type) {
	case map[string]any:
		m = mm
	case map[string]float32:
		m = map[string]any{}
		lat, ok := mm["lat"]
		if ok {
			m["lat"] = lat
		}
		lon, ok := mm["lon"]
		if ok {
			m["lon"] = lon
		}
	case map[string]float64:
		m = map[string]any{}
		lat, ok := mm["lat"]
		if ok {
			m["lat"] = lat
		}
		lon, ok := mm["lon"]
		if ok {
			m["lon"] = lon
		}
	default:
		return false
	}

	lat, ok := m["lat"]
	if !ok {
		return false
	}
	switch lat.(type) {
	case float32, float64:
	default:
		return false
	}

	lon, ok := m["lon"]
	if !ok {
		return false
	}
	switch lon.(type) {
	case float32, float64:
	default:
		return false
	}

	return true
}

func (s fieldTypeObject) is(_ string, val any) bool {
	_, ok := val.(map[string]any)
	return ok
}

func (s fieldTypeUnknown) is(_ string, _ any) bool {
	return false
}
