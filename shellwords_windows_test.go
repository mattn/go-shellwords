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
		{"foo bar\\  ", []string{`foo`, `bar\`}},
		{`\\uncpath foo`, []string{`\\uncpath`, `foo`}},
		{`upx c:\github.com\jftuga\test\test.exe`, []string{`upx`, `c:\github.com\jftuga\test\test.exe`}},
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

	pwd, err := shellRun("dir", "")
	if err != nil {
		t.Fatal(err)
	}

	pwd2, err := shellRun("dir", path.Join(dir, "/_example"))
	if err != nil {
		t.Fatal(err)
	}

	if pwd == pwd2 {
		t.Fatal("`dir` should be changed")
	}
}

func TestShellRunNoEnv(t *testing.T) {
	old := os.Getenv("COMSPEC")
	defer os.Setenv("COMSPEC", old)
	os.Unsetenv("COMSPEC")

	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	pwd, err := shellRun("dir", "")
	if err != nil {
		t.Fatal(err)
	}

	pwd2, err := shellRun("dir", path.Join(dir, "/_example"))
	if err != nil {
		t.Fatal(err)
	}

	if pwd == pwd2 {
		t.Fatal("`dir` should be changed")
	}
}
