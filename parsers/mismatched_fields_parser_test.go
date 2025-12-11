package parsers

import (
	"testing"

	"github.com/block/iterable-go/types"
	"github.com/stretchr/testify/assert"
)

func TestMismatchedFieldsParamsFromResponse(t *testing.T) {
	tests := []struct {
		name     string
		response types.PostResponse
		want     types.MismatchedFieldsParams
		wantOk   bool
	}{
		{
			name: "valid response",
			response: types.PostResponse{
				Message: "Project 3549: The request does not have the same data types...",
				Code:    "RequestFieldsTypesMismatched",
				Params: map[string]any{
					"validationErrors": map[string]any{
						"my_project_etl:::soc_signup_at": map[string]any{
							"incomingTypes":  []any{"string", "keyword"},
							"expectedType":   "date",
							"category":       "user",
							"offendingValue": "2020-04-13 14:04:09.000",
							"_type":          "UnexpectedType",
						},
					},
				},
			},
			want: types.MismatchedFieldsParams{
				ValidationErrors: types.MismatchedFieldsErrors{
					"my_project_etl:::soc_signup_at": {
						IncomingTypes:  []string{"string", "keyword"},
						ExpectedType:   "date",
						Category:       "user",
						OffendingValue: "2020-04-13 14:04:09.000",
						Type:           "UnexpectedType",
					},
				},
			},
			wantOk: true,
		},
		{
			name: "empty params",
			response: types.PostResponse{
				Message: "Some message",
				Code:    "SomeCode",
				Params:  map[string]any{},
			},
			want:   types.MismatchedFieldsParams{},
			wantOk: true,
		},
		{
			name: "nil params",
			response: types.PostResponse{
				Message: "Some message",
				Code:    "SomeCode",
			},
			want:   types.MismatchedFieldsParams{},
			wantOk: true,
		},
		{
			name:     "empty response",
			response: types.PostResponse{},
			want:     types.MismatchedFieldsParams{},
			wantOk:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := MismatchedFieldsParamsFromResponse(tt.response)
			assert.Equal(t, tt.wantOk, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMismatchedFieldsParamsFromParams(t *testing.T) {
	tests := []struct {
		name   string
		params map[string]any
		want   types.MismatchedFieldsParams
		wantOk bool
	}{
		{
			name: "valid params",
			params: map[string]any{
				"validationErrors": map[string]any{
					"my_project_etl:::soc_signup_at": map[string]any{
						"incomingTypes":  []any{"string", "keyword"},
						"expectedType":   "date",
						"category":       "user",
						"offendingValue": "2020-04-13 14:04:09.000",
						"_type":          "UnexpectedType",
					},
				},
			},
			want: types.MismatchedFieldsParams{
				ValidationErrors: types.MismatchedFieldsErrors{
					"my_project_etl:::soc_signup_at": {
						IncomingTypes:  []string{"string", "keyword"},
						ExpectedType:   "date",
						Category:       "user",
						OffendingValue: "2020-04-13 14:04:09.000",
						Type:           "UnexpectedType",
					},
				},
			},
			wantOk: true,
		},
		{
			name:   "empty params",
			params: map[string]any{},
			want:   types.MismatchedFieldsParams{},
			wantOk: true,
		},
		{
			name:   "nil params",
			params: nil,
			want:   types.MismatchedFieldsParams{},
			wantOk: true,
		},
		{
			name: "invalid params with non-serializable value",
			params: map[string]any{
				"validationErrors": make(chan int),
			},
			want:   types.MismatchedFieldsParams{},
			wantOk: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := MismatchedFieldsParamsFromParams(tt.params)
			assert.Equal(t, tt.wantOk, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMismatchedFieldsParamsFromBytes(t *testing.T) {
	tests := []struct {
		name   string
		data   []byte
		want   types.MismatchedFieldsParams
		wantOk bool
	}{
		{
			name: "valid json",
			data: []byte(`{
				"validationErrors": {
					"my_project_etl:::soc_signup_at": {
						"incomingTypes": ["string", "keyword"],
						"expectedType": "date",
						"category": "user",
						"offendingValue": "2020-04-13 14:04:09.000",
						"_type": "UnexpectedType"
					}
				}
			}`),
			want: types.MismatchedFieldsParams{
				ValidationErrors: types.MismatchedFieldsErrors{
					"my_project_etl:::soc_signup_at": {
						IncomingTypes:  []string{"string", "keyword"},
						ExpectedType:   "date",
						Category:       "user",
						OffendingValue: "2020-04-13 14:04:09.000",
						Type:           "UnexpectedType",
					},
				},
			},
			wantOk: true,
		},
		{
			name:   "empty json object",
			data:   []byte(`{}`),
			want:   types.MismatchedFieldsParams{},
			wantOk: true,
		},
		{
			name:   "invalid json",
			data:   []byte(`{"invalid json"`),
			want:   types.MismatchedFieldsParams{},
			wantOk: false,
		},
		{
			name:   "null json",
			data:   []byte(`null`),
			want:   types.MismatchedFieldsParams{},
			wantOk: true,
		},
		{
			name:   "nil data",
			data:   nil,
			want:   types.MismatchedFieldsParams{},
			wantOk: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := MismatchedFieldsParamsFromBytes(tt.data)
			assert.Equal(t, tt.wantOk, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_MismatchedFields_With_FieldTypes_Usage_Example(t *testing.T) {
	data := []byte(`{
		"validationErrors": {
			"my_project_etl:::soc_signup_at": {
				"incomingTypes": ["string", "keyword"],
				"expectedType": "date",
				"category": "user",
				"offendingValue": "2020-04-13 14:04:09.000",
				"_type": "UnexpectedType"
			}
		}
	}`)

	mismatchedFieldsCache := map[string]types.FieldType{}
	params, ok := MismatchedFieldsParamsFromBytes(data)
	assert.True(t, ok)

	for fieldName, field := range params.ValidationErrors {
		fieldType := types.FieldTypes.Parse(field.ExpectedType)
		if types.FieldTypes.IsKnown(fieldType) {
			mismatchedFieldsCache[fieldName] = fieldType
		}
	}

	// Usage
	incomingMessage := map[string]any{
		"my_project_etl:::soc_signup_at": "2020-04-13 14:04:09.000",
	}
	fieldName := "my_project_etl:::soc_signup_at"
	incomingField := incomingMessage[fieldName]
	cachedFieldType := mismatchedFieldsCache[fieldName]

	isOk := types.FieldTypes.Is(fieldName, incomingField, cachedFieldType)
	// required Date, but was String
	assert.False(t, isOk)
}
