package act

import (
	"encoding/json"
	"fmt"
)

type NameValues struct {
	Items []*NameValuesItem `json:"items"`
}

type NameValuesItem struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func (nv *NameValues) Set(nameValues string) error {
	if err := json.Unmarshal([]byte(nameValues), &nv.Items); err != nil {
		return fmt.Errorf("cannot unmarshal name_values JSON: %w", err)
	}

	return nil
}

func (nv *NameValues) Type() string {
	return "JSON name/values"
}

func (nv *NameValues) String() string {
	data, _ := json.Marshal(nv.Items)
	return string(data)
}

func (nv *NameValues) Values() map[string]string {
	mappy := make(map[string]string)

	for _, item := range nv.Items {
		mappy[item.Name] = item.Value
	}

	return mappy
}
