package version

import (
	"reflect"
	"slices"
	"testing"
)

func TestParse(t *testing.T) {
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

		// invalid
		{"1.1.a", Version{}, false},
		{"1.1.1-invalid_prerelease", Version{}, false},
		{"1.1.1-metadata+notsupported", Version{}, false},
		{"1.0.0-a-b_1", Version{}, false},
		{"1", Version{}, false},
		{"1.1", Version{}, false},
		{"lastest", Version{}, false},
		{"bleeding", Version{}, false},
		{"stable", Version{}, false},
	}

	for _, tc := range tests {
		v, err := New(tc.input)
		if err != nil {
			if tc.valid {
				t.Errorf("expected %v to be valid, got %v", tc.input, err)
			}
		}

		if !v.Equal(tc.expected) {
			t.Errorf("expected %v, got %v", tc.expected, v)
		}
	}
}

func TestString(t *testing.T) {
	for _, str := range []string{
		"1.1.1",
		"1.1.0-alpha",
		"1.1.0-alpha.beta",
		"1.1.0-alpha.beta.1.2.a",
		"2.2.0",
	} {
		v, err := New(str)
		if err != nil {
			t.Fatal(err)
		}

		output := v.String()
		if output != str {
			t.Errorf("expected %v, got %v", str, output)
		}
	}
}

func TestPrecedence(t *testing.T) {
	const (
		lt = -1
		eq = 0
		gt = 1
	)

	tests := []struct {
		a        Version
		expected int
		b        Version
	}{
		{Version{1, 0, 0, []string{"beta", "2"}}, lt, Version{1, 0, 0, []string{"beta", "11"}}},
		{Version{1, 0, 0, []string{"alpha"}}, lt, Version{1, 0, 0, []string{"beta"}}},
		{Version{1, 0, 0, nil}, gt, Version{1, 0, 0, []string{"beta"}}},
		{Version{2, 0, 0, nil}, gt, Version{1, 0, 0, nil}},
		{Version{2, 1, 1, nil}, gt, Version{2, 0, 1, nil}},
		{Version{2, 1, 1, nil}, gt, Version{2, 0, 1, []string{"rc.1"}}},
		{Version{2, 1, 1, nil}, eq, Version{2, 1, 1, nil}},
		{Version{2, 1, 1, nil}, eq, Version{2, 1, 1, nil}},
	}

	for _, tc := range tests {
		var ok bool
		switch tc.expected {
		case lt:
			ok = tc.a.LessThan(tc.b)
		case eq:
			ok = tc.a.Equal(tc.b)
		case gt:
			ok = tc.a.GreaterThan(tc.b)
		}

		if !ok {
			t.Fatalf("%v vs. %v (%d): expected match", tc.a, tc.b, tc.expected)
		}

		actual := Compare(tc.a, tc.b)
		if actual != tc.expected {
			t.Fatalf("%v vs. %v: expected %d, got %d", tc.a, tc.b, tc.expected, actual)
		}
	}
}

func TestOrder(t *testing.T) {
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
	}

	ordered := []Version{
		{1, 0, 0, []string{"alpha"}},
		{1, 0, 0, []string{"alpha", "1"}},
		{1, 0, 0, []string{"alpha", "beta"}},
		{1, 0, 0, []string{"alpha", "beta", "1"}},
		{1, 0, 0, []string{"alpha", "beta", "1", "a"}},
		{1, 0, 0, []string{"alpha", "beta", "1", "b"}},
		{1, 0, 0, []string{"beta"}},
		{1, 0, 0, []string{"beta", "2"}},
		{1, 0, 0, []string{"beta", "11"}},
		{1, 0, 0, []string{"beta", "222"}},
		{1, 0, 0, []string{"rc", "1"}},
		{1, 0, 0, []string{"rc", "a"}},
		{1, 0, 0, nil},
		{2, 0, 0, []string{"alpha"}},
		{2, 0, 0, nil},
	}

	sorted := slices.Clone(versions)
	slices.SortFunc(sorted, Compare)

	if !reflect.DeepEqual(sorted, ordered) {
		t.Fatalf("expected %v, got %v", ordered, sorted)
	}
}
