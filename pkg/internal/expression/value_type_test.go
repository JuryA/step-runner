package expression

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

type TestFoo struct {
	Inherited int
}

func TestObjectToProtoValue(t *testing.T) {
	cases := []struct {
		name    string
		object  any
		want    *structpb.Value
		wantErr bool
	}{{
		name:   "structpb.Value pointer",
		object: structpb.NewStringValue("test"),
		want:   structpb.NewStringValue("test"),
	}, {
		name:   "structpb.Value struct",
		object: *structpb.NewStringValue("test"),
		want:   structpb.NewStringValue("test"),
	}, {
		name: "map[string]*structpb.Value",
		object: map[string]*structpb.Value{
			"foo": structpb.NewStringValue("bar"),
			"num": structpb.NewNumberValue(42),
		},
		want: structpb.NewStructValue(&structpb.Struct{Fields: map[string]*structpb.Value{
			"foo": structpb.NewStringValue("bar"),
			"num": structpb.NewNumberValue(42),
		}}),
	}, {
		name:   "string",
		object: "hello",
		want:   structpb.NewStringValue("hello"),
	}}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := ObjectToProtoValue(c.object)
			if c.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, c.want, got)
			}
		})
	}
}

func TestDigObject(t *testing.T) {
	cases := []struct {
		object  any
		key     string
		want    any
		wantErr error
	}{{
		object:  "string",
		key:     "string-key",
		wantErr: errors.New(`"string" is not supported`),
	}, {
		object: map[string]string{"map-key": "value"},
		key:    "map-key",
		want:   "value",
	}, {
		object: map[string]int{"map-key": 32},
		key:    "map-key",
		want:   32,
	}, {
		object:  map[string]int{"map-key": 32},
		key:     "non-existing-key",
		wantErr: errors.New(`the "non-existing-key" was not found`),
	}, {
		object: struct{ StructField int }{StructField: 10},
		key:    "StructField",
		want:   10,
	}, {
		object: &struct{ StructField int }{StructField: 10},
		key:    "StructField",
		want:   10,
	}, {
		object: struct {
			Value int `json:"tag"`
		}{Value: 20},
		key:  "tag",
		want: 20,
	}, {
		object: struct {
			Value int `json:"tag,omitempty"`
		}{Value: 20},
		key:  "tag",
		want: 20,
	}, {
		object: &struct {
			Value int `json:"tag"`
		}{Value: 20},
		key:  "tag",
		want: 20,
	}, {
		object: &struct {
			TestFoo
			Value int
		}{TestFoo: TestFoo{Inherited: 30}, Value: 20},
		key:  "Inherited",
		want: 30,
	}, {
		object: structpb.NewStructValue(&structpb.Struct{Fields: map[string]*structpb.Value{
			"color":  structpb.NewStringValue("yellow"),
			"number": structpb.NewNumberValue(3),
		}}),
		key:  "color",
		want: structpb.NewStringValue("yellow"),
	}}

	for _, c := range cases {
		t.Run(c.key, func(t *testing.T) {
			got, err := DigObject(c.object, c.key)
			if c.wantErr != nil {
				require.Equal(t, c.wantErr, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, c.want, got)
			}
		})
	}
}
