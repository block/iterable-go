package types

import (
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var input = map[string]any{
	// valid String
	"field10": "boo",
	// valid String
	// valid Date
	// * see: https://support.iterable.com/hc/en-us/articles/208183076-Field-Data-Types#date
	"field20": "2020-04-13",
	"field21": "2020-04-13 13:37:19",
	"field22": "2020-04-13 13:37:19 -04:00",
	"field23": "2020-04-13 13:37:19 -00:00",
	"field24": "2020-04-13T13:37:19.039-04:00",
	"field25": "2020-04-13T13:37:19.039Z",
	// valid String
	// invalid Date
	// * see: https://support.iterable.com/hc/en-us/articles/208183076-Field-Data-Types#common-problems-with-date-formats
	"field30": "2024-11-25T13:37:19.039",       // (missing offset)
	"field31": "2024-11-25T13:37:19.039 00:00", // (offset is missing a preceding + or -)
	"field32": "2024-11-25T13:37:19Z",          // (missing milliseconds)
	"field33": "2024/11/25T13:37:19.039Z",      // (using slashes instead of hyphens)
	"field34": "2024-11-25 13:37:19.039Z",      // (missing T separator between date and time)
	"field35": "2020-04-13 13:37:19.039",       // (missing ZZ or Z suffix)
	// valid Long
	"field40": 10,
	"field41": -10,
	"field42": time.Now().UnixNano(),
	// valid Double
	"field50": 10.6,
	"field51": -10.6,
	// valid Boolean
	"field70": true,
	"field71": false,
	// valid String
	// invalid Boolean
	"field72": "True",
	// valid Geo
	"test_geo_location": map[string]any{
		"lat": 41.505493,
		"lon": -81.681290,
	},
	"test1_geo_location": map[string]float32{
		"lat": 41.505493,
		"lon": -81.681290,
	},
	// invalid Geo
	"test2_geo_location": map[string]float32{
		"lat": 41.505493,
	},
	"test3_geo_location": map[string]any{
		"foo": "bar",
	},
	"test4_geo_location": map[string]any{},

	// valid Object
	"field80": map[string]any{
		"name":        "Snowball",
		"color":       "White",
		"age":         4,
		"favoriteToy": "Fluffy stick",
	},
	// invalid any type
	"field100": struct{}{},
	"field101": make(chan string, 10),
}

func Test_FieldType_String(t *testing.T) {
	expectTrue := []string{
		"field10",
		"field20", "field21", "field22", "field23", "field24", "field25",
		"field30", "field31", "field32", "field33", "field34", "field35",
		"field72",
	}

	for field, val := range input {
		if slices.Contains(expectTrue, field) {
			assert.True(t, FieldTypes.Is(field, val, FieldTypes.String), field)
		} else {
			assert.False(t, FieldTypes.Is(field, val, FieldTypes.String), field)
		}
	}
}

func Test_FieldType_Date(t *testing.T) {
	expectTrue := []string{
		"field20", "field21", "field22", "field23", "field24", "field25",
	}

	for field, val := range input {
		if slices.Contains(expectTrue, field) {
			assert.True(t, FieldTypes.Is(field, val, FieldTypes.Date), field)
		} else {
			assert.False(t, FieldTypes.Is(field, val, FieldTypes.Date), field)
		}
	}
}

func Test_FieldType_Long(t *testing.T) {
	expectTrue := []string{
		"field40", "field41", "field42",
	}

	for field, val := range input {
		if slices.Contains(expectTrue, field) {
			assert.True(t, FieldTypes.Is(field, val, FieldTypes.Long), field)
		} else {
			assert.False(t, FieldTypes.Is(field, val, FieldTypes.Long), field)
		}
	}
}

func Test_FieldType_Double(t *testing.T) {
	expectTrue := []string{
		"field50", "field51",
	}

	for field, val := range input {
		if slices.Contains(expectTrue, field) {
			assert.True(t, FieldTypes.Is(field, val, FieldTypes.Double), field)
		} else {
			assert.False(t, FieldTypes.Is(field, val, FieldTypes.Double), field)
		}
	}
}

func Test_FieldType_Boolean(t *testing.T) {
	expectTrue := []string{
		"field70", "field71",
	}

	for field, val := range input {
		if slices.Contains(expectTrue, field) {
			assert.True(t, FieldTypes.Is(field, val, FieldTypes.Boolean), field)
		} else {
			assert.False(t, FieldTypes.Is(field, val, FieldTypes.Boolean), field)
		}
	}
}

func Test_FieldType_Geo(t *testing.T) {
	expectTrue := []string{
		"test_geo_location", "test1_geo_location",
	}

	for field, val := range input {
		if slices.Contains(expectTrue, field) {
			assert.True(t, FieldTypes.Is(field, val, FieldTypes.Geo), field)
		} else {
			assert.False(t, FieldTypes.Is(field, val, FieldTypes.Geo), field)
		}
	}
}
