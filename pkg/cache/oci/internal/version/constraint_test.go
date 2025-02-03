package version

import (
	"reflect"
	"testing"
)

func TestConstraintParse(t *testing.T) {
	tests := []struct {
		input    string
		expected Version
		valid    bool
	}{
		// valid
		{"1.0.0-alpha", Version{1, 0, 0, []string{"alpha"}}, true},
		{"1.0.0-alpha.1", Version{1, 0, 0, []string{"alpha", "1"}}, true},
		{"1.0.0-beta.11", Version{1, 0, 0, []string{"beta", "11"}}, true},
		{"1.0.0-a.b.c.d", Version{1, 0, 0, []string{"a", "b", "c", "d"}}, true},
		{"1.0.0", Version{1, 0, 0, nil}, true},
		{"1.1.2", Version{1, 1, 2, nil}, true},
		{"1.1.2-a-b-c", Version{1, 1, 2, []string{"a-b-c"}}, true},
		{"1.1.2-a-b-c.1", Version{1, 1, 2, []string{"a-b-c", "1"}}, true},
		{"1", Version{1, Any, Any, nil}, true},
		{"1.1", Version{1, 1, Any, nil}, true},
		{"latest", Version{Latest, Any, Any, nil}, true},
		{"", Version{Any, Any, Any, nil}, true},

		// invalid
		{"1.1.a", Version{}, false},
		{"1.1.1-invalid_prerelease", Version{}, false},
		{"1.1.1-metadata+notsupported", Version{}, false},
		{"1.0.0-a-b_1", Version{}, false},
		{"1-rc.1", Version{}, false},
		{"bleeding", Version{}, false},
		{"stable", Version{}, false},
	}

	for _, tc := range tests {
		c, err := NewConstraint(tc.input)
		if err != nil {
			if tc.valid {
				t.Errorf("expected %v to be valid, got %v", tc.input, err)
			}
		}

		if !Version(c).Equal(tc.expected) {
			t.Errorf("expected %v, got %v", tc.expected, c)
		}
	}
}

func TestConstraintString(t *testing.T) {
	for _, str := range []string{
		"latest",
		"1.1.1",
		"1.1.0-alpha",
		"1.1.0-alpha.beta",
		"1.1.0-alpha.beta.1.2.a",
		"2.2.0",
		"1",
		"1.1",
		"2",
		"2.2.2",
		"",
	} {
		v, err := NewConstraint(str)
		if err != nil {
			t.Fatal(err)
		}

		output := v.String()
		if output != str {
			t.Errorf("expected %v, got %v", str, output)
		}
	}
}

func TestMatch(t *testing.T) {
	versions := []Version{
		{1, 0, 0, []string{"rc", "a"}},
		{1, 0, 0, []string{"beta", "2"}},
		{1, 0, 0, []string{"alpha", "beta", "1"}},
		{1, 0, 0, []string{"alpha"}},
		{1, 0, 0, []string{"beta", "222"}},
		{1, 0, 0, []string{"beta", "11"}},
		{1, 0, 0, []string{"alpha", "beta", "1", "a"}},
		{1, 0, 0, []string{"alpha", "beta", "1", "b"}},
		{1, 0, 0, []string{"beta"}},
		{1, 0, 0, []string{"rc", "1"}},
		{1, 0, 0, nil},
		{2, 0, 0, []string{"alpha"}},
		{2, 0, 0, nil},
		{1, 0, 0, []string{"alpha", "beta"}},
		{1, 0, 0, []string{"alpha", "1"}},
		{1, 1, 0, nil},
		{1, 1, 1, nil},
		{1, 1, 2, nil},
		{1, 100, 0, nil},
		{2, 100, 0, nil},
		{2, 100, 0, []string{"-rc.1"}},
	}

	tests := []struct {
		constraint Version
		matches    []string
	}{
		{Version{1, 0, 0, []string{"alpha"}}, []string{"1.0.0-alpha"}},
		{Version{1, Any, Any, nil}, []string{"1.0.0", "1.1.0", "1.1.1", "1.1.2", "1.100.0"}},
		{Version{2, Any, Any, nil}, []string{"2.0.0", "2.100.0"}},
		{Version{1, 1, Any, nil}, []string{"1.1.0", "1.1.1", "1.1.2"}},
		{Version{Any, Any, Any, nil}, []string{"1.0.0", "1.1.0", "1.1.1", "1.1.2", "1.100.0", "2.0.0", "2.100.0"}},
		{Version{Latest, Any, Any, nil}, []string{"2.100.0"}},
	}

	for _, tc := range tests {
		filtered := Constraint(tc.constraint).Match(versions)

		versions := make([]string, 0, len(filtered))
		for _, version := range filtered {
			versions = append(versions, version.String())
		}

		if !reflect.DeepEqual(versions, tc.matches) {
			t.Fatalf("expected %v, got %v", tc.matches, versions)
		}
	}
}
