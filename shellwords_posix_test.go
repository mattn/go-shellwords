// +build !windows

package shellwords

import (
	"fmt"
	"os"
	"path"
	"reflect"
	"testing"
)

func TestEscaping(t *testing.T) {
	var testcases = []struct {
		line     string
		expected []string
	}{
		{"foo bar\\  ", []string{`foo`, `bar `}},
		{"foo 'bar\\ '", []string{`foo`, `bar\ `}},
		{`var "--bar=\"baz'"`, []string{`var`, `--bar="baz'`}},
		{`var "--bar=\'baz\'"`, []string{`var`, `--bar='baz'`}},
	}

	for i, testcase := range testcases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			args, err := Parse(testcase.line)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(args, testcase.expected) {
				t.Fatalf("Expected %#v for %q, but %#v:", testcase.expected, testcase.line, args)
			}
		})
	}
}

func TestShellRun(t *testing.T) {
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	pwd, err := shellRun("pwd", "")
	if err != nil {
		t.Fatal(err)
	}

	pwd2, err := shellRun("pwd", path.Join(dir, "/_example"))
	if err != nil {
		t.Fatal(err)
	}

	if pwd == pwd2 {
		t.Fatal("`pwd` should be changed")
	}
}

func TestShellRunNoEnv(t *testing.T) {
	old := os.Getenv("SHELL")
	defer os.Setenv("SHELL", old)
	os.Unsetenv("SHELL")

	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	pwd, err := shellRun("pwd", "")
	if err != nil {
		t.Fatal(err)
	}

	pwd2, err := shellRun("pwd", path.Join(dir, "/_example"))
	if err != nil {
		t.Fatal(err)
	}

	if pwd == pwd2 {
		t.Fatal("`pwd` should be changed")
	}
}
