package shellwords

import (
	"reflect"
	"testing"
)

var testcases = []struct {
	line     string
	expected []string
}{
	{`var --bar=baz`, []string{`var`, `--bar=baz`}},
	{`var --bar="baz"`, []string{`var`, `--bar=baz`}},
	{`var "--bar=baz"`, []string{`var`, `--bar=baz`}},
	{`var "--bar='baz'"`, []string{`var`, `--bar='baz'`}},
	{`var "--bar=\"baz'"`, []string{`var`, `--bar="baz'`}},
	{`var "--bar baz"`, []string{`var`, `--bar baz`}},
	{`var --"bar baz"`, []string{`var`, `--bar baz`}},
	{`var  --"bar baz"`, []string{`var`, `--bar baz`}},
}

func TestSimple(t *testing.T) {
	for _, testcase := range testcases {
		args, err := Parse(testcase.line)
		if err != nil {
			t.Fatalf(err.Error())
		}
		if !reflect.DeepEqual(args, testcase.expected) {
			t.Fatalf("Expected %v, but %v:", testcase.expected, args)
		}
	}
}
