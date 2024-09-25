package schema

import (
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"github.com/stretchr/testify/require"
)

func check[T any](
	t *testing.T,
	marshal func(any) ([]byte, error),
	unmarshal func([]byte, any) error,
	data []byte,
	want T,
	schema *jsonschema.Schema,
	schemaData any,
) {

	// Unmarshal
	v := new(T)
	err := unmarshal(data, v)
	require.NoError(t, err)
	require.Equal(t, want, *v)

	// Marshal
	roundTripData, err := marshal(v)
	require.NoError(t, err)

	// Unmarshal
	roundTripV := new(T)
	err = unmarshal(roundTripData, roundTripV)
	require.NoError(t, err)
	require.Equal(t, want, *roundTripV)

	// Validate T with schema
	stepsData, err := marshal(schemaData)
	require.NoError(t, err)
	var untyped any
	err = unmarshal(stepsData, &untyped)
	require.NoError(t, err)
	err = schema.Validate(untyped)
	require.NoError(t, err)
}
