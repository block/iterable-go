package parsers

import (
	"encoding/json"

	"github.com/block/iterable-go/types"
)

func MismatchedFieldsParamsFromResponse(res types.PostResponse) (types.MismatchedFieldsParams, bool) {
	return MismatchedFieldsParamsFromParams(res.Params)
}

func MismatchedFieldsParamsFromParams(params map[string]any) (types.MismatchedFieldsParams, bool) {
	// This code adds extra work to the CPU, because it has to Unmarshal
	// the response body bytes to types.PostResponse first.
	// Then, MismatchedFieldsParamsFromParams Marshals the Params map back into json.
	// And finally, MismatchedFieldsParamsFromBytes needs to Unmarshal the json
	// into types.MismatchedFieldsParams.
	//
	// Although this is not efficient, it helps us keep the parsing logic in one place.
	// We accept this overhead over having multiple parsers (one for strings and one for bytes)
	data, err := json.Marshal(params)
	if err != nil {
		var empty types.MismatchedFieldsParams
		return empty, false
	}

	return MismatchedFieldsParamsFromBytes(data)
}

func MismatchedFieldsParamsFromBytes(data []byte) (types.MismatchedFieldsParams, bool) {
	var result types.MismatchedFieldsParams
	err := json.Unmarshal(data, &result)
	if err != nil {
		var empty types.MismatchedFieldsParams
		return empty, false
	}

	return result, true
}
